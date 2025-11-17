package main

import (
	"fmt"
	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/gabkaclassic/metrics/pkg/logger"
	"log"
	"log/slog"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {

	cfg, err := config.ParseConfig()

	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	logger.SetupLogger(logger.LogConfig(cfg.Log))

	slog.Debug("Server starting...")

	return nil
}
