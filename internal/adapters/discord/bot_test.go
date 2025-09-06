package discord

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

func TestNewBot(t *testing.T) {
	cfg := &config.DiscordConfig{
		BotToken:  "test-token",
		ChannelID: "test-channel",
		Username:  "TestBot",
	}
	log := logger.New("info", "test")

	bot := NewBot(cfg, log)

	assert.NotNil(t, bot)
	assert.Equal(t, cfg, bot.config)
	assert.Equal(t, log, bot.logger)
	assert.NotNil(t, bot.client)
	assert.NotNil(t, bot.messageSender)
	assert.NotNil(t, bot.embedBuilder)
}

func TestBot_SendMeteoraPoolNotification_ValidEvent(t *testing.T) {
	cfg := &config.DiscordConfig{
		BotToken:  "test-token",
		ChannelID: "test-channel",
		Username:  "TestBot",
	}
	log := logger.New("info", "test")

	bot := NewBot(cfg, log)

	event := &domain.MeteoraPoolEvent{
		PoolAddress:     "test-pool",
		TokenAMint:      "token-a",
		TokenBMint:      "token-b",
		TokenASymbol:    "SOL",
		TokenBSymbol:    "USDC",
		CreatorWallet:   "creator-123",
		TransactionHash: "tx-123",
		CreatedAt:       time.Now(),
	}

	// Test that the method doesn't panic with valid input
	// Note: This will fail in CI without Discord credentials, which is expected
	err := bot.SendMeteoraPoolNotification(context.Background(), event)

	// We expect an error since we don't have real Discord credentials
	// But the important thing is that the method executes without panicking
	assert.Error(t, err)
	assert.NotNil(t, bot)
}

func TestBot_IsHealthy_NoConfiguration(t *testing.T) {
	cfg := &config.DiscordConfig{} // Empty config
	log := logger.New("info", "test")

	bot := NewBot(cfg, log)

	err := bot.IsHealthy(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "neither bot token nor webhook URL is configured")
}

func TestBot_IsHealthy_WithConfiguration(t *testing.T) {
	cfg := &config.DiscordConfig{
		BotToken:  "test-token",
		ChannelID: "test-channel",
	}
	log := logger.New("info", "test")

	bot := NewBot(cfg, log)

	// This will attempt to connect to Discord, which will fail in test environment
	// But it should not fail due to configuration issues
	err := bot.IsHealthy(context.Background())

	// We expect some kind of connection/network error, not a configuration error
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "neither bot token nor webhook URL is configured")
}

func TestBot_SendTokenCreationNotification_ValidEvent(t *testing.T) {
	cfg := &config.DiscordConfig{
		BotToken:  "test-token",
		ChannelID: "test-channel",
		Username:  "TestBot",
	}
	log := logger.New("info", "test")

	bot := NewBot(cfg, log)

	event := &domain.TokenCreationEvent{
		TokenAddress:    "test-token-addr",
		CreatorAddress:  "creator-123",
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		TransactionHash: "tx-123",
		Timestamp:       time.Now(),
	}

	// Test that the method doesn't panic with valid input
	err := bot.SendTokenCreationNotification(context.Background(), event)

	// We expect an error since we don't have real Discord credentials
	assert.Error(t, err)
	assert.NotNil(t, bot)
}

func TestBot_SendTransactionNotification_ValidEvent(t *testing.T) {
	cfg := &config.DiscordConfig{
		BotToken:  "test-token",
		ChannelID: "test-channel",
		Username:  "TestBot",
	}
	log := logger.New("info", "test")

	bot := NewBot(cfg, log)

	tx := &domain.Transaction{
		Signature: "test-sig",
		ProgramID: "test-program",
		Accounts:  []string{"acc1", "acc2"},
		Timestamp: time.Now(),
	}

	// Test that the method doesn't panic with valid input
	err := bot.SendTransactionNotification(context.Background(), tx)

	// We expect an error since we don't have real Discord credentials
	assert.Error(t, err)
	assert.NotNil(t, bot)
}
