package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

const (
	// DefaultTimeout is the default HTTP timeout for Discord requests
	DefaultTimeout = 30 * time.Second

	// MeteoraBrandColor is the official Meteora brand color (teal)
	MeteoraBrandColor = 0x00D4AA

	// HTTPStatusOK is the HTTP 200 OK status code
	HTTPStatusOK = 200

	// HTTPStatusMultipleChoices is the upper bound for successful HTTP status codes
	HTTPStatusMultipleChoices = 300

	// MaxRetriesDefault is the default maximum number of retries for Discord operations
	MaxRetriesDefault = 3

	// RetryDelayDefault is the default delay between retry attempts
	RetryDelayDefault = 5 * time.Second

	// EmbedColorDefault is the default color for Discord embeds
	EmbedColorDefault = 0x00ff00

	// BotTokenPrefix is the prefix for Discord bot tokens
	BotTokenPrefix = "Bot "
)

// Bot represents a Discord bot adapter
type Bot struct {
	config        *config.DiscordConfig
	logger        logger.Logger
	client        *DiscordClient
	messageSender *MessageSender
	embedBuilder  *EmbedBuilder
}

// NewBot creates a new Discord bot adapter
func NewBot(cfg *config.DiscordConfig, log logger.Logger) *Bot {
	client := NewDiscordClient(cfg, log)

	return &Bot{
		config:        cfg,
		logger:        log,
		client:        client,
		messageSender: NewMessageSender(cfg, log, client.GetSession()),
		embedBuilder:  NewEmbedBuilder(cfg),
	}
}

// SendTokenCreationNotification sends a token creation notification to Discord
func (b *Bot) SendTokenCreationNotification(ctx context.Context, event *domain.TokenCreationEvent) error {
	b.logger.WithFields(map[string]interface{}{
		"token_address": event.TokenAddress,
		"token_name":    event.GetDisplayName(),
	}).Info("Sending token creation notification to Discord")

	embed := b.embedBuilder.CreateTokenCreationEmbed(event)
	message := DiscordMessage{
		Username:  b.config.Username,
		AvatarURL: b.config.AvatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	return b.messageSender.SendMessage(ctx, message)
}

// SendTransactionNotification sends a transaction notification to Discord
func (b *Bot) SendTransactionNotification(ctx context.Context, tx *domain.Transaction) error {
	b.logger.WithFields(map[string]interface{}{
		"signature":  tx.Signature,
		"program_id": tx.ProgramID,
	}).Info("Sending transaction notification to Discord")

	embed := b.embedBuilder.CreateTransactionEmbed(tx)
	message := DiscordMessage{
		Username:  b.config.Username,
		AvatarURL: b.config.AvatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	return b.messageSender.SendMessage(ctx, message)
}

// SendMeteoraPoolNotification sends a Meteora pool notification to Discord
func (b *Bot) SendMeteoraPoolNotification(ctx context.Context, event *domain.MeteoraPoolEvent) error {
	b.logger.WithFields(map[string]interface{}{
		"pool_address": event.PoolAddress,
		"token_pair":   event.GetPairDisplayName(),
		"creator":      event.CreatorWallet,
	}).Info("Sending Meteora pool notification to Discord")

	embed := b.embedBuilder.CreateMeteoraPoolEmbed(event)
	message := DiscordMessage{
		Username:  b.config.Username,
		AvatarURL: b.config.AvatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	return b.messageSender.SendMessage(ctx, message)
}

// IsHealthy checks if the Discord bot is healthy and can send messages
func (b *Bot) IsHealthy(ctx context.Context) error {
	if b.config.BotToken == "" && b.config.WebhookURL == "" {
		return fmt.Errorf("neither bot token nor webhook URL is configured")
	}

	return b.client.CheckHealth(ctx)
}

// Start starts the Discord bot
func (b *Bot) Start(ctx context.Context) error {
	return b.client.Start(ctx)
}

// Stop stops the Discord bot
func (b *Bot) Stop() error {
	return b.client.Stop()
}

// IsRunning returns whether the bot is currently running
func (b *Bot) IsRunning() bool {
	return b.client.IsRunning()
}
