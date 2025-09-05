package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MeteoraMetrics holds all Meteora-related Prometheus metrics
type MeteoraMetrics struct {
	// Pool creation metrics
	PoolsCreatedTotal    prometheus.Counter
	PoolsCreatedRate     prometheus.Gauge
	PoolCreationDuration prometheus.Histogram

	// Pool statistics
	ActivePoolsGauge        prometheus.Gauge
	TotalLiquidityUSD       prometheus.Gauge
	AverageLiquidityPerPool prometheus.Gauge

	// Token metrics
	UniqueTokensGauge  prometheus.Gauge
	TokenPairsGauge    prometheus.Gauge
	PopularTokensGauge *prometheus.GaugeVec

	// Creator metrics
	UniqueCreatorsGauge prometheus.Gauge
	TopCreatorsGauge    *prometheus.GaugeVec

	// Quality metrics
	SpamPoolsFiltered   prometheus.Counter
	ValidPoolsProcessed prometheus.Counter
	QualityFilterRate   prometheus.Gauge

	// Processing metrics
	EventsProcessedTotal    prometheus.Counter
	EventProcessingDuration prometheus.Histogram
	EventProcessingErrors   prometheus.Counter

	// Discord notification metrics
	NotificationsSentTotal prometheus.Counter
	NotificationErrors     prometheus.Counter
	NotificationDuration   prometheus.Histogram

	// Scanner health metrics
	ScannerHealthStatus  prometheus.Gauge
	LastEventTimestamp   prometheus.Gauge
	RPCConnectionStatus  prometheus.Gauge
	GRPCConnectionStatus prometheus.Gauge

	// Pool type distribution
	PoolTypeDistribution *prometheus.GaugeVec

	// Fee rate statistics
	AverageFeeRate      prometheus.Gauge
	FeeRateDistribution *prometheus.HistogramVec
}

// NewMeteoraMetrics creates and registers all Meteora-related metrics
func NewMeteoraMetrics() *MeteoraMetrics {
	return &MeteoraMetrics{
		// Pool creation metrics
		PoolsCreatedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "meteora_pools_created_total",
			Help: "Total number of Meteora pools created",
		}),

		PoolsCreatedRate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_pools_created_rate",
			Help: "Rate of Meteora pools created per hour",
		}),

		PoolCreationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "meteora_pool_creation_duration_seconds",
			Help:    "Time taken to process pool creation events",
			Buckets: prometheus.DefBuckets,
		}),

		// Pool statistics
		ActivePoolsGauge: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_active_pools_total",
			Help: "Total number of active Meteora pools",
		}),

		TotalLiquidityUSD: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_total_liquidity_usd",
			Help: "Total liquidity across all Meteora pools in USD",
		}),

		AverageLiquidityPerPool: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_average_liquidity_per_pool_usd",
			Help: "Average liquidity per pool in USD",
		}),

		// Token metrics
		UniqueTokensGauge: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_unique_tokens_total",
			Help: "Total number of unique tokens in Meteora pools",
		}),

		TokenPairsGauge: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_token_pairs_total",
			Help: "Total number of unique token pairs",
		}),

		PopularTokensGauge: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "meteora_popular_tokens_pool_count",
			Help: "Number of pools for popular tokens",
		}, []string{"token_symbol", "token_mint"}),

		// Creator metrics
		UniqueCreatorsGauge: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_unique_creators_total",
			Help: "Total number of unique pool creators",
		}),

		TopCreatorsGauge: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "meteora_top_creators_pool_count",
			Help: "Number of pools created by top creators",
		}, []string{"creator_address"}),

		// Quality metrics
		SpamPoolsFiltered: promauto.NewCounter(prometheus.CounterOpts{
			Name: "meteora_spam_pools_filtered_total",
			Help: "Total number of spam pools filtered out",
		}),

		ValidPoolsProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "meteora_valid_pools_processed_total",
			Help: "Total number of valid pools processed",
		}),

		QualityFilterRate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_quality_filter_rate",
			Help: "Percentage of pools that pass quality filters",
		}),

		// Processing metrics
		EventsProcessedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "meteora_events_processed_total",
			Help: "Total number of Meteora events processed",
		}),

		EventProcessingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "meteora_event_processing_duration_seconds",
			Help:    "Time taken to process Meteora events",
			Buckets: prometheus.DefBuckets,
		}),

		EventProcessingErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "meteora_event_processing_errors_total",
			Help: "Total number of event processing errors",
		}),

		// Discord notification metrics
		NotificationsSentTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "meteora_notifications_sent_total",
			Help: "Total number of Discord notifications sent",
		}),

		NotificationErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "meteora_notification_errors_total",
			Help: "Total number of Discord notification errors",
		}),

		NotificationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "meteora_notification_duration_seconds",
			Help:    "Time taken to send Discord notifications",
			Buckets: prometheus.DefBuckets,
		}),

		// Scanner health metrics
		ScannerHealthStatus: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_scanner_health_status",
			Help: "Health status of Meteora scanner (1 = healthy, 0 = unhealthy)",
		}),

		LastEventTimestamp: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_last_event_timestamp",
			Help: "Timestamp of the last processed event",
		}),

		RPCConnectionStatus: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_rpc_connection_status",
			Help: "RPC connection status (1 = connected, 0 = disconnected)",
		}),

		GRPCConnectionStatus: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_grpc_connection_status",
			Help: "gRPC connection status (1 = connected, 0 = disconnected)",
		}),

		// Pool type distribution
		PoolTypeDistribution: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "meteora_pool_type_distribution",
			Help: "Distribution of pools by type",
		}, []string{"pool_type"}),

		// Fee rate statistics
		AverageFeeRate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "meteora_average_fee_rate_basis_points",
			Help: "Average fee rate across all pools in basis points",
		}),

		FeeRateDistribution: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "meteora_fee_rate_distribution",
			Help:    "Distribution of fee rates across pools",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000}, // basis points
		}, []string{"pool_type"}),
	}
}

