package discord

import (
	"context"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/supesu/sniping-bot-v2/internal/adapters"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// DiscordClient handles Discord connection and session management
type DiscordClient struct {
	config            *config.DiscordConfig
	logger            logger.Logger
	isRunning         bool
	mu                sync.RWMutex
	connectionManager *DiscordConnectionManager
}

// NewDiscordClient creates a new Discord client
func NewDiscordClient(cfg *config.DiscordConfig, log logger.Logger) *DiscordClient {
	client := &DiscordClient{
		config: cfg,
		logger: log,
	}

	// Initialize connection manager with default config
	connConfig := adapters.DefaultConnectionConfig()
	client.connectionManager = NewDiscordConnectionManager(cfg, log, connConfig)

	return client
}

// Connect establishes a Discord session
func (c *DiscordClient) Connect() error {
	return c.connectionManager.Connect(context.Background())
}

// Start opens the Discord connection and starts auto-reconnection
func (c *DiscordClient) Start(ctx context.Context) error {
	if err := c.connectionManager.Connect(ctx); err != nil {
		return err
	}

	// Start auto-reconnection
	c.connectionManager.StartAutoReconnect(ctx)

	c.mu.Lock()
	c.isRunning = true
	c.mu.Unlock()

	c.logger.Info("Discord client started successfully with auto-reconnection")
	return nil
}

// Stop closes the Discord connection and stops auto-reconnection
func (c *DiscordClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isRunning = false

	// Stop auto-reconnection
	c.connectionManager.StopAutoReconnect()

	// Disconnect
	return c.connectionManager.Disconnect()
}

// IsRunning returns whether the client is running
func (c *DiscordClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

// GetSession returns the Discord session
func (c *DiscordClient) GetSession() *discordgo.Session {
	return c.connectionManager.GetSession()
}

// CheckHealth performs a health check
func (c *DiscordClient) CheckHealth(ctx context.Context) error {
	return c.connectionManager.IsHealthy(ctx)
}
