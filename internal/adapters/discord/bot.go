package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// Bot represents a Discord bot adapter
type Bot struct {
	config     *config.DiscordConfig
	logger     logger.Logger
	httpClient *http.Client
}

// NewBot creates a new Discord bot adapter
func NewBot(cfg *config.DiscordConfig, log logger.Logger) *Bot {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	return &Bot{
		config: cfg,
		logger: log,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SendTokenCreationNotification sends a token creation notification to Discord
func (b *Bot) SendTokenCreationNotification(ctx context.Context, event *domain.TokenCreationEvent) error {
	b.logger.WithFields(map[string]interface{}{
		"token_address": event.TokenAddress,
		"token_name":    event.GetDisplayName(),
	}).Info("Sending token creation notification to Discord")

	embed := b.createTokenCreationEmbed(event)
	message := DiscordMessage{
		Username:  b.config.Username,
		AvatarURL: b.config.AvatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	return b.sendMessage(ctx, message)
}

// SendTransactionNotification sends a transaction notification to Discord
func (b *Bot) SendTransactionNotification(ctx context.Context, tx *domain.Transaction) error {
	b.logger.WithFields(map[string]interface{}{
		"signature":  tx.Signature,
		"program_id": tx.ProgramID,
	}).Info("Sending transaction notification to Discord")

	embed := b.createTransactionEmbed(tx)
	message := DiscordMessage{
		Username:  b.config.Username,
		AvatarURL: b.config.AvatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	return b.sendMessage(ctx, message)
}

// SendMeteoraPoolNotification sends a Meteora pool notification to Discord
func (b *Bot) SendMeteoraPoolNotification(ctx context.Context, event *domain.MeteoraPoolEvent) error {
	b.logger.WithFields(map[string]interface{}{
		"pool_address": event.PoolAddress,
		"token_pair":   event.GetPairDisplayName(),
		"creator":      event.CreatorWallet,
	}).Info("Sending Meteora pool notification to Discord")

	embed := b.createMeteoraPoolEmbed(event)
	message := DiscordMessage{
		Username:  b.config.Username,
		AvatarURL: b.config.AvatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	return b.sendMessage(ctx, message)
}

// IsHealthy checks if the Discord bot is healthy and can send messages
func (b *Bot) IsHealthy(ctx context.Context) error {
	if b.config.WebhookURL == "" && b.config.BotToken == "" {
		return fmt.Errorf("neither webhook URL nor bot token is configured")
	}

	// For webhook, we can't really test without sending a message
	// For bot token, we could ping the Discord API, but for simplicity we'll just check config
	return nil
}

// createTokenCreationEmbed creates a Discord embed for token creation events
func (b *Bot) createTokenCreationEmbed(event *domain.TokenCreationEvent) DiscordEmbed {
	embed := DiscordEmbed{
		Title:       "🪙 New Token Created!",
		Description: fmt.Sprintf("A new token **%s** has been created on Solana!", event.GetDisplayName()),
		Color:       b.config.EmbedColor,
		Timestamp:   event.Timestamp.Format(time.RFC3339),
		Fields: []EmbedField{
			{
				Name:   "Token Address",
				Value:  fmt.Sprintf("`%s`", event.TokenAddress),
				Inline: false,
			},
			{
				Name:   "Creator",
				Value:  fmt.Sprintf("`%s`", event.CreatorAddress),
				Inline: false,
			},
		},
		Footer: EmbedFooter{
			Text: "Solana Token Sniping Bot",
		},
	}

	// Add optional fields if available
	if event.TokenName != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Name",
			Value:  event.TokenName,
			Inline: true,
		})
	}

	if event.TokenSymbol != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Symbol",
			Value:  event.TokenSymbol,
			Inline: true,
		})
	}

	if event.InitialSupply != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Initial Supply",
			Value:  event.InitialSupply,
			Inline: true,
		})
	}

	embed.Fields = append(embed.Fields, EmbedField{
		Name:   "Decimals",
		Value:  strconv.Itoa(event.Decimals),
		Inline: true,
	})

	embed.Fields = append(embed.Fields, EmbedField{
		Name:   "Slot",
		Value:  strconv.FormatUint(event.Slot, 10),
		Inline: true,
	})

	embed.Fields = append(embed.Fields, EmbedField{
		Name:   "Transaction",
		Value:  fmt.Sprintf("[View on Solscan](https://solscan.io/tx/%s)", event.TransactionHash),
		Inline: false,
	})

	return embed
}

// createTransactionEmbed creates a Discord embed for general transactions
func (b *Bot) createTransactionEmbed(tx *domain.Transaction) DiscordEmbed {
	embed := DiscordEmbed{
		Title:       "📊 Transaction Detected",
		Description: fmt.Sprintf("New transaction detected for program `%s`", tx.ProgramID),
		Color:       b.config.EmbedColor,
		Timestamp:   tx.Timestamp.Format(time.RFC3339),
		Fields: []EmbedField{
			{
				Name:   "Signature",
				Value:  fmt.Sprintf("`%s`", tx.Signature),
				Inline: false,
			},
			{
				Name:   "Program ID",
				Value:  fmt.Sprintf("`%s`", tx.ProgramID),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  tx.Status.String(),
				Inline: true,
			},
			{
				Name:   "Slot",
				Value:  strconv.FormatUint(tx.Slot, 10),
				Inline: true,
			},
			{
				Name:   "Accounts",
				Value:  strconv.Itoa(len(tx.Accounts)),
				Inline: true,
			},
		},
		Footer: EmbedFooter{
			Text: "Solana Token Sniping Bot",
		},
	}

	// Add scanner ID if available
	if tx.ScannerID != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Scanner ID",
			Value:  tx.ScannerID,
			Inline: true,
		})
	}

	// Add transaction link
	embed.Fields = append(embed.Fields, EmbedField{
		Name:   "Transaction",
		Value:  fmt.Sprintf("[View on Solscan](https://solscan.io/tx/%s)", tx.Signature),
		Inline: false,
	})

	return embed
}

