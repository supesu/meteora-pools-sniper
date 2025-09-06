package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// MessageSender handles Discord message sending (both bot and webhook)
type MessageSender struct {
	config     *config.DiscordConfig
	logger     logger.Logger
	httpClient *http.Client
	session    *discordgo.Session
}

// NewMessageSender creates a new message sender
func NewMessageSender(cfg *config.DiscordConfig, log logger.Logger, session *discordgo.Session) *MessageSender {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = DefaultTimeout
	}

	return &MessageSender{
		config: cfg,
		logger: log,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		session: session,
	}
}

// SendMessage sends a Discord message using the appropriate method
func (s *MessageSender) SendMessage(ctx context.Context, message DiscordMessage) error {
	method := s.getSendMethod()

	switch method {
	case "bot":
		return s.sendBotMessage(ctx, message)
	case "webhook":
		return s.sendWebhookMessage(ctx, message)
	default:
		return fmt.Errorf("unsupported send method: %s", method)
	}
}

// getSendMethod determines which sending method to use
func (s *MessageSender) getSendMethod() string {
	if s.config.BotToken != "" && s.session != nil {
		return "bot"
	}
	if s.config.WebhookURL != "" {
		return "webhook"
	}
	return ""
}

// sendBotMessage sends a message using the Discord bot API
func (s *MessageSender) sendBotMessage(ctx context.Context, message DiscordMessage) error {
	if s.session == nil {
		return fmt.Errorf("discord session not available")
	}

	if s.config.ChannelID == "" {
		return fmt.Errorf("channel ID not configured")
	}

	// Convert DiscordMessage to discordgo.MessageSend
	dgMessage := &discordgo.MessageSend{}

	if len(message.Embeds) > 0 {
		dgMessage.Embeds = make([]*discordgo.MessageEmbed, len(message.Embeds))
		for i, embed := range message.Embeds {
			dgFields := make([]*discordgo.MessageEmbedField, len(embed.Fields))
			for j, field := range embed.Fields {
				dgFields[j] = &discordgo.MessageEmbedField{
					Name:   field.Name,
					Value:  field.Value,
					Inline: field.Inline,
				}
			}

			dgMessage.Embeds[i] = &discordgo.MessageEmbed{
				Title:       embed.Title,
				Description: embed.Description,
				Color:       embed.Color,
				Fields:      dgFields,
				Timestamp:   embed.Timestamp,
			}
		}
	}

	_, err := s.session.ChannelMessageSendComplex(s.config.ChannelID, dgMessage)
	if err != nil {
		return fmt.Errorf("failed to send bot message: %w", err)
	}

	return nil
}

// sendWebhookMessage sends a message using Discord webhooks
func (s *MessageSender) sendWebhookMessage(ctx context.Context, message DiscordMessage) error {
	if s.config.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.WebhookURL, bytes.NewBuffer(messageJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < HTTPStatusOK || resp.StatusCode >= HTTPStatusMultipleChoices {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	return nil
}