// RecordPoolCreated records a new pool creation event
func (m *MeteoraMetrics) RecordPoolCreated(poolType string, feeRate uint64, processingDuration float64) {
	m.PoolsCreatedTotal.Inc()
	m.ValidPoolsProcessed.Inc()
	m.PoolCreationDuration.Observe(processingDuration)
	m.PoolTypeDistribution.WithLabelValues(poolType).Inc()
	m.FeeRateDistribution.WithLabelValues(poolType).Observe(float64(feeRate))
}

// RecordSpamFiltered records a spam pool that was filtered out
func (m *MeteoraMetrics) RecordSpamFiltered() {
	m.SpamPoolsFiltered.Inc()
}

// RecordEventProcessed records a processed event
func (m *MeteoraMetrics) RecordEventProcessed(duration float64) {
	m.EventsProcessedTotal.Inc()
	m.EventProcessingDuration.Observe(duration)
}

// RecordEventError records an event processing error
func (m *MeteoraMetrics) RecordEventError() {
	m.EventProcessingErrors.Inc()
}

// RecordNotificationSent records a sent Discord notification
func (m *MeteoraMetrics) RecordNotificationSent(duration float64) {
	m.NotificationsSentTotal.Inc()
	m.NotificationDuration.Observe(duration)
}

// RecordNotificationError records a Discord notification error
func (m *MeteoraMetrics) RecordNotificationError() {
	m.NotificationErrors.Inc()
}

// UpdatePoolStats updates pool statistics
func (m *MeteoraMetrics) UpdatePoolStats(activePools int, totalLiquidityUSD, avgLiquidityUSD float64) {
	m.ActivePoolsGauge.Set(float64(activePools))
	m.TotalLiquidityUSD.Set(totalLiquidityUSD)
	m.AverageLiquidityPerPool.Set(avgLiquidityUSD)
}

// UpdateTokenStats updates token statistics
func (m *MeteoraMetrics) UpdateTokenStats(uniqueTokens, tokenPairs int) {
	m.UniqueTokensGauge.Set(float64(uniqueTokens))
	m.TokenPairsGauge.Set(float64(tokenPairs))
}

// UpdateCreatorStats updates creator statistics
func (m *MeteoraMetrics) UpdateCreatorStats(uniqueCreators int) {
	m.UniqueCreatorsGauge.Set(float64(uniqueCreators))
}

// UpdateHealthStatus updates scanner health metrics
func (m *MeteoraMetrics) UpdateHealthStatus(healthy bool, lastEventTime float64) {
	if healthy {
		m.ScannerHealthStatus.Set(1)
	} else {
		m.ScannerHealthStatus.Set(0)
	}
	m.LastEventTimestamp.Set(lastEventTime)
}

// UpdateConnectionStatus updates connection status metrics
func (m *MeteoraMetrics) UpdateConnectionStatus(rpcConnected, grpcConnected bool) {
	if rpcConnected {
		m.RPCConnectionStatus.Set(1)
	} else {
		m.RPCConnectionStatus.Set(0)
	}

	if grpcConnected {
		m.GRPCConnectionStatus.Set(1)
	} else {
		m.GRPCConnectionStatus.Set(0)
	}
}

// UpdateQualityFilterRate updates the quality filter success rate
func (m *MeteoraMetrics) UpdateQualityFilterRate(rate float64) {
	m.QualityFilterRate.Set(rate)
}

// UpdateFeeRateStats updates fee rate statistics
func (m *MeteoraMetrics) UpdateFeeRateStats(averageFeeRate float64) {
	m.AverageFeeRate.Set(averageFeeRate)
}

// UpdatePopularTokens updates popular token metrics
func (m *MeteoraMetrics) UpdatePopularTokens(tokenStats map[string]map[string]int) {
	// Reset all existing metrics
	m.PopularTokensGauge.Reset()

	// Update with new data
	for tokenSymbol, mints := range tokenStats {
		for tokenMint, poolCount := range mints {
			m.PopularTokensGauge.WithLabelValues(tokenSymbol, tokenMint).Set(float64(poolCount))
		}
	}
}

// UpdateTopCreators updates top creator metrics
func (m *MeteoraMetrics) UpdateTopCreators(creatorStats map[string]int) {
	// Reset all existing metrics
	m.TopCreatorsGauge.Reset()

	// Update with new data (top 10 creators)
	count := 0
	for creatorAddress, poolCount := range creatorStats {
		if count >= 10 {
			break
		}
		m.TopCreatorsGauge.WithLabelValues(creatorAddress).Set(float64(poolCount))
		count++
	}
}
