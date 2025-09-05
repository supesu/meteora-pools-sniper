package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
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
	config     *config.DiscordConfig
	logger     logger.Logger
	httpClient *http.Client
	session    *discordgo.Session
	isRunning  bool
	mu         sync.RWMutex
}

// NewBot creates a new Discord bot adapter
func NewBot(cfg *config.DiscordConfig, log logger.Logger) *Bot {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = DefaultTimeout
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
	if b.config.BotToken == "" && b.config.WebhookURL == "" {
		return fmt.Errorf("neither bot token nor webhook URL is configured")
	}

	// For now, assume healthy if we have configuration
	// TODO: Implement proper health checks
	return nil
}

// checkSessionHealth performs a health check on the Discord session
func (b *Bot) checkSessionHealth(ctx context.Context) error {
	b.mu.RLock()
	session := b.session
	isRunning := b.isRunning
	b.mu.RUnlock()

	b.logger.WithFields(map[string]interface{}{
		"session_nil": session == nil,
		"is_running":  isRunning,
	}).Debug("Discord session health check")

	if session == nil {
		return fmt.Errorf("bot session is not available")
	}

	// For now, just check that we have a session
	// The session should be valid if it was created successfully
	// TODO: Add more sophisticated health checks later

	return nil
}

// Start starts the Discord bot and connects to the gateway
func (b *Bot) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isRunning {
		return fmt.Errorf("bot is already running")
	}

	if b.config.BotToken == "" {
		return fmt.Errorf("bot token is required for gateway connection")
	}

	// Create a new Discord session using the provided bot token
	session, err := discordgo.New(BotTokenPrefix + b.config.BotToken)
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Add event handlers
	session.AddHandler(b.onReady)
	session.AddHandler(b.onDisconnect)

	// Open the websocket connection to Discord
	err = session.Open()
	if err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	// Set session and running state only after successful connection
	b.session = session
	b.isRunning = true

	b.logger.WithFields(map[string]interface{}{
		"session_state": "connected",
		"data_ready":    session.DataReady,
	}).Info("Discord bot started successfully")

	return nil
}

// Stop stops the Discord bot and closes the gateway connection
func (b *Bot) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.isRunning {
		return nil
	}

	if b.session != nil {
		err := b.session.Close()
		if err != nil {
			b.logger.WithError(err).Error("Failed to close Discord session")
			return err
		}
	}

	b.isRunning = false
	b.session = nil

	b.logger.Info("Discord bot stopped successfully")
	return nil
}

// IsRunning returns whether the bot is currently running
func (b *Bot) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	isRunning := b.isRunning
	b.logger.WithFields(map[string]interface{}{
		"is_running":  isRunning,
		"has_session": b.session != nil,
	}).Debug("Checking if Discord bot is running")
	return isRunning
}

// onReady is called when the bot successfully connects to Discord
func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Ensure the running state is properly set
	b.isRunning = true

	b.logger.WithFields(map[string]interface{}{
		"username":      event.User.Username,
		"discriminator": event.User.Discriminator,
		"guilds":        len(event.Guilds),
		"session_id":    event.SessionID,
		"data_ready":    s.DataReady,
	}).Info("Discord bot is ready and online")
}

// onDisconnect is called when the bot disconnects from Discord
func (b *Bot) onDisconnect(s *discordgo.Session, event *discordgo.Disconnect) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Mark as not running when disconnected
	b.isRunning = false

	b.logger.WithFields(map[string]interface{}{
		"reason":            "disconnected from Discord gateway",
		"is_running_set_to": false,
	}).Warn("Discord bot disconnected")

	// Note: The discordgo library handles reconnection automatically
	// When it reconnects, onReady will be called again and we'll set isRunning = true
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
		Color:       MeteoraBrandColor, // Meteora brand color (teal)
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

