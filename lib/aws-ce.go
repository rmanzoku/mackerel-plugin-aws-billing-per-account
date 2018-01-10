package mpawsce

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	mp "github.com/mackerelio/go-mackerel-plugin"
)

const (
	namespace = "AWS/CE"
	region    = "us-east-1"
)

var graphdef = map[string]mp.Graphs{
	"usage.blended_cost": {
		Label: "AWS Monthly Cost Blended",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "#", Diff: false, Stacked: true},
		},
	},
	"usage.unblended_cost": {
		Label: "AWS Monthly Cost Unblended",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "#", Diff: false, Stacked: true},
		},
	},
	"usage.usage_quantity": {
		Label: "AWS Monthly Usage Quantity",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "#", Diff: false, Stacked: true},
		},
	},
	"forecast.blended_cost": {
		Label: "Forecast AWS Monthly Cost Blended",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "#", Diff: false, Stacked: true},
		},
	},
	"forecast.unblended_cost": {
		Label: "Forecast AWS Monthly Cost Unblended",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "#", Diff: false, Stacked: true},
		},
	},
	"forecast.usage_quantity": {
		Label: "Forecast AWS Monthly Usage Quantity",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "#", Diff: false, Stacked: true},
		},
	},
}

// CEPlugin mackerel plugin for Cost Explorer
type CEPlugin struct {
	Prefix          string
	Metrics         string
	DisableName     bool
	EnableForecast  bool
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	CostExplorer    *costexplorer.CostExplorer
}

func convertMetricsName(in string) string {
	if in == "BlendedCost" {
		return "blended_cost"
	}

	if in == "UnblendedCost" {
		return "unblended_cost"
	}

	if in == "UsageQuantity" {
		return "usage_quantity"
	}

	return "unknown"
}

func (c *CEPlugin) prepare() error {

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	config := aws.NewConfig()
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		config = config.WithCredentials(credentials.NewStaticCredentials(c.AccessKeyID, c.SecretAccessKey, ""))
	}
	config = config.WithRegion(c.Region)

	c.CostExplorer = costexplorer.New(sess, config)
	return nil
}

// FetchMetrics interface for mackerelplugin
func (c CEPlugin) FetchMetrics() (map[string]float64, error) {

	ret := make(map[string]float64)

	now := time.Now().UTC()
	firstday := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastday := firstday.AddDate(0, 1, 0)

	start := fmt.Sprintf("%d-%02d-%02d", firstday.Year(), firstday.Month(), firstday.Day())
	end := fmt.Sprintf("%d-%02d-%02d", now.Year(), now.Month(), now.Day())

	elasp := float64(now.Unix() - firstday.Unix())
	period := float64(lastday.Unix() - firstday.Unix())

	dimentionValues, err := c.CostExplorer.GetDimensionValues(&costexplorer.GetDimensionValuesInput{
		Dimension: aws.String("LINKED_ACCOUNT"),
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(start),
			End:   aws.String(end),
		},
	})

	if err != nil {
		return ret, err
	}

	accounts := make(map[string]string)
	if !c.DisableName {
		for _, v := range dimentionValues.DimensionValues {
			name := *v.Attributes["description"]
			// Mackerel allows /[-a-zA-Z0-9_]/ for name
			name = strings.Replace(name, ".", "", -1)
			name = strings.Replace(name, ",", "", -1)
			name = strings.Replace(name, " ", "-", -1)

			accounts[*v.Value] = name
		}
	}

	costAndUsage, err := c.CostExplorer.GetCostAndUsage(&costexplorer.GetCostAndUsageInput{
		Granularity: aws.String("MONTHLY"),
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(start),
			End:   aws.String(end),
		},
		Metrics: []*string{
			aws.String(c.Metrics),
		},
		GroupBy: []*costexplorer.GroupDefinition{
			&costexplorer.GroupDefinition{
				Type: aws.String("DIMENSION"),
				Key:  aws.String("LINKED_ACCOUNT"),
			},
		},
	})

	if err != nil {
		return ret, err
	}

	for _, g := range costAndUsage.ResultsByTime[0].Groups {
		key := *g.Keys[0]
		if !c.DisableName {
			key = accounts[*g.Keys[0]]
		}

		usage, err := strconv.ParseFloat(*g.Metrics[c.Metrics].Amount, 64)
		if err != nil {
			return ret, err
		}

		ret["usage."+convertMetricsName(c.Metrics)+"."+key] = usage

		if c.EnableForecast {
			ret["forecast."+convertMetricsName(c.Metrics)+"."+key] = usage * period / elasp
		}

	}

	return ret, nil

}

// GraphDefinition interface for mackerelplugin
func (c CEPlugin) GraphDefinition() map[string]mp.Graphs {
	graphdef := graphdef
	return graphdef
}

// MetricKeyPrefix interface for PluginWithPrefix
func (c CEPlugin) MetricKeyPrefix() string {
	if c.Prefix == "" {
		c.Prefix = "aws-ce"
	}
	return c.Prefix
}

// Do the plugin
func Do() {
	var (
		optPrefix          = flag.String("metric-key-prefix", "aws-ce", "Metric key prefix.")
		optMetrics         = flag.String("metrics", "UnblendedCost", "Choise from [BlendedCost, UnblendedCost, UsageQuantity].")
		optDisableName     = flag.Bool("disable-name", false, "Disable to get account name. Output account ID.")
		optEnableForecast  = flag.Bool("enable-forecast", false, "Enable to forecast billing.")
		optAccessKeyID     = flag.String("access-key-id", "", "AWS Access Key ID")
		optSecretAccessKey = flag.String("secret-access-key", "", "AWS Secret Access Key")
		optTempfile        = flag.String("tempfile", "", "Temp file name")
	)
	flag.Parse()

	var ce CEPlugin

	ce.Prefix = *optPrefix
	ce.Metrics = *optMetrics
	ce.DisableName = *optDisableName
	ce.EnableForecast = *optEnableForecast
	ce.AccessKeyID = *optAccessKeyID
	ce.SecretAccessKey = *optSecretAccessKey
	ce.Region = region

	var err error
	err = ce.prepare()
	if err != nil {
		log.Fatalln(err)
	}

	helper := mp.NewMackerelPlugin(ce)
	helper.Tempfile = *optTempfile
	helper.Run()
}
