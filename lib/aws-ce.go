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
	"billing.#": {
		Label: "AWS Monthly Billing",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "BlendedCost", Label: "Blended Cost", Diff: false, Stacked: true},
			{Name: "UnblendedCost", Label: "Unblended Cost", Diff: false, Stacked: true},
			{Name: "UsageQuantity", Label: "Usage Quantity", Diff: false, Stacked: true},
		},
	},
}

// CEPlugin mackerel plugin for Cost Explorer
type CEPlugin struct {
	Prefix          string
	Metrics         string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	CostExplorer    *costexplorer.CostExplorer
}

func (c *CEPlugin) createConnection() error {
	var creds = credentials.NewSharedCredentials("", "default")
	c.CostExplorer = costexplorer.New(session.New(&aws.Config{Credentials: creds, Region: &c.Region}))
	return nil
}

// FetchMetrics interface for mackerelplugin
func (c CEPlugin) FetchMetrics() (map[string]float64, error) {

	ret := make(map[string]float64)

	now := time.Now().UTC()
	start := fmt.Sprintf("%d-%d-01", now.Year(), now.Month())
	end := fmt.Sprintf("%d-%d-%02d", now.Year(), now.Month(), now.Day())

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
	for _, v := range dimentionValues.DimensionValues {
		name := *v.Attributes["description"]
		// Mackerel allows /[-a-zA-Z0-9_]/ for name
		name = strings.Replace(name, ".", "", -1)
		name = strings.Replace(name, ",", "", -1)
		name = strings.Replace(name, " ", "-", -1)

		accounts[*v.Value] = name
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
		ret["billing."+accounts[*g.Keys[0]]+"."+c.Metrics], err = strconv.ParseFloat(*g.Metrics[c.Metrics].Amount, 64)
		if err != nil {
			return ret, err
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
		optPrefix          = flag.String("metric-key-prefix", "aws-ce", "Metric key prefix")
		optMetrics         = flag.String("metrics", "UnblendedCost", "Choise from [BlendedCost, UnblendedCost, UsageQuantity]")
		optAccessKeyID     = flag.String("access-key-id", "", "AWS Access Key ID")
		optSecretAccessKey = flag.String("secret-access-key", "", "AWS Secret Access Key")
		optTempfile        = flag.String("tempfile", "", "Temp file name")
	)
	flag.Parse()

	var ce CEPlugin

	ce.Prefix = *optPrefix
	ce.Metrics = *optMetrics
	ce.AccessKeyID = *optAccessKeyID
	ce.SecretAccessKey = *optSecretAccessKey
	ce.Region = region

	var err error
	err = ce.createConnection()
	if err != nil {
		log.Fatalln(err)
	}

	helper := mp.NewMackerelPlugin(ce)
	helper.Tempfile = *optTempfile
	helper.Run()
}
