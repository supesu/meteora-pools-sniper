package solana

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/supesu/sniping-bot-v2/internal/adapters"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCConnectionManager manages gRPC connections with auto-reconnection
type GRPCConnectionManager struct {
	*adapters.BaseConnectionManager

	config     *config.Config
	logger     logger.Logger
	grpcClient interface{} // Will be cast to specific client type
	grpcConn   *grpc.ClientConn
	connMu     sync.RWMutex
	address    string
}

// NewGRPCConnectionManager creates a new gRPC connection manager
func NewGRPCConnectionManager(cfg *config.Config, log logger.Logger, connConfig adapters.ConnectionConfig) *GRPCConnectionManager {
	manager := &GRPCConnectionManager{
		config:  cfg,
		logger:  log,
		address: cfg.Address(),
	}

	connectFunc := func(ctx context.Context) error {
		return manager.connectGRPC(ctx)
	}

	disconnectFunc := func() error {
		return manager.disconnectGRPC()
	}

	healthCheckFunc := func(ctx context.Context) error {
		return manager.checkGRPCHealth(ctx)
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

// connectGRPC establishes the gRPC connection
func (m *GRPCConnectionManager) connectGRPC(ctx context.Context) error {
	m.connMu.Lock()
	defer m.connMu.Unlock()

	if m.grpcConn != nil {
		// Close existing connection
		if err := m.grpcConn.Close(); err != nil {
			m.logger.WithError(err).Warn("Failed to close existing gRPC connection")
		}
	}

	m.logger.WithField("address", m.address).Info("Connecting to gRPC server...")

	conn, err := grpc.NewClient(
		m.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to establish gRPC connection: %w", err)
	}

	m.grpcConn = conn

	// Wait for connection to be ready
	if err := m.waitForConnectionReady(ctx); err != nil {
		conn.Close()
		m.grpcConn = nil
		return fmt.Errorf("gRPC connection not ready: %w", err)
	}

	m.logger.WithField("address", m.address).Info("gRPC connection established successfully")
	return nil
}

// waitForConnectionReady waits for the gRPC connection to be ready
func (m *GRPCConnectionManager) waitForConnectionReady(ctx context.Context) error {
	state := m.grpcConn.GetState()
	if state == connectivity.Ready {
		return nil
	}

	// Wait for state change with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for gRPC connection to be ready")
		default:
			if m.grpcConn.WaitForStateChange(ctx, state) {
				state = m.grpcConn.GetState()
				if state == connectivity.Ready {
					return nil
				}
				if state == connectivity.Shutdown {
					return fmt.Errorf("gRPC connection shut down")
				}
			}
		}
	}
}

// disconnectGRPC closes the gRPC connection
func (m *GRPCConnectionManager) disconnectGRPC() error {
	m.connMu.Lock()
	defer m.connMu.Unlock()

	if m.grpcConn != nil {
		err := m.grpcConn.Close()
		m.grpcConn = nil
		m.grpcClient = nil
		return err
	}
	return nil
}

// checkGRPCHealth performs a health check on the gRPC connection
func (m *GRPCConnectionManager) checkGRPCHealth(ctx context.Context) error {
	m.connMu.RLock()
	conn := m.grpcConn
	m.connMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("gRPC connection is nil")
	}

	state := conn.GetState()
	if state != connectivity.Ready {
		return fmt.Errorf("gRPC connection not ready, state: %s", state)
	}

	return nil
}

// GetConnection returns the gRPC connection
func (m *GRPCConnectionManager) GetConnection() *grpc.ClientConn {
	m.connMu.RLock()
	defer m.connMu.RUnlock()
	return m.grpcConn
}

// IsConnected returns whether the gRPC client is connected
func (m *GRPCConnectionManager) IsConnected() bool {
	return m.GetConnection() != nil
}
