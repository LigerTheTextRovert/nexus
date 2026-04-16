// Package main
package main

import (
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
	cfg.Validate()

	server := server.New(&cfg)
	return server.Start()
}
