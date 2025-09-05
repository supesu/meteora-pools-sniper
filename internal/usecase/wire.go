//go:build wireinject
// +build wireinject

package usecase

import (
	"github.com/google/wire"
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
