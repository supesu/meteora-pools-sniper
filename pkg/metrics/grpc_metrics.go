package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GRPCMetrics holds all gRPC server metrics
type GRPCMetrics struct {
	// Server uptime and health
	UptimeSeconds        prometheus.Gauge
	HealthStatus         prometheus.Gauge
	RequestsTotal        *prometheus.CounterVec
	RequestDuration      *prometheus.HistogramVec
	ErrorsTotal          *prometheus.CounterVec
	ActiveConnections    prometheus.Gauge
	ProcessedEventsTotal *prometheus.CounterVec

	// Meteora-specific metrics
	MeteoraEventsProcessed   prometheus.Counter
	MeteoraProcessingErrors  prometheus.Counter
	MeteoraPoolsStored       prometheus.Counter
	MeteoraNotificationsSent prometheus.Counter

	startTime time.Time
}

// NewGRPCMetrics creates a new instance of gRPC metrics
func NewGRPCMetrics() *GRPCMetrics {
	return &GRPCMetrics{
		UptimeSeconds: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "grpc_server_uptime_seconds",
			Help: "How long the gRPC server has been running in seconds",
		}),
		HealthStatus: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "grpc_server_health_status",
			Help: "Health status of the gRPC server (1 = healthy, 0 = unhealthy)",
		}),
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC requests processed",
		}, []string{"method", "status"}),
		RequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
		ErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_server_errors_total",
			Help: "Total number of gRPC errors",
		}, []string{"method", "error_type"}),
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "grpc_server_active_connections",
			Help: "Number of active gRPC connections",
		}),
		ProcessedEventsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_server_processed_events_total",
			Help: "Total number of events processed by type",
		}, []string{"event_type"}),

		// Meteora-specific metrics
		MeteoraEventsProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "grpc_meteora_events_processed_total",
			Help: "Total number of Meteora events processed by gRPC server",
		}),
		MeteoraProcessingErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "grpc_meteora_processing_errors_total",
			Help: "Total number of Meteora event processing errors",
		}),
		MeteoraPoolsStored: promauto.NewCounter(prometheus.CounterOpts{
			Name: "grpc_meteora_pools_stored_total",
			Help: "Total number of Meteora pools stored via gRPC server",
		}),
		MeteoraNotificationsSent: promauto.NewCounter(prometheus.CounterOpts{
			Name: "grpc_meteora_notifications_sent_total",
			Help: "Total number of Meteora Discord notifications sent via gRPC server",
		}),

		startTime: time.Now(),
	}
}

// RecordRequest records metrics for a gRPC request
func (m *GRPCMetrics) RecordRequest(method, status string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(method, status).Inc()
	m.RequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordError records an error metric
func (m *GRPCMetrics) RecordError(method, errorType string) {
	m.ErrorsTotal.WithLabelValues(method, errorType).Inc()
}

// IncrementActiveConnections increments the active connections counter
func (m *GRPCMetrics) IncrementActiveConnections() {
	m.ActiveConnections.Inc()
}

// DecrementActiveConnections decrements the active connections counter
func (m *GRPCMetrics) DecrementActiveConnections() {
	m.ActiveConnections.Dec()
}

// RecordProcessedEvent records a processed event
func (m *GRPCMetrics) RecordProcessedEvent(eventType string) {
	m.ProcessedEventsTotal.WithLabelValues(eventType).Inc()
}

// RecordMeteoraEvent records Meteora-specific metrics
func (m *GRPCMetrics) RecordMeteoraEvent() {
	m.MeteoraEventsProcessed.Inc()
}

// RecordMeteoraError records a Meteora processing error
func (m *GRPCMetrics) RecordMeteoraError() {
	m.MeteoraProcessingErrors.Inc()
}

// RecordMeteoraPoolStored records when a Meteora pool is stored
func (m *GRPCMetrics) RecordMeteoraPoolStored() {
	m.MeteoraPoolsStored.Inc()
}

// RecordMeteoraNotificationSent records when a Meteora notification is sent
func (m *GRPCMetrics) RecordMeteoraNotificationSent() {
	m.MeteoraNotificationsSent.Inc()
}

// UpdateUptime updates the uptime metric
func (m *GRPCMetrics) UpdateUptime() {
	m.UptimeSeconds.Set(time.Since(m.startTime).Seconds())
}

// SetHealthStatus sets the health status
func (m *GRPCMetrics) SetHealthStatus(healthy bool) {
	if healthy {
		m.HealthStatus.Set(1)
	} else {
		m.HealthStatus.Set(0)
	}
}

// StartUptimeTracker starts a goroutine to update uptime metrics periodically
func (m *GRPCMetrics) StartUptimeTracker() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			m.UpdateUptime()
		}
	}()
}
