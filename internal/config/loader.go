// Package config, all the config loading is handled here.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Route struct {
	Path        string `yaml:"path"`
	BackendURL  string `yaml:"backend_URL"`
	StripPrefix bool   `yaml:"strip_prefix"`
}

type Config struct {
	Routes []Route `yaml:"routes"`
	Port   int
}

func LoadConfig(configPath string, c *Config) error {
	yamlConfig, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("an error occurs during reading the config file: %w", err)
	}

	err = yaml.Unmarshal(yamlConfig, c)
	if err != nil {
		return fmt.Errorf("an error occurs while unmarshaling the config file: %w", err)
	}

	return nil
}
