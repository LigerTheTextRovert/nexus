package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BackendURL string `yaml:"backendURL"`
	Port       string `yaml:"port"`
}

func LoadConfig(configPath string, c *Config) (*Config, error) {
	yamlConfig, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	err = yaml.Unmarshal(yamlConfig, c)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	return c, nil
}
