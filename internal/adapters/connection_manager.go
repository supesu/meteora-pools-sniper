package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// ConnectionState represents the state of a connection
type ConnectionState int

const (
	// StateDisconnected indicates the connection is not established
	StateDisconnected ConnectionState = iota
	// StateConnecting indicates the connection is being established
	StateConnecting
	// StateConnected indicates the connection is established and healthy
	StateConnected
	// StateReconnecting indicates the connection is being re-established
	StateReconnecting
	// StateFailed indicates the connection has failed and won't retry
	StateFailed
)

// ConnectionManager defines the interface for managing connections with auto-reconnection
type ConnectionManager interface {
	// Connect establishes the connection
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect() error

	// GetState returns the current connection state
	GetState() ConnectionState

	// IsHealthy checks if the connection is healthy
	IsHealthy(ctx context.Context) error

	// StartAutoReconnect starts the auto-reconnection loop
	StartAutoReconnect(ctx context.Context)

	// StopAutoReconnect stops the auto-reconnection loop
	StopAutoReconnect()
}

// BaseConnectionManager provides common functionality for connection managers
type BaseConnectionManager struct {
	logger              logger.Logger
	state               ConnectionState
	maxRetries          int
	baseRetryDelay      time.Duration
	maxRetryDelay       time.Duration
	healthCheckInterval time.Duration

	connectFunc     func(ctx context.Context) error
	disconnectFunc  func() error
	healthCheckFunc func(ctx context.Context) error

	ctx           context.Context
	cancel        context.CancelFunc
	reconnectChan chan struct{}
}

// NewBaseConnectionManager creates a new base connection manager
func NewBaseConnectionManager(
	logger logger.Logger,
	maxRetries int,
	baseRetryDelay time.Duration,
	maxRetryDelay time.Duration,
	healthCheckInterval time.Duration,
	connectFunc func(ctx context.Context) error,
	disconnectFunc func() error,
	healthCheckFunc func(ctx context.Context) error,
) *BaseConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &BaseConnectionManager{
		logger:              logger,
		state:               StateDisconnected,
		maxRetries:          maxRetries,
		baseRetryDelay:      baseRetryDelay,
		maxRetryDelay:       maxRetryDelay,
		healthCheckInterval: healthCheckInterval,
		connectFunc:         connectFunc,
		disconnectFunc:      disconnectFunc,
		healthCheckFunc:     healthCheckFunc,
		ctx:                 ctx,
		cancel:              cancel,
		reconnectChan:       make(chan struct{}, 1),
	}
}

// Connect implements ConnectionManager.Connect
func (m *BaseConnectionManager) Connect(ctx context.Context) error {
	m.setState(StateConnecting)

	if err := m.connectFunc(ctx); err != nil {
		m.setState(StateDisconnected)
		return err
	}

	m.setState(StateConnected)
	return nil
}

// Disconnect implements ConnectionManager.Disconnect
func (m *BaseConnectionManager) Disconnect() error {
	m.setState(StateDisconnected)

	if m.disconnectFunc != nil {
		return m.disconnectFunc()
	}
	return nil
}

// GetState implements ConnectionManager.GetState
func (m *BaseConnectionManager) GetState() ConnectionState {
	return m.state
}

// IsHealthy implements ConnectionManager.IsHealthy
func (m *BaseConnectionManager) IsHealthy(ctx context.Context) error {
	if m.state != StateConnected {
		return fmt.Errorf("connection is not in connected state")
	}

	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

// StartAutoReconnect implements ConnectionManager.StartAutoReconnect
func (m *BaseConnectionManager) StartAutoReconnect(ctx context.Context) {
	go m.autoReconnectLoop(ctx)
	go m.healthCheckLoop(ctx)
}

// StopAutoReconnect implements ConnectionManager.StopAutoReconnect
func (m *BaseConnectionManager) StopAutoReconnect() {
	m.cancel()
}

// autoReconnectLoop handles the auto-reconnection logic
func (m *BaseConnectionManager) autoReconnectLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.reconnectChan:
			if err := m.performReconnection(ctx); err != nil {
				m.logger.WithError(err).Error("Auto-reconnection failed")
			}
		}
	}
}

// healthCheckLoop periodically checks connection health
func (m *BaseConnectionManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if m.state == StateConnected {
				if err := m.IsHealthy(ctx); err != nil {
					m.logger.WithError(err).Warn("Connection health check failed, triggering reconnection")
					m.triggerReconnection()
				}
			}
		}
	}
}

// performReconnection attempts to reconnect with exponential backoff
func (m *BaseConnectionManager) performReconnection(ctx context.Context) error {
	if m.state == StateReconnecting || m.state == StateConnecting {
		return nil // Already reconnecting
	}

	m.setState(StateReconnecting)

	retryDelay := m.baseRetryDelay
	for attempt := 1; attempt <= m.maxRetries; attempt++ {
		m.logger.WithField("attempt", attempt).Info("Attempting to reconnect...")

		if err := m.connectFunc(ctx); err != nil {
			m.logger.WithError(err).WithField("attempt", attempt).Warn("Reconnection attempt failed")

			if attempt < m.maxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryDelay):
					// Exponential backoff with max delay
					retryDelay *= 2
					if retryDelay > m.maxRetryDelay {
						retryDelay = m.maxRetryDelay
					}
				}
				continue
			}

			m.setState(StateFailed)
			return fmt.Errorf("failed to reconnect after %d attempts: %w", m.maxRetries, err)
		}

		m.setState(StateConnected)
		m.logger.Info("Successfully reconnected")
		return nil
	}

	return fmt.Errorf("reconnection failed")
}

// TriggerReconnection triggers a reconnection attempt
func (m *BaseConnectionManager) TriggerReconnection() {
	select {
	case m.reconnectChan <- struct{}{}:
	default:
		// Channel is full, reconnection already queued
	}
}

// triggerReconnection triggers a reconnection attempt (internal method)
func (m *BaseConnectionManager) triggerReconnection() {
	m.TriggerReconnection()
}

// setState updates the connection state thread-safely
func (m *BaseConnectionManager) setState(state ConnectionState) {
	m.state = state
}

// ConnectionConfig holds configuration for connection managers
type ConnectionConfig struct {
	MaxRetries          int
	BaseRetryDelay      time.Duration
	MaxRetryDelay       time.Duration
	HealthCheckInterval time.Duration
}

// DefaultConnectionConfig returns default connection configuration
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		MaxRetries:          10,
		BaseRetryDelay:      1 * time.Second,
		MaxRetryDelay:       30 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
}
