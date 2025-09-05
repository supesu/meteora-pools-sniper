package discord

import (
	"context"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// DiscordClient handles Discord connection and session management
type DiscordClient struct {
	config    *config.DiscordConfig
	logger    logger.Logger
	session   *discordgo.Session
	isRunning bool
	mu        sync.RWMutex
}

// NewDiscordClient creates a new Discord client
func NewDiscordClient(cfg *config.DiscordConfig, log logger.Logger) *DiscordClient {
	return &DiscordClient{
		config: cfg,
		logger: log,
	}
}

// Connect establishes a Discord session
func (c *DiscordClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		return nil // Already connected
	}

	var err error
	if c.config.BotToken != "" {
		c.session, err = discordgo.New("Bot " + c.config.BotToken)
	} else if c.config.WebhookURL != "" {
		// Webhook connections don't need a session
		c.logger.Info("Using webhook-based Discord notifications")
		return nil
	} else {
		return fmt.Errorf("neither bot token nor webhook URL configured")
	}

	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Add event handlers
	c.session.AddHandler(c.onReady)
	c.session.AddHandler(c.onDisconnect)

	// Set intents
	c.session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	return nil
}

// Start opens the Discord connection
func (c *DiscordClient) Start(ctx context.Context) error {
	if c.session == nil {
		if err := c.Connect(); err != nil {
			return err
		}
	}

	if err := c.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	c.mu.Lock()
	c.isRunning = true
	c.mu.Unlock()

	c.logger.Info("Discord client started successfully")
	return nil
}

// Stop closes the Discord connection
func (c *DiscordClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isRunning = false

	if c.session != nil {
		err := c.session.Close()
		c.session = nil
		return err
	}

	return nil
}

// IsRunning returns whether the client is running
func (c *DiscordClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

// GetSession returns the Discord session
func (c *DiscordClient) GetSession() *discordgo.Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.session
}

// onReady handles the ready event
func (c *DiscordClient) onReady(s *discordgo.Session, event *discordgo.Ready) {
	c.logger.WithFields(map[string]interface{}{
		"username":      event.User.Username,
		"discriminator": event.User.Discriminator,
		"guilds":        len(event.Guilds),
	}).Info("Discord bot is ready")
}

// onDisconnect handles disconnection events
func (c *DiscordClient) onDisconnect(s *discordgo.Session, event *discordgo.Disconnect) {
	c.mu.Lock()
	c.isRunning = false
	c.mu.Unlock()

	c.logger.Warn("Discord bot disconnected")
}

// CheckHealth performs a health check
func (c *DiscordClient) CheckHealth(ctx context.Context) error {
	if !c.IsRunning() {
		return fmt.Errorf("Discord client is not running")
	}

	session := c.GetSession()
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
