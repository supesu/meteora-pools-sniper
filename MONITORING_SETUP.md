# Meteora Pool Monitoring Dashboard

## Overview

This document describes the comprehensive monitoring solution for the Meteora pool scanning system, including Prometheus metrics collection and Grafana visualization.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Meteora        │    │  gRPC Server    │    │  Prometheus     │
│  Scanner        │───▶│                 │───▶│                 │
│  :9092/metrics  │    │  :9091/metrics  │    │  :9090          │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
┌─────────────────┐    ┌─────────────────┐           │
│  Node Exporter  │    │  Grafana        │◀──────────┘
│  :9100/metrics  │───▶│  :3000          │
└─────────────────┘    └─────────────────┘
```

## Components

### 1. Prometheus Metrics Collection

#### Meteora Scanner Metrics (Port 9092)
- **Pool Creation Metrics**
  - `meteora_pools_created_total` - Total pools created
  - `meteora_pools_created_rate` - Pool creation rate per hour
  - `meteora_pool_creation_duration_seconds` - Pool processing time

- **Pool Statistics**
  - `meteora_active_pools_total` - Active pools count
  - `meteora_total_liquidity_usd` - Total liquidity in USD
  - `meteora_average_liquidity_per_pool_usd` - Average liquidity per pool

- **Token Metrics**
  - `meteora_unique_tokens_total` - Unique tokens count
  - `meteora_token_pairs_total` - Token pairs count
  - `meteora_popular_tokens_pool_count` - Pool count by token

- **Creator Metrics**
  - `meteora_unique_creators_total` - Unique creators count
  - `meteora_top_creators_pool_count` - Pool count by creator

- **Quality Metrics**
  - `meteora_spam_pools_filtered_total` - Spam pools filtered
  - `meteora_valid_pools_processed_total` - Valid pools processed
  - `meteora_quality_filter_rate` - Quality filter success rate

- **Processing Metrics**
  - `meteora_events_processed_total` - Total events processed
  - `meteora_event_processing_duration_seconds` - Processing time
  - `meteora_event_processing_errors_total` - Processing errors

- **Discord Notification Metrics**
  - `meteora_notifications_sent_total` - Notifications sent
  - `meteora_notification_errors_total` - Notification errors
  - `meteora_notification_duration_seconds` - Notification time

- **Health Metrics**
  - `meteora_scanner_health_status` - Scanner health (1=healthy, 0=unhealthy)
  - `meteora_last_event_timestamp` - Last event timestamp
  - `meteora_rpc_connection_status` - RPC connection status
  - `meteora_grpc_connection_status` - gRPC connection status

#### gRPC Server Metrics (Port 9091)
- Standard HTTP server metrics via `/metrics` endpoint
- Request duration, error rates, and throughput

#### System Metrics (Port 9100)
- CPU, memory, disk, and network metrics via Node Exporter

### 2. Grafana Dashboard

The Meteora dashboard includes:

#### Pool Overview
- **Pool Statistics Panel** - Line chart showing total and active pools
- **Total Liquidity Panel** - Stat panel showing total USD liquidity
- **Processing Rates Panel** - Line chart of pool creation and event processing rates
- **Scanner Health Panel** - Status indicator for scanner health

#### Pool Analytics
- **Pool Type Distribution** - Pie chart of pool types
- **Popular Tokens** - Pie chart showing token distribution by pool count
- **Event Processing Duration** - Line chart with 95th and 50th percentiles
- **Discord Notifications** - Line chart of notifications sent vs errors

#### Key Metrics
- **Quality Filter Success Rate** - Stat panel showing filter effectiveness
- **Unique Tokens** - Count of unique tokens in pools
- **Unique Creators** - Count of unique pool creators
- **Average Fee Rate** - Average fee rate in basis points

## Setup Instructions

### 1. Start the Monitoring Stack

```bash
# Start all services including monitoring
docker-compose up -d

