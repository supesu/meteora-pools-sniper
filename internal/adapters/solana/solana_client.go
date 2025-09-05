package solana

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gorilla/websocket"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// SolanaClient handles Solana RPC and WebSocket connections
type SolanaClient struct {
	config    *config.Config
	logger    logger.Logger
	rpcClient *rpc.Client

	wsConn   *websocket.Conn
	wsMu     sync.Mutex
	wsDialer *websocket.Dialer

	ctx    context.Context
	cancel context.CancelFunc
}

// NewSolanaClient creates a new Solana client
func NewSolanaClient(cfg *config.Config, log logger.Logger) *SolanaClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &SolanaClient{
		config: cfg,
		logger: log,
		wsDialer: &websocket.Dialer{
			HandshakeTimeout: 45 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}
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

// ConnectWebSocket establishes WebSocket connection
func (c *SolanaClient) ConnectWebSocket() error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()

	if c.wsConn != nil {
		return nil // Already connected
	}

	u, err := url.Parse(c.config.Solana.WSEndpoint)
	if err != nil {
		return fmt.Errorf("invalid WebSocket endpoint: %w", err)
	}

	c.logger.WithField("ws_url", c.config.Solana.WSEndpoint).Info("Connecting to WebSocket...")

	conn, _, err := c.wsDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.wsConn = conn
	c.logger.Info("WebSocket connection established")
	return nil
}

// GetWebSocketConn returns the WebSocket connection
func (c *SolanaClient) GetWebSocketConn() *websocket.Conn {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	return c.wsConn
}

// CloseWebSocket closes the WebSocket connection
func (c *SolanaClient) CloseWebSocket() error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()

	if c.wsConn != nil {
		err := c.wsConn.Close()
		c.wsConn = nil
		return err
	}
	return nil
}

// SubscribeToProgramLogs subscribes to program logs via WebSocket
func (c *SolanaClient) SubscribeToProgramLogs(programID solana.PublicKey) error {
	conn := c.GetWebSocketConn()
	if conn == nil {
		return fmt.Errorf("WebSocket connection not established")
	}

	subscription := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "logsSubscribe",
		"params": []interface{}{
			map[string]interface{}{
				"mentions": []string{programID.String()},
			},
			map[string]interface{}{
				"commitment": "confirmed",
			},
		},
	}

	if err := conn.WriteJSON(subscription); err != nil {
		return fmt.Errorf("failed to subscribe to program logs: %w", err)
	}

	c.logger.WithField("program_id", programID.String()).Info("Subscribed to program logs")
	return nil
}

// ReadWebSocketMessage reads a message from WebSocket
func (c *SolanaClient) ReadWebSocketMessage() ([]byte, error) {
	conn := c.GetWebSocketConn()
	if conn == nil {
		return nil, fmt.Errorf("WebSocket connection not established")
	}

	_, message, err := conn.ReadMessage()
	return message, err
}

// Stop stops the client and closes connections
func (c *SolanaClient) Stop() {
	c.cancel()
	c.CloseWebSocket()
}
