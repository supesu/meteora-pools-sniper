package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/supesu/sniping-bot-v2/internal"
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
	log.Info("Starting gRPC server...")

	// Initialize Discord bot
	discordBot := internal.InitializeDiscordBot()

	// Start Discord bot
	if cfg.Discord.BotToken != "" && cfg.Discord.BotToken != "YOUR_DISCORD_BOT_TOKEN_HERE" {
		log.Info("Starting Discord bot...")
		botCtx := context.Background()
		if err := discordBot.Start(botCtx); err != nil {
			log.WithError(err).Error("Failed to start Discord bot")
			// Continue with server startup even if Discord bot fails
		} else {
			log.Info("Discord bot started successfully")
			defer func() {
				if err := discordBot.Stop(); err != nil {
					log.WithError(err).Error("Failed to stop Discord bot")
				}
			}()
		}
	} else {
		log.Warn("Discord bot token not configured, skipping bot startup")
	}

	// Create gRPC server using Wire dependency injection
	server := internal.InitializeGRPCServer()
	defer server.Stop()

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal")
		cancel()
		server.Stop()
	}()

	// Start the server
	log.WithFields(map[string]interface{}{
		"address":     cfg.Address(),
		"environment": cfg.Environment,
		"log_level":   cfg.LogLevel,
	}).Info("Server configuration loaded")

	if err := server.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start server")
	}

	<-ctx.Done()
	log.Info("gRPC server shutdown complete")
}
