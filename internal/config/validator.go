package config

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

// Validate port is set and valid.
// Validate each route has non-empty path.
// Enforce normalized path format (starts with /, no trailing slash except /).
// Validate backend_URL is parseable and has http/https scheme + host.
// Detect duplicate route paths.
// Return clear, actionable startup errors.

func (c *Config) PortValidator() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("your port number should be between 1 and 65535")
	}
	return nil
}

func (c *Config) DeduplicateRoutes() {
	seen := make(map[string]bool)
	var unique []Route
	var duplicates []string

	for _, v := range c.Routes {
		if seen[v.Path] {
			duplicates = append(duplicates, v.Path)
		} else {
			seen[v.Path] = true
			unique = append(unique, v)
		}
	}

	if len(duplicates) > 0 {
		log.Printf("warning: duplicated paths will be ignored, duplicated paths: %v\n", duplicates)
	}

	c.Routes = unique
}

func (c *Config) PathValidator(path string) error {
	//first deduplicate paths
	c.DeduplicateRoutes()

	if path == "" {
		return fmt.Errorf("you can not use empty path")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("your path must start with /")
	}
	return nil
}

func (c *Config) BackendURLValidator(backendURL string) error {
	u, err := url.ParseRequestURI(backendURL)
	if err != nil {
		return fmt.Errorf("please enter a valid backend_URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("error: backend_URL must use http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("error: backend_URL must include a host")
	}
	return nil
}
