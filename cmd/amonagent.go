package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/amonapp/amonagent"
	"github.com/amonapp/amonagent/collectors"
	"github.com/amonapp/amonagent/logging"
	"github.com/amonapp/amonagent/plugins"

	_ "github.com/amonapp/amonagent/plugins/all"
	"github.com/amonapp/amonagent/settings"
)

var agentLogger = logging.GetLogger("amonagent.main")

var fTest = flag.Bool("test", false, "gather all metrics, print them out, and exit")
var fDebug = flag.Bool("debug", false, "Starts the agent and displays the metrics sent in the terminal")
var fListPlugins = flag.Bool("list-plugins", false, "lists all available plugins and exit")
var fTestPlugin = flag.String("test-plugin", "", "gather plugin metrics, print them out, and exit")
var fPluginConfig = flag.String("plugin-config", "", "Shows the example config for a plugin")
var fVersion = flag.Bool("version", false, "display the version")
var fPidfile = flag.String("pidfile", "", "file to write our pid to")
var fMachineID = flag.Bool("machineid", false, "Get or Create unique machine id, this value is used to identify hosts")

// Amonagent version
//	-ldflags "-X main.Version=`git describe --always --tags`"

// Version - XXX
var Version string

// ListPlugins -- XXX
func ListPlugins() {
	allPlugins := plugins.Plugins
	fmt.Println("\033[92m \nAvailable plugins: \033[0m")
	for r := range allPlugins {
		fmt.Println(r)
	}
}

// Debug - XXX
func Debug() {

	creator, ok := plugins.ServicePlugins["statsd"]
	if ok {
		statsd := creator()
		pluginConfig, _ := plugins.GetConfigPath("statsd")

		statsd.Start(pluginConfig.Path)

		// statsd.Collect(pluginConfig.Path)

		// statsd.Stop()

	}

}

func main() {
	// Debug()
	// return
	flag.Parse()

	machineID := collectors.GetOrCreateMachineID()

	if *fListPlugins {
		ListPlugins()
		return
	}

	if len(*fPluginConfig) > 0 {
		pluginConfig, _ := plugins.GetConfigPath(*fPluginConfig)
		creator, ok := plugins.Plugins[pluginConfig.Name]
		if ok {
			plugin := creator()
			conf := plugin.SampleConfig()
			fmt.Println(conf)
		} else {
			fmt.Printf("Non existing plugin: %s", pluginConfig.Name)
			ListPlugins()
		}
		return
	}

	if len(*fTestPlugin) > 0 {
		pluginConfig, err := plugins.GetConfigPath(*fTestPlugin)

		if err != nil {
			fmt.Printf("Can't get config file for plugin: %s", err)
		}

		creator, ok := plugins.Plugins[pluginConfig.Name]
		if ok {
			plugin := creator()
			start := time.Now()
			PluginResult, err := plugin.Collect(pluginConfig.Path)
			if err != nil {
				fmt.Printf("Can't get stats for plugin: %s", err)
			}
			fmt.Println(PluginResult)

			elapsed := time.Since(start)
			fmt.Printf("\n Executed in %s", elapsed)
		} else {
			fmt.Printf("\033[91mNon existing plugin: %s or missing config file in /etc/opt/amonagent/plugins-enabled/%s.conf \033[0m", *fTestPlugin, *fTestPlugin)
			ListPlugins()
		}
		return
		// [0;31m
	}

	if *fVersion {
		v := fmt.Sprintf("Amon - Version %s", Version)
		fmt.Println(v)
		return
	}

	config := settings.Settings()

	serverKey := config.ServerKey

	ag, err := amonagent.NewAgent(config)
	if err != nil {
		log.Fatal(err)
	}

	if *fTest {
		err = ag.Test(config)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if *fMachineID {
		fmt.Print(machineID)
		return
	}

	if len(machineID) == 0 && len(serverKey) == 0 {
		log.Fatal("Can't detect Machine ID. Please define `server_key` in /etc/opt/amonagent/amonagent.conf ")
	}

	shutdown := make(chan struct{})
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt)
	go func() {
		<-signals
		close(shutdown)
	}()

	log.Printf("Starting Amon Agent (version %s)\n", Version)

	if *fPidfile != "" {
		// Ensure the required directory structure exists.
		err := os.MkdirAll(filepath.Dir(*fPidfile), 0700)
		if err != nil {
			log.Fatal(3, "Failed to verify pid directory", err)
		}

		f, err := os.Create(*fPidfile)
		if err != nil {
			log.Fatalf("Unable to create pidfile: %s", err)
		}

		fmt.Fprintf(f, "%d\n", os.Getpid())

		f.Close()
	}

	ag.Run(shutdown, *fDebug)
}
