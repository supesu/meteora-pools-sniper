package container

import (
	"context"

	"github.com/supesu/sniping-bot-v2/internal/adapters/discord"
	"github.com/supesu/sniping-bot-v2/internal/adapters/grpc"
	"github.com/supesu/sniping-bot-v2/internal/infrastructure/events"
	"github.com/supesu/sniping-bot-v2/internal/infrastructure/repository"
	"github.com/supesu/sniping-bot-v2/internal/usecase"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// Container holds all application dependencies
type Container struct {
	// Configuration and infrastructure
	Config *config.Config
	Logger logger.Logger

	// Repositories (data layer)
	TransactionRepo  domain.TransactionRepository
	SubscriptionRepo domain.SubscriptionRepository
	MeteoraRepo      domain.MeteoraRepository
	DiscordRepo      domain.DiscordNotificationRepository

	// Event publisher
	EventPublisher domain.EventPublisher

	// Use cases (application layer)
	ProcessTransactionUC        *usecase.ProcessTransactionUseCase
	GetTransactionHistoryUC     *usecase.GetTransactionHistoryUseCase
	ManageSubscriptionsUC       *usecase.ManageSubscriptionsUseCase
	NotifyTokenCreationUC       *usecase.NotifyTokenCreationUseCase
	ProcessMeteoraEventUC       *usecase.ProcessMeteoraEventUseCase
	NotifyMeteoraPoolCreationUC *usecase.NotifyMeteoraPoolCreationUseCase

	// Services (presentation layer)
	GRPCServer *grpc.Server
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *config.Config, log logger.Logger) *Container {
	container := &Container{
		Config: cfg,
		Logger: log,
	}

	container.setupRepositories()
	container.setupEventPublisher()
	container.setupUseCases()
	container.setupServices()

	return container
}

// setupRepositories initializes repository implementations
func (c *Container) setupRepositories() {
	c.Logger.Info("Setting up repositories")

	// In-memory implementations for basic data storage
	c.TransactionRepo = repository.NewTransactionRepository()
	c.SubscriptionRepo = repository.NewSubscriptionRepository()
	c.MeteoraRepo = repository.NewMeteoraRepository(c.Logger)

	// Discord notification repository
	c.DiscordRepo = discord.NewBot(&c.Config.Discord, c.Logger)

	c.Logger.Info("Repositories initialized")
}

// setupEventPublisher initializes the event publisher
func (c *Container) setupEventPublisher() {
	c.Logger.Info("Setting up event publisher")

	c.EventPublisher = events.NewMemoryEventPublisher(c.Logger)

	c.Logger.Info("Event publisher initialized")
}

// setupUseCases initializes use case implementations
func (c *Container) setupUseCases() {
	c.Logger.Info("Setting up use cases")

	// Process Transaction Use Case
	c.ProcessTransactionUC = usecase.NewProcessTransactionUseCase(
		c.TransactionRepo,
		c.EventPublisher,
		c.Logger,
	)

	// Get Transaction History Use Case
	c.GetTransactionHistoryUC = usecase.NewGetTransactionHistoryUseCase(
		c.TransactionRepo,
		c.Logger,
	)

	// Manage Subscriptions Use Case
	c.ManageSubscriptionsUC = usecase.NewManageSubscriptionsUseCase(
		c.SubscriptionRepo,
		c.EventPublisher,
		c.Logger,
	)

	// Notify Token Creation Use Case
	c.NotifyTokenCreationUC = usecase.NewNotifyTokenCreationUseCase(
		c.DiscordRepo,
		c.Logger,
	)

	// Process Meteora Event Use Case
	c.ProcessMeteoraEventUC = usecase.NewProcessMeteoraEventUseCase(
		c.MeteoraRepo,
		c.EventPublisher,
		c.Logger,
	)

	// Notify Meteora Pool Creation Use Case
	c.NotifyMeteoraPoolCreationUC = usecase.NewNotifyMeteoraPoolCreationUseCase(
		c.DiscordRepo,
		c.Logger,
	)

	c.Logger.Info("Use cases initialized")
}

// setupServices initializes service implementations
func (c *Container) setupServices() {
	c.Logger.Info("Setting up services")

	// gRPC Server
	deps := grpc.Dependencies{
		Config:                      c.Config,
		Logger:                      c.Logger,
		ProcessTransactionUC:        c.ProcessTransactionUC,
		GetTransactionHistoryUC:     c.GetTransactionHistoryUC,
		ManageSubscriptionsUC:       c.ManageSubscriptionsUC,
		NotifyTokenCreationUC:       c.NotifyTokenCreationUC,
		ProcessMeteoraEventUC:       c.ProcessMeteoraEventUC,
		NotifyMeteoraPoolCreationUC: c.NotifyMeteoraPoolCreationUC,
		SubscriptionRepo:            c.SubscriptionRepo,
	}

	c.GRPCServer = grpc.NewServer(deps)

	c.Logger.Info("Services initialized")
}

// GetGRPCServer returns the configured gRPC server
func (c *Container) GetGRPCServer() *grpc.Server {
	return c.GRPCServer
}

// GetEventPublisher returns the event publisher
func (c *Container) GetEventPublisher() domain.EventPublisher {
	return c.EventPublisher
}

// StartDiscordBot starts the Discord bot if configured
func (c *Container) StartDiscordBot(ctx context.Context) error {
	if c.DiscordRepo == nil {
		c.Logger.Warn("Discord repository not configured, skipping bot startup")
		return nil
	}

	// Type assert to discord.Bot to access Start method
	if bot, ok := c.DiscordRepo.(*discord.Bot); ok {
		// Check if bot is already running
		if bot.IsRunning() {
			c.Logger.Info("Discord bot is already running")
			return nil
		}

		err := bot.Start(ctx)
		if err != nil {
			c.Logger.WithError(err).Error("Failed to start Discord bot")
			return err
		}
		c.Logger.Info("Discord bot started successfully")
		return nil
	}

	c.Logger.Warn("Discord repository is not a Bot instance, cannot start bot")
	return nil
}

// StopDiscordBot stops the Discord bot if running
func (c *Container) StopDiscordBot() error {
	if c.DiscordRepo == nil {
		return nil
	}

	// Type assert to discord.Bot to access Stop method
	if bot, ok := c.DiscordRepo.(*discord.Bot); ok {
		err := bot.Stop()
		if err != nil {
			c.Logger.WithError(err).Error("Failed to stop Discord bot")
			return err
		}
		c.Logger.Info("Discord bot stopped successfully")
		return nil
	}

	return nil
}

// Shutdown performs cleanup of container resources
func (c *Container) Shutdown() {
	c.Logger.Info("Shutting down container")

	// Stop Discord bot first
	if err := c.StopDiscordBot(); err != nil {
		c.Logger.WithError(err).Error("Error stopping Discord bot")
	}

	if c.GRPCServer != nil {
		c.GRPCServer.Stop()
	}

	c.Logger.Info("Container shutdown complete")
}
