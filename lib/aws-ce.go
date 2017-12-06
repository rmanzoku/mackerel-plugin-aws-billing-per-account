package mpawsce

import (
	"flag"
	"fmt"
	"log"

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
		Label: "billing",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "total", Label: "Pending", Diff: false, Stacked: true},
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

func (c CEPlugin) createConnection() (*costexplorer.CostExplorer, error) {
	var creds = credentials.NewSharedCredentials("", "default")
	conn := costexplorer.New(session.New(&aws.Config{Credentials: creds, Region: &c.Region}))
	return conn, nil
}

// FetchMetrics interface for mackerelplugin
func (c CEPlugin) FetchMetrics() (map[string]float64, error) {
	ret := make(map[string]float64)

	fmt.Println(c.CostExplorer.GetCostAndUsage(&costexplorer.GetCostAndUsageInput{
		Granularity: aws.String("MONTHLY"),
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String("2017-11-01"),
			End:   aws.String("2017-12-01"),
		},
		Metrics: []*string{
			aws.String("UnblendedCost"),
		},
		GroupBy: []*costexplorer.GroupDefinition{
			&costexplorer.GroupDefinition{
				Type: aws.String("DIMENSION"),
				Key:  aws.String("LINKED_ACCOUNT"),
			},
		},
	}))

	// gd1 := costexplorer.GroupDefinition{
	// 	Type: aws.String("DIMENSION"),
	// 	Key:  aws.String("LINKED_ACCOUNT"),
	// }

	// gb := []*costexplorer.GroupDefinition{}
	// gb = append(gb, &gd1)

	// input := costexplorer.GetCostAndUsageInput{
	// 	GroupBy: gb,
	// }
	// fmt.Println(c.CostExplorer)
	// input := costexplorer.GetCostAndUsageInput{}

	// result, err := c.CostExplorer.GetCostAndUsage(&input)
	// if err != nil {
	// 	return ret, err
	// }

	// fmt.Println(result)
	ret["billing.hogehoge.total"] = 1.0
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
	ce.CostExplorer, err = ce.createConnection()
	if err != nil {
		log.Fatalln(err)
	}

	helper := mp.NewMackerelPlugin(ce)
	helper.Tempfile = *optTempfile
	helper.Run()
}
