// Package main
package main

import (
	"fmt"
	"log"

	"github.com/LigerTheTextRovert/nexus/internal/config"
	"github.com/LigerTheTextRovert/nexus/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var cfg config.Config

	config.LoadConfig("configs/config.yml", &cfg)

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to load config file: %v", err)
	}

	server := server.New(&cfg)
	return server.Start()
}
