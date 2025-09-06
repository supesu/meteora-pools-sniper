package solana

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/supesu/sniping-bot-v2/internal/adapters"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// WebSocketConnectionManager manages WebSocket connections with auto-reconnection
type WebSocketConnectionManager struct {
	*adapters.BaseConnectionManager

	config   *config.Config
	logger   logger.Logger
	wsConn   *websocket.Conn
	wsMu     sync.RWMutex
	wsDialer *websocket.Dialer
	endpoint string

	// Subscription management
	subscriptions map[string]interface{}
	subMu         sync.RWMutex
}

// NewWebSocketConnectionManager creates a new WebSocket connection manager
func NewWebSocketConnectionManager(cfg *config.Config, log logger.Logger, connConfig adapters.ConnectionConfig) *WebSocketConnectionManager {
	manager := &WebSocketConnectionManager{
		config:        cfg,
		logger:        log,
		wsDialer:      &websocket.Dialer{HandshakeTimeout: 45 * time.Second},
		endpoint:      cfg.Solana.WSEndpoint,
		subscriptions: make(map[string]interface{}),
	}

	connectFunc := func(ctx context.Context) error {
		return manager.connectWebSocket(ctx)
	}

	disconnectFunc := func() error {
		return manager.disconnectWebSocket()
	}

	healthCheckFunc := func(ctx context.Context) error {
		return manager.checkWebSocketHealth(ctx)
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

// connectWebSocket establishes the WebSocket connection
func (m *WebSocketConnectionManager) connectWebSocket(ctx context.Context) error {
	m.logger.Debug("WebSocket connectWebSocket method called")
	m.wsMu.Lock()
	defer m.wsMu.Unlock()

	if m.wsConn != nil {
		// Close existing connection
		if err := m.wsConn.Close(); err != nil {
			m.logger.WithError(err).Warn("Failed to close existing WebSocket connection")
		}
	}

	u, err := url.Parse(m.endpoint)
	if err != nil {
		return fmt.Errorf("invalid WebSocket endpoint: %w", err)
	}

	m.logger.WithField("ws_url", m.endpoint).Info("Connecting to WebSocket...")
	m.logger.WithField("parsed_url", u.String()).Debug("Parsed WebSocket URL")

	conn, _, err := m.wsDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	m.wsConn = conn
	m.logger.Info("WebSocket connection established successfully")

	// Re-subscribe to all previous subscriptions
	if err := m.resubscribeAll(); err != nil {
		m.logger.WithError(err).Warn("Failed to resubscribe to some subscriptions")
	}

	return nil
}

// disconnectWebSocket closes the WebSocket connection
func (m *WebSocketConnectionManager) disconnectWebSocket() error {
	m.wsMu.Lock()
	defer m.wsMu.Unlock()

	if m.wsConn != nil {
		err := m.wsConn.Close()
		m.wsConn = nil
		return err
	}
	return nil
}

// checkWebSocketHealth performs a health check on the WebSocket connection
func (m *WebSocketConnectionManager) checkWebSocketHealth(ctx context.Context) error {
	m.wsMu.RLock()
	conn := m.wsConn
	m.wsMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("WebSocket connection is nil")
	}

	// Try to set a read deadline to test if connection is alive
	if err := conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Reset the deadline
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		return fmt.Errorf("failed to reset read deadline: %w", err)
	}

	return nil
}

// GetConnection returns the WebSocket connection
func (m *WebSocketConnectionManager) GetConnection() *websocket.Conn {
	m.wsMu.RLock()
	defer m.wsMu.RUnlock()
	return m.wsConn
}

// SubscribeToProgramLogs subscribes to program logs via WebSocket
func (m *WebSocketConnectionManager) SubscribeToProgramLogs(programID string, subscriptionParams map[string]interface{}) error {
	conn := m.GetConnection()
	if conn == nil {
		return fmt.Errorf("WebSocket connection not established")
	}

	subscription := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(), // Unique ID for this subscription
		"method":  "logsSubscribe",
		"params":  []interface{}{subscriptionParams},
	}

	if err := conn.WriteJSON(subscription); err != nil {
		return fmt.Errorf("failed to subscribe to program logs: %w", err)
	}

	// Store subscription for reconnection
	m.subMu.Lock()
	m.subscriptions[programID] = subscriptionParams
	m.subMu.Unlock()

	m.logger.WithField("program_id", programID).Info("Subscribed to program logs")
	return nil
}

// ReadMessage reads a message from WebSocket
func (m *WebSocketConnectionManager) ReadMessage() ([]byte, error) {
	conn := m.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("WebSocket connection not established")
	}

	_, message, err := conn.ReadMessage()
	return message, err
}

// resubscribeAll re-subscribes to all stored subscriptions
func (m *WebSocketConnectionManager) resubscribeAll() error {
	m.subMu.RLock()
	subscriptions := make(map[string]interface{})
	for k, v := range m.subscriptions {
		subscriptions[k] = v
	}
	m.subMu.RUnlock()

	for programID, params := range subscriptions {
		if err := m.SubscribeToProgramLogs(programID, params.(map[string]interface{})); err != nil {
			m.logger.WithError(err).WithField("program_id", programID).Error("Failed to resubscribe")
			continue
		}
	}

	return nil
}

// ClearSubscriptions clears all stored subscriptions
func (m *WebSocketConnectionManager) ClearSubscriptions() {
	m.subMu.Lock()
	m.subscriptions = make(map[string]interface{})
	m.subMu.Unlock()
}
