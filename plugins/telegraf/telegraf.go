package telegraf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/amonapp/amonagent/plugins"
	"github.com/gonuts/go-shellquote"
	"github.com/mitchellh/mapstructure"
)

// Telegraf - XXX
type Telegraf struct {
	Config Config
}

// Description - XXX
func (t *Telegraf) Description() string {
	return "Collects data from Telegraf"
}

var sampleConfig = `
#   Available config options:
#
#     {"config": "/etc/opt/telegraf/telegraf.conf"}
#
#
# Config location: /etc/opt/amonagent/plugins-enabled/telegraf.conf
`

// SampleConfig - XXX
func (t *Telegraf) SampleConfig() string {
	return sampleConfig
}

// Config - XXX
type Config struct {
	Config string `mapstructure:"config"`
}

func (m Metric) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}

// Metric - XXX
type Metric struct {
	Plugin string `json:"plugin"`
	Gauge  string `json:"gauge"`
	Value  string `json:"value"`
}

// ParsedLine - XXX
type ParsedLine struct {
	Elements []Metric
}

// SetConfigDefaults - XXX
func (t *Telegraf) SetConfigDefaults(configPath string) error {
	c, err := plugins.ReadConfigPath(configPath)
	if err != nil {
		fmt.Printf("Can't read config file: %s %v\n", configPath, err)
	}
	var config Config
	decodeError := mapstructure.Decode(c, &config)
	if decodeError != nil {
		fmt.Print("Can't decode config file", decodeError.Error())
	}

	t.Config = config

	return nil
}

// ParseLine - XXX
func ParseLine(s string) (ParsedLine, error) {
	line := ParsedLine{}
	// split by space
	space := func(c rune) bool {
		return c == ' '
	}

	// split by =,
	eq := func(c rune) bool {
		return c == '='
	}

	// split by ,
	comma := func(c rune) bool {
		return c == ','
	}

	//split metric name by _
	underscore := func(c rune) bool {
		return c == '_'
	}

	measurementLine := strings.FieldsFunc(s, space)
	// line := ParsedLine{}
	// skip non-essential information like * Plugin: name
	if len(measurementLine) > 0 {

		lineStarter := measurementLine[0]
		// > ping,url=www.google.com average_response_ms=2.596,packets_received=1i 1454321712994367057
		if lineStarter == ">" {

			if len(measurementLine) == 4 {
				// ping,url=www.google.com
				pluginMeta := strings.FieldsFunc(measurementLine[1], comma)
				if len(pluginMeta) > 1 {
					chartName := strings.Join(pluginMeta[1:], "|") // url=google.com
					chartName = strings.Replace(chartName, ".", "", -1)
					chartName = strings.Replace(chartName, "=", ":", -1)

					metricValue := strings.FieldsFunc(measurementLine[2], comma)
					for _, v := range metricValue {
						m := Metric{}
						// inodes_used=0i
						// total=0i

						metric := strings.FieldsFunc(v, eq)
						if len(metric) == 2 {
							var value string
							toFloat, err := strconv.ParseFloat(metric[1], 64)
							if err != nil {
								value = strings.Replace(metric[1], "i", "", -1)
							} else {
								value = strconv.FormatFloat(toFloat, 'f', -1, 64)
							}

							splitOnUnderscore := strings.FieldsFunc(metric[0], underscore)

							var cleanName string

							if len(splitOnUnderscore) > 2 {
								cleanName = strings.Join(splitOnUnderscore[0:], ".")
							} else {

								cleanName = strings.Join(splitOnUnderscore[:], ".")
							}

							m.Plugin = "telegraf." + pluginMeta[0] // ping
							m.Gauge = chartName + "_" + cleanName
							m.Value = value

							line.Elements = append(line.Elements, m)

						}

					}

				}

			}
		}

	}

	return line, nil

}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

// Run - XXX
func Run(command string) (string, error) {
	splitCmd, err := shellquote.Split(command)
	if err != nil || len(splitCmd) == 0 {
		return "", fmt.Errorf("exec: unable to parse command, %s", err)
	}

	cmd := exec.Command(splitCmd[0], splitCmd[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("exec: %s for command '%s'", err, command)
	}

	return out.String(), nil
}

// Collect - XXX
func (t *Telegraf) Collect(configPath string) (interface{}, error) {
	t.SetConfigDefaults(configPath)
	Command := fmt.Sprintf("/usr/bin/telegraf -test -config %s", t.Config.Config)
	Commandresult, err := Run(Command)
	if err != nil {
		fmt.Println("Can't execute command: ", err)
	}

	plugins := make(map[string]interface{})
	lines := strings.Split(Commandresult, "\n")
	var result []Metric
	for _, line := range lines {
		metrics, _ := ParseLine(line)

		if len(metrics.Elements) > 0 {
			for _, m := range metrics.Elements {
				if len(m.Gauge) > 0 {
					result = append(result, m)
				}
			}
		}

	}
	// Filter unique plugins
	AllPlugins := []string{}
	for _, r := range result {
		if !contains(AllPlugins, r.Plugin) {
			AllPlugins = append(AllPlugins, r.Plugin)
		}
	}
	for _, p := range AllPlugins {
		plugins[p] = make(map[string]interface{})
		GaugesWrapper := make(map[string]interface{})
		gauges := make(map[string]interface{})
		for _, r := range result {

			if r.Plugin == p {
				gauges[r.Gauge] = r.Value
			}

		}

		GaugesWrapper["gauges"] = gauges

		// if len(plugin) > 0 {
		// 	plugins[p] = GaugesWrapper
		// }

		plugins[p] = GaugesWrapper

	}

	// s, _ := json.Marshal(plugins)
	// fmt.Println(string(s))

	return plugins, nil
}

func init() {
	plugins.Add("telegraf", func() plugins.Plugin {
		return &Telegraf{}
	})
}
