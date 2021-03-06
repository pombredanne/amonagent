package plugins

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/amonapp/amonagent/settings"
)

// PluginConfig - XXX
type PluginConfig struct {
	Path string
	Name string
}

// PluginConfigPath - XXX
var PluginConfigPath = path.Join(settings.ConfigPath, "plugins-enabled")

// ReadConfigPath - Works only with flat config files, do something different for nested configs
func ReadConfigPath(path string) (interface{}, error) {
	var data map[string]interface{}
	file, e := ioutil.ReadFile(path)
	if e != nil {
		return data, e
	}

	if err := json.Unmarshal(file, &data); err != nil {
		return data, err
	}

	return data, nil

}

// GetConfigPath - XXX
func GetConfigPath(plugin string) (PluginConfig, error) {
	config := PluginConfig{}

	// On Linux /etc/opt/amonagent/plugins-enabled/plugin.conf
	var pluginPath = path.Join(PluginConfigPath, strings.Join([]string{plugin, "conf"}, "."))
	config.Path = pluginPath
	config.Name = plugin

	return config, nil
}

// GetAllEnabledPlugins - XXX
func GetAllEnabledPlugins() ([]PluginConfig, error) {
	fileList := []PluginConfig{}

	if _, err := os.Stat(PluginConfigPath); os.IsNotExist(err) {
		if err != nil {
			if os.IsNotExist(err) {
				// Plugin config directory doesn't exist for some reason. Create
				PluginDirErr := os.MkdirAll(PluginConfigPath, os.ModePerm)

				if PluginDirErr != nil {
					fmt.Printf("Plugin directory doesn't exist: %s\n", PluginConfigPath)
				}

			}
			return fileList, err
		}

	}

	filepath.Walk(PluginConfigPath, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			// Only files ending with .conf
			fileName := strings.Split(f.Name(), ".conf")
			if len(fileName) == 2 {

				// Go over the list of available plugins and add the config only if it is for an existing plugin
				for name := range Plugins {
					if name == fileName[0] {
						f := PluginConfig{Path: path, Name: fileName[0]}
						fileList = append(fileList, f)
					}

				}

			}

		}
		return nil
	})

	return fileList, nil
}

// Plugin - XXX
type Plugin interface {
	// Description returns a one-sentence description on the Plugin
	Description() string

	SampleConfig() string

	// Collects all the metrics and returns a struct with the results
	Collect(string) (interface{}, error)
}

// ServicePlugin - XXX
type ServicePlugin interface {
	// SampleConfig returns the default configuration of the Plugin
	SampleConfig() string

	// Description returns a one-sentence description on the Plugin
	Description() string

	// Collects all the metrics and returns a struct with the results
	Collect(string) (interface{}, error)

	// Start starts the service
	Start(string) error

	// Stop stops the services and closes any necessary channels and connections
	Stop()
}

// PluginRegistry - XXX
type PluginRegistry func() Plugin

// Plugins - XXX
var Plugins = map[string]PluginRegistry{}

// Add - XXX
func Add(name string, registry PluginRegistry) {
	Plugins[name] = registry
}

// ServicePluginRegistry - XXX
type ServicePluginRegistry func() ServicePlugin

// Plugins - XXX
var ServicePlugins = map[string]ServicePluginRegistry{}

// AddService - XXX
func AddService(name string, registry ServicePluginRegistry) {
	ServicePlugins[name] = registry
}