// createMeteoraPoolEmbed creates a Discord embed for Meteora pool events
func (b *Bot) createMeteoraPoolEmbed(event *domain.MeteoraPoolEvent) DiscordEmbed {
	embed := DiscordEmbed{
		Title:       "🌊 New Meteora Pool Created!",
		Description: fmt.Sprintf("A new liquidity pool **%s** has been created on Meteora!", event.GetPairDisplayName()),
		Color:       0x00D4AA, // Meteora brand color (teal)
		Timestamp:   event.CreatedAt.Format(time.RFC3339),
		Fields: []EmbedField{
			{
				Name:   "Pool Address",
				Value:  fmt.Sprintf("`%s`", event.PoolAddress),
				Inline: false,
			},
			{
				Name:   "Token Pair",
				Value:  event.GetPairDisplayName(),
				Inline: true,
			},
			{
				Name:   "Pool Type",
				Value:  event.PoolType,
				Inline: true,
			},
			{
				Name:   "Creator",
				Value:  fmt.Sprintf("`%s`", event.CreatorWallet),
				Inline: false,
			},
		},
		Footer: EmbedFooter{
			Text: fmt.Sprintf("Slot: %d • Meteora Pool Scanner", event.Slot),
		},
	}

	// Add token details if available
	if event.TokenAName != "" || event.TokenASymbol != "" {
		tokenADisplay := event.TokenASymbol
		if event.TokenAName != "" && event.TokenAName != event.TokenASymbol {
			tokenADisplay = fmt.Sprintf("%s (%s)", event.TokenASymbol, event.TokenAName)
		}
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Token A",
			Value:  tokenADisplay,
			Inline: true,
		})
	}

	if event.TokenBName != "" || event.TokenBSymbol != "" {
		tokenBDisplay := event.TokenBSymbol
		if event.TokenBName != "" && event.TokenBName != event.TokenBSymbol {
			tokenBDisplay = fmt.Sprintf("%s (%s)", event.TokenBSymbol, event.TokenBName)
		}
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Token B",
			Value:  tokenBDisplay,
			Inline: true,
		})
	}

	// Add liquidity information if available
	if event.InitialLiquidityA > 0 || event.InitialLiquidityB > 0 {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Initial Liquidity",
			Value:  fmt.Sprintf("A: %d • B: %d", event.InitialLiquidityA, event.InitialLiquidityB),
			Inline: true,
		})
	}

	// Add fee rate if available
	if event.FeeRate > 0 {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Fee Rate",
			Value:  fmt.Sprintf("%.4f%%", float64(event.FeeRate)/10000.0),
			Inline: true,
		})
	}

	// Add action buttons
	embed.Fields = append(embed.Fields, EmbedField{
		Name: "Links",
		Value: fmt.Sprintf(
			"[View on Meteora](https://app.meteora.ag/pools/%s) • [View Transaction](https://solscan.io/tx/%s)",
			event.PoolAddress,
			event.TransactionHash,
		),
		Inline: false,
	})

	return embed
}

// sendMessage sends a message to Discord using webhook or bot API
func (b *Bot) sendMessage(ctx context.Context, message DiscordMessage) error {
	var err error

	for attempt := 1; attempt <= b.config.MaxRetries; attempt++ {
		if b.config.WebhookURL != "" {
			err = b.sendWebhookMessage(ctx, message)
		} else if b.config.BotToken != "" {
			err = b.sendBotMessage(ctx, message)
		} else {
			return fmt.Errorf("no Discord webhook URL or bot token configured")
		}

		if err == nil {
			return nil
		}

		if attempt < b.config.MaxRetries {
			retryDelay, parseErr := time.ParseDuration(b.config.RetryDelay)
			if parseErr != nil {
				retryDelay = 5 * time.Second
			}

			b.logger.WithFields(map[string]interface{}{
				"attempt":     attempt,
				"max_retries": b.config.MaxRetries,
				"retry_delay": retryDelay.String(),
				"error":       err.Error(),
			}).Warn("Failed to send Discord message, retrying...")

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				continue
			}
		}
	}

	return fmt.Errorf("failed to send Discord message after %d attempts: %w", b.config.MaxRetries, err)
}

// sendWebhookMessage sends a message using Discord webhook
func (b *Bot) sendWebhookMessage(ctx context.Context, message DiscordMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", b.config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// sendBotMessage sends a message using Discord bot API
func (b *Bot) sendBotMessage(ctx context.Context, message DiscordMessage) error {
	// Convert to bot API format
	botMessage := BotMessage{
		Content: message.Content,
		Embeds:  message.Embeds,
	}

	payload, err := json.Marshal(botMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal bot message: %w", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", b.config.ChannelID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create bot request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", b.config.BotToken))

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send bot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bot request failed with status: %d", resp.StatusCode)
	}

	return nil
}