// sendMessage sends a message to Discord using bot API or webhook as fallback
func (b *Bot) sendMessage(ctx context.Context, message DiscordMessage) error {
	var err error

	maxRetries := b.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = MaxRetriesDefault
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		var sendErr error

		// Prioritize bot API (gateway connection) over webhooks
		if b.config.BotToken != "" {
			sendErr = b.sendBotMessage(ctx, message)
		} else if b.config.WebhookURL != "" {
			sendErr = b.sendWebhookMessage(ctx, message)
		} else {
			return fmt.Errorf("no Discord bot token or webhook URL configured")
		}

		if sendErr == nil {
			return nil
		}

		// Log the error with context
		b.logger.WithFields(map[string]interface{}{
			"attempt":     attempt,
			"max_retries": maxRetries,
			"method":      b.getSendMethod(),
			"error":       sendErr.Error(),
		}).Warn("Failed to send Discord message")

		if attempt < maxRetries {
			retryDelay, parseErr := time.ParseDuration(b.config.RetryDelay)
			if parseErr != nil {
				retryDelay = RetryDelayDefault
			}

			b.logger.WithFields(map[string]interface{}{
				"attempt":     attempt,
				"retry_delay": retryDelay.String(),
			}).Info("Retrying Discord message send after delay")

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while retrying: %w", ctx.Err())
			case <-time.After(retryDelay):
				continue
			}
		}

		// Final attempt failed
		return fmt.Errorf("failed to send Discord message after %d attempts: %w", maxRetries, sendErr)
	}

	return fmt.Errorf("failed to send Discord message after %d attempts: %w", maxRetries, err)
}

// getSendMethod returns a string indicating which send method is being used
func (b *Bot) getSendMethod() string {
	if b.config.BotToken != "" {
		return "bot_api"
	} else if b.config.WebhookURL != "" {
		return "webhook"
	}
	return "unknown"
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

	if resp.StatusCode < HTTPStatusOK || resp.StatusCode >= HTTPStatusMultipleChoices {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// sendBotMessage sends a message using Discord bot API
func (b *Bot) sendBotMessage(ctx context.Context, message DiscordMessage) error {
	b.mu.RLock()
	session := b.session
	isRunning := b.isRunning
	b.mu.RUnlock()

	if !isRunning || session == nil {
		// Fall back to HTTP API if session is not available
		return b.sendBotMessageHTTP(ctx, message)
	}

	// Convert DiscordEmbed to discordgo.MessageEmbed
	var embeds []*discordgo.MessageEmbed
	for _, embed := range message.Embeds {
		dgEmbed := &discordgo.MessageEmbed{
			Title:       embed.Title,
			Description: embed.Description,
			URL:         embed.URL,
			Timestamp:   embed.Timestamp,
			Color:       embed.Color,
		}

		// Convert fields
		if len(embed.Fields) > 0 {
			dgEmbed.Fields = make([]*discordgo.MessageEmbedField, len(embed.Fields))
			for i, field := range embed.Fields {
				dgEmbed.Fields[i] = &discordgo.MessageEmbedField{
					Name:   field.Name,
					Value:  field.Value,
					Inline: field.Inline,
				}
			}
		}

		// Convert footer
		if embed.Footer.Text != "" {
			dgEmbed.Footer = &discordgo.MessageEmbedFooter{
				Text: embed.Footer.Text,
			}
		}

		embeds = append(embeds, dgEmbed)
	}

	// Send message using discordgo session
	_, err := session.ChannelMessageSendComplex(b.config.ChannelID, &discordgo.MessageSend{
		Content: message.Content,
		Embeds:  embeds,
	})

	if err != nil {
		return fmt.Errorf("failed to send message via discordgo: %w", err)
	}

	return nil
}

// sendBotMessageHTTP sends a message using Discord HTTP API (fallback)
func (b *Bot) sendBotMessageHTTP(ctx context.Context, message DiscordMessage) error {
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
	req.Header.Set("Authorization", fmt.Sprintf("%s%s", BotTokenPrefix, b.config.BotToken))

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send bot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < HTTPStatusOK || resp.StatusCode >= HTTPStatusMultipleChoices {
		return fmt.Errorf("bot request failed with status: %d", resp.StatusCode)
	}

	return nil
}
