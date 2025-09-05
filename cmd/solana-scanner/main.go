package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/supesu/sniping-bot-v2/internal/adapters/solana"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel, cfg.Environment)

	// Create Meteora scanner
	log.Info("Starting Meteora pool scanner...")
	scanner := solana.NewSimpleMeteoraScanner(cfg, log)

	// Handle graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal")
		cancel()

		scanner.Stop()
	}()

	// Start the scanner
	if err := scanner.Start(); err != nil {
		log.WithField("error", err).Fatal("Failed to start scanner")
	}

	// Wait for shutdown
	scanner.Wait()
	log.Info("Scanner shutdown complete")
}
