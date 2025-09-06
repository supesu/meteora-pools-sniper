package discord

import (
	"context"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/supesu/sniping-bot-v2/internal/adapters"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// DiscordConnectionManager manages Discord connections with auto-reconnection
type DiscordConnectionManager struct {
	*adapters.BaseConnectionManager

	config  *config.DiscordConfig
	logger  logger.Logger
	session *discordgo.Session
	mu      sync.RWMutex
}

// NewDiscordConnectionManager creates a new Discord connection manager
func NewDiscordConnectionManager(cfg *config.DiscordConfig, log logger.Logger, connConfig adapters.ConnectionConfig) *DiscordConnectionManager {
	manager := &DiscordConnectionManager{
		config: cfg,
		logger: log,
	}

	connectFunc := func(ctx context.Context) error {
		return manager.connectDiscord(ctx)
	}

	disconnectFunc := func() error {
		return manager.disconnectDiscord()
	}

	healthCheckFunc := func(ctx context.Context) error {
		return manager.checkDiscordHealth(ctx)
	}

	baseManager := adapters.NewBaseConnectionManager(
		log,
		connConfig.MaxRetries,
		connConfig.BaseRetryDelay,
		connConfig.MaxRetryDelay,
		connConfig.HealthCheckInterval,
		connectFunc,
		disconnectFunc,
		healthCheckFunc,
	)

	manager.BaseConnectionManager = baseManager
	return manager
}

// connectDiscord establishes the Discord connection
func (m *DiscordConnectionManager) connectDiscord(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session != nil {
		// Close existing session if it exists
		if err := m.session.Close(); err != nil {
			m.logger.WithError(err).Warn("Failed to close existing Discord session")
		}
	}

	var err error
	if m.config.BotToken != "" {
		m.session, err = discordgo.New("Bot " + m.config.BotToken)
	} else if m.config.WebhookURL != "" {
		// Webhook connections don't need a session
		m.logger.Info("Using webhook-based Discord notifications")
		return nil
	} else {
		return fmt.Errorf("neither bot token nor webhook URL configured")
	}

	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Add event handlers
	m.session.AddHandler(m.onReady)
	m.session.AddHandler(m.onDisconnect)
	m.session.AddHandler(m.onConnect)

	// Set intents
	m.session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	// Open the connection
	if err := m.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	m.logger.Info("Discord connection established successfully")
	return nil
}

// disconnectDiscord closes the Discord connection
func (m *DiscordConnectionManager) disconnectDiscord() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session != nil {
		err := m.session.Close()
		m.session = nil
		return err
	}
	return nil
}

// checkDiscordHealth performs a health check on the Discord connection
func (m *DiscordConnectionManager) checkDiscordHealth(ctx context.Context) error {
	m.mu.RLock()
	session := m.session
	m.mu.RUnlock()

	if session == nil {
		return fmt.Errorf("Discord session is nil")
	}

	// Try to get current user to verify connection
	_, err := session.User("@me")
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	return nil
}

// GetSession returns the Discord session
func (m *DiscordConnectionManager) GetSession() *discordgo.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.session
}

// onReady handles the ready event
func (m *DiscordConnectionManager) onReady(s *discordgo.Session, event *discordgo.Ready) {
	m.logger.WithFields(map[string]interface{}{
		"username":      event.User.Username,
		"discriminator": event.User.Discriminator,
		"guilds":        len(event.Guilds),
	}).Info("Discord bot is ready")
}

// onConnect handles connection events
func (m *DiscordConnectionManager) onConnect(s *discordgo.Session, event *discordgo.Connect) {
	m.logger.Info("Discord bot connected to gateway")
}

// onDisconnect handles disconnection events
func (m *DiscordConnectionManager) onDisconnect(s *discordgo.Session, event *discordgo.Disconnect) {
	m.logger.Warn("Discord bot disconnected from gateway")

	// Trigger reconnection if we're not already reconnecting
	if m.GetState() == adapters.StateConnected {
		m.triggerReconnection()
	}
}

// triggerReconnection triggers a reconnection attempt
func (m *DiscordConnectionManager) triggerReconnection() {
	// Use the base manager's trigger method
	m.TriggerReconnection()
}