# Or start only monitoring services
docker-compose up -d prometheus grafana node-exporter
```

### 2. Access the Dashboard

1. **Grafana**: http://localhost:3000
   - Username: `admin`
   - Password: `meteora123`

2. **Prometheus**: http://localhost:9090
   - Raw metrics and query interface

3. **Meteora Scanner Metrics**: http://localhost:9092/metrics
   - Direct access to scanner metrics

4. **gRPC Server Metrics**: http://localhost:9091/metrics
   - Direct access to server metrics

### 3. Dashboard Navigation

The Meteora dashboard is automatically provisioned and available at:
- **Dashboard Name**: "Meteora Pool Statistics Dashboard"
- **Folder**: "Meteora"
- **Tags**: meteora, solana, defi, pools

## Key Metrics to Monitor

### 🚨 **Critical Alerts**

1. **Scanner Health** - Should always be 1 (healthy)
2. **Event Processing Errors** - Should be minimal
3. **RPC/gRPC Connection Status** - Should be 1 (connected)
4. **Quality Filter Rate** - Should be > 90%

### 📊 **Performance Metrics**

1. **Pool Creation Rate** - Trending upward indicates growth
2. **Processing Duration** - Should be < 1 second
3. **Total Liquidity** - Indicates market activity
4. **Notification Success Rate** - Should be > 95%

### 📈 **Business Metrics**

1. **Active Pools** - Total pools being tracked
2. **Unique Tokens** - Token diversity
3. **Top Creators** - Most active pool creators
4. **Popular Token Pairs** - Market favorites

## Alerting Rules

### Prometheus Alerting Rules (Optional)

Create `monitoring/alerts.yml`:

```yaml
groups:
  - name: meteora-alerts
    rules:
      - alert: MeteoraScanner Down
        expr: meteora_scanner_health_status == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Meteora scanner is unhealthy"
          
      - alert: High Processing Errors
        expr: rate(meteora_event_processing_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High event processing error rate"
          
      - alert: Low Quality Filter Rate
        expr: meteora_quality_filter_rate < 0.8
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Quality filter rate below 80%"
```

## Troubleshooting

### Common Issues

1. **Metrics not appearing**
   - Check service ports are exposed (9091, 9092)
   - Verify Prometheus configuration targets
   - Check service logs for startup errors

2. **Dashboard not loading**
   - Verify Grafana provisioning volumes are mounted
   - Check Grafana logs for provisioning errors
   - Ensure Prometheus datasource is configured

3. **Connection errors**
   - Verify Docker network connectivity
   - Check container names in prometheus.yml match docker-compose services
   - Ensure all services are in the same network

### Logs and Debugging

```bash
# Check service logs
docker logs sniping-bot-solana-scanner
docker logs sniping-bot-grpc-server
docker logs sniping-bot-prometheus
docker logs sniping-bot-grafana

# Check metrics endpoints
curl http://localhost:9092/metrics  # Scanner metrics
curl http://localhost:9091/metrics  # Server metrics
curl http://localhost:9090/targets  # Prometheus targets
```

## Data Retention

- **Prometheus**: 200 hours (8+ days) of metrics data
- **Grafana**: Persistent dashboards and configurations
- **Volumes**: `prometheus_data` and `grafana_data` for persistence

## Security Considerations

1. **Grafana Access**: Change default password in production
2. **Metrics Endpoints**: Consider adding authentication for production
3. **Network Security**: Use proper firewall rules for metric ports
4. **Data Privacy**: Metrics don't contain sensitive pool or user data

## Performance Impact

- **Metrics Collection**: ~1-2% CPU overhead
- **Storage**: ~10MB/day for metrics data
- **Network**: Minimal impact (~1KB/s per scrape)

## Customization

### Adding New Metrics

1. Add metric definition to `pkg/metrics/meteora_metrics.go`
2. Update recording logic in scanner/server
3. Add panels to Grafana dashboard JSON
4. Rebuild and restart services

### Dashboard Modifications

1. Edit `monitoring/grafana/dashboards/meteora-dashboard.json`
2. Or use Grafana UI and export JSON
3. Restart Grafana to reload provisioned dashboards

## Production Recommendations

1. **External Grafana**: Use managed Grafana service for production
2. **Alert Manager**: Set up Prometheus AlertManager for notifications
3. **Backup**: Regular backup of Grafana configurations and Prometheus data
4. **Monitoring**: Monitor the monitoring stack itself
5. **Scaling**: Consider Prometheus federation for multi-instance deployments

---

## Quick Start Commands

```bash
# Start everything
docker-compose up -d

# View logs
docker-compose logs -f meteora-scanner

# Stop monitoring
docker-compose stop prometheus grafana node-exporter

# Reset metrics data
docker-compose down -v
docker-compose up -d
```

The monitoring solution provides comprehensive visibility into the Meteora pool scanning system with real-time metrics, historical trends, and alerting capabilities.
