//go:build wireinject
// +build wireinject

package usecase

import (
	"github.com/google/wire"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// UseCaseSet provides all use case implementations
var UseCaseSet = wire.NewSet(
	NewProcessTransactionUseCase,
	NewGetTransactionHistoryUseCase,
	NewManageSubscriptionsUseCase,
	NewNotifyTokenCreationUseCase,
	NewProcessMeteoraEventUseCase,
	NewNotifyMeteoraPoolCreationUseCase,
)

// ProcessTransactionUseCaseProvider provides process transaction use case
func ProcessTransactionUseCaseProvider(
	transactionRepo domain.TransactionRepository,
	eventPublisher domain.EventPublisher,
	log logger.Logger,
) *ProcessTransactionUseCase {
	wire.Build(UseCaseSet)
	return nil
}

// GetTransactionHistoryUseCaseProvider provides get transaction history use case
func GetTransactionHistoryUseCaseProvider(
	transactionRepo domain.TransactionRepository,
	log logger.Logger,
) *GetTransactionHistoryUseCase {
	wire.Build(UseCaseSet)
	return nil
}

// ManageSubscriptionsUseCaseProvider provides manage subscriptions use case
func ManageSubscriptionsUseCaseProvider(
	subscriptionRepo domain.SubscriptionRepository,
	eventPublisher domain.EventPublisher,
	log logger.Logger,
) *ManageSubscriptionsUseCase {
	wire.Build(UseCaseSet)
	return nil
}

// NotifyTokenCreationUseCaseProvider provides notify token creation use case
func NotifyTokenCreationUseCaseProvider(
	discordRepo domain.DiscordNotificationRepository,
	log logger.Logger,
) *NotifyTokenCreationUseCase {
	wire.Build(UseCaseSet)
	return nil
}

// ProcessMeteoraEventUseCaseProvider provides process Meteora event use case
func ProcessMeteoraEventUseCaseProvider(
	meteoraRepo domain.MeteoraRepository,
	eventPublisher domain.EventPublisher,
	log logger.Logger,
) *ProcessMeteoraEventUseCase {
	wire.Build(UseCaseSet)
	return nil
}

// NotifyMeteoraPoolCreationUseCaseProvider provides notify Meteora pool creation use case
func NotifyMeteoraPoolCreationUseCaseProvider(
	discordRepo domain.DiscordNotificationRepository,
	log logger.Logger,
) *NotifyMeteoraPoolCreationUseCase {
	wire.Build(UseCaseSet)
	return nil
}
