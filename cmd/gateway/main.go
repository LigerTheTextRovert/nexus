// Package main
package main

import (
	"fmt"

	"log/slog"
	"os"

	"github.com/LigerTheTextRovert/nexus/internal/config"
	"github.com/LigerTheTextRovert/nexus/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("gateway failed to start", "err", err)
		os.Exit(1)
	}
}

func run() error {
	var cfg config.Config

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
	slog.SetDefault(logger)

	err := config.LoadConfig("configs/config.yml", &cfg)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}
	slog.Info("config loaded", "path", "configs/config.yml", "routes", len(cfg.Routes))

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to validate the config file: %w", err)
	}
	slog.Info("config validated", "port", cfg.Port, "routes", len(cfg.Routes))

	server, err := server.New(&cfg)
	if err != nil {
		return fmt.Errorf("failed to start the server :%w", err)
	}
	return server.Start()
}
