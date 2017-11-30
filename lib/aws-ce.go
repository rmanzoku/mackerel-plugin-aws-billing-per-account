package mpawsce

import (
	"flag"

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
}

// FetchMetrics interface for mackerelplugin
func (c CEPlugin) FetchMetrics() (map[string]float64, error) {
	ret := make(map[string]float64)

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

	helper := mp.NewMackerelPlugin(ce)
	helper.Tempfile = *optTempfile
	helper.Run()
}
