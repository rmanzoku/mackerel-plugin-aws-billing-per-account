package mpawsce

import (
	"flag"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	mp "github.com/mackerelio/go-mackerel-plugin"
)

const (
	namespace = "AWS/CE"
	region    = "us-east-1"
	metrics   = "UnblendedCost"
)

var graphdef = map[string]mp.Graphs{
	"billing.#": {
		Label: "billing",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: metrics, Label: metrics, Diff: false, Stacked: true},
		},
	},
}

// CEPlugin mackerel plugin for Cost Explorer
type CEPlugin struct {
	Prefix          string
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

	start := "2017-11-01"
	end := "2017-12-01"

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
			aws.String(metrics),
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
		ret["billing."+accounts[*g.Keys[0]]+"."+metrics], err = strconv.ParseFloat(*g.Metrics[metrics].Amount, 64)
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
		optAccessKeyID     = flag.String("access-key-id", "", "AWS Access Key ID")
		optSecretAccessKey = flag.String("secret-access-key", "", "AWS Secret Access Key")
		optTempfile        = flag.String("tempfile", "", "Temp file name")
	)
	flag.Parse()

	var ce CEPlugin

	ce.Prefix = *optPrefix
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
