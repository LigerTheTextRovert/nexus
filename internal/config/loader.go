// Package config, all the config loading is handled here.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type HTTPMethod = string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
)

type Backend struct {
	URL string `yaml:"url"`
}

type RateLimit struct {
	Requests int    `yaml:"requests"`
	Per      string `yaml:"per"`
}

type Route struct {
	Path        string       `yaml:"path"`
	Methods     []HTTPMethod `yaml:"methods"`
	BackendURL  []Backend    `yaml:"backends"`
	RateLimit   *RateLimit   `yaml:"rate_limit"`
	StripPrefix bool         `yaml:"strip_prefix"`
}

type Config struct {
	Routes []Route `yaml:"routes"`
	Port   int     `yaml:"port"`
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
