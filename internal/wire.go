//go:build wireinject
// +build wireinject

package internal

import (
	"github.com/google/wire"
	"github.com/supesu/sniping-bot-v2/internal/adapters/discord"
	"github.com/supesu/sniping-bot-v2/internal/adapters/grpc"
	"github.com/supesu/sniping-bot-v2/internal/infrastructure/events"
	"github.com/supesu/sniping-bot-v2/internal/infrastructure/repository"
	"github.com/supesu/sniping-bot-v2/internal/usecase"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// ApplicationSet provides the complete application dependency set
var ApplicationSet = wire.NewSet(
	repository.NewTransactionRepository,
	repository.NewSubscriptionRepository,
	repository.NewMeteoraRepository,
	usecase.UseCaseSet,
	discord.NewBot,
	grpc.NewServer,
	events.NewMemoryEventPublisher,
	ProvideLogger,
	ProvideConfig,
	ProvideDiscordConfig,
	ProvideGRPCDependencies,
	wire.Bind(new(domain.TransactionRepository), new(*repository.TransactionRepository)),
	wire.Bind(new(domain.SubscriptionRepository), new(*repository.SubscriptionRepository)),
	wire.Bind(new(domain.MeteoraRepository), new(*repository.MeteoraRepository)),
	wire.Bind(new(domain.DiscordNotificationRepository), new(*discord.Bot)),
	wire.Bind(new(domain.EventPublisher), new(*events.MemoryEventPublisher)),
)

// ProvideLogger creates a logger instance
func ProvideLogger(cfg *config.Config) logger.Logger {
	return logger.New(cfg.LogLevel, cfg.Environment)
}

// ProvideConfig creates a config instance
func ProvideConfig() *config.Config {
	cfg, err := config.Load("")
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}
	return cfg
}

// ProvideDiscordConfig extracts Discord config from main config
func ProvideDiscordConfig(cfg *config.Config) *config.DiscordConfig {
	return &cfg.Discord
}

// ProvideGRPCDependencies creates the dependencies struct for gRPC server
func ProvideGRPCDependencies(
	cfg *config.Config,
	log logger.Logger,
	processTransactionUC *usecase.ProcessTransactionUseCase,
	getTransactionHistoryUC *usecase.GetTransactionHistoryUseCase,
	manageSubscriptionsUC *usecase.ManageSubscriptionsUseCase,
	notifyTokenCreationUC *usecase.NotifyTokenCreationUseCase,
	processMeteoraEventUC *usecase.ProcessMeteoraEventUseCase,
	notifyMeteoraPoolCreationUC *usecase.NotifyMeteoraPoolCreationUseCase,
	subscriptionRepo domain.SubscriptionRepository,
) grpc.Dependencies {
	return grpc.Dependencies{
		Config:                      cfg,
		Logger:                      log,
		ProcessTransactionUC:        processTransactionUC,
		GetTransactionHistoryUC:     getTransactionHistoryUC,
		ManageSubscriptionsUC:       manageSubscriptionsUC,
		NotifyTokenCreationUC:       notifyTokenCreationUC,
		ProcessMeteoraEventUC:       processMeteoraEventUC,
		NotifyMeteoraPoolCreationUC: notifyMeteoraPoolCreationUC,
		SubscriptionRepo:            subscriptionRepo,
	}
}

// InitializeGRPCServer creates a fully configured gRPC server
func InitializeGRPCServer() *grpc.Server {
	wire.Build(
		ApplicationSet,
	)
	return nil
}
