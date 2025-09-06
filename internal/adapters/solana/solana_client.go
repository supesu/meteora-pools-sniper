package solana

import (
	"context"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gorilla/websocket"
	"github.com/supesu/sniping-bot-v2/internal/adapters"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// SolanaClient handles Solana RPC and WebSocket connections
type SolanaClient struct {
	config    *config.Config
	logger    logger.Logger
	rpcClient *rpc.Client

	wsConn              *websocket.Conn
	wsMu                sync.Mutex
	wsDialer            *websocket.Dialer
	wsConnectionManager *WebSocketConnectionManager

	ctx    context.Context
	cancel context.CancelFunc
}

// NewSolanaClient creates a new Solana client
func NewSolanaClient(cfg *config.Config, log logger.Logger) *SolanaClient {
	ctx, cancel := context.WithCancel(context.Background())

	client := &SolanaClient{
		config: cfg,
		logger: log,
		wsDialer: &websocket.Dialer{
			HandshakeTimeout: 45 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize WebSocket connection manager with default config
	connConfig := adapters.DefaultConnectionConfig()
	client.wsConnectionManager = NewWebSocketConnectionManager(cfg, log, connConfig)

	return client
}

// ConnectRPC establishes RPC connection
func (c *SolanaClient) ConnectRPC() error {
	c.rpcClient = rpc.New(c.config.Solana.RPC)
	c.logger.WithField("rpc_url", c.config.Solana.RPC).Info("RPC client initialized")
	return nil
}

// GetRPCClient returns the RPC client
func (c *SolanaClient) GetRPCClient() *rpc.Client {
	return c.rpcClient
}

// ConnectWebSocket establishes WebSocket connection with auto-reconnection
func (c *SolanaClient) ConnectWebSocket() error {
	c.logger.Info("Attempting to establish WebSocket connection...")

	if err := c.wsConnectionManager.Connect(context.Background()); err != nil {
		c.logger.WithError(err).Error("Failed to establish WebSocket connection")
		return err
	}

	c.logger.Info("WebSocket connection established successfully, starting auto-reconnection")

	// Start auto-reconnection
	c.wsConnectionManager.StartAutoReconnect(c.ctx)

	return nil
}

// GetWebSocketConn returns the WebSocket connection
func (c *SolanaClient) GetWebSocketConn() *websocket.Conn {
	return c.wsConnectionManager.GetConnection()
}

// CloseWebSocket closes the WebSocket connection and stops auto-reconnection
func (c *SolanaClient) CloseWebSocket() error {
	c.wsConnectionManager.StopAutoReconnect()
	return c.wsConnectionManager.Disconnect()
}

// SubscribeToProgramLogs subscribes to program logs via WebSocket
func (c *SolanaClient) SubscribeToProgramLogs(programID solana.PublicKey) error {
	subscriptionParams := map[string]interface{}{
		"mentions":   []string{programID.String()},
		"commitment": "confirmed",
	}

	return c.wsConnectionManager.SubscribeToProgramLogs(programID.String(), subscriptionParams)
}

// ReadWebSocketMessage reads a message from WebSocket
func (c *SolanaClient) ReadWebSocketMessage() ([]byte, error) {
	return c.wsConnectionManager.ReadMessage()
}

// Stop stops the client and closes connections
func (c *SolanaClient) Stop() {
	c.cancel()
	c.CloseWebSocket()
}
