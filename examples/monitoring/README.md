# QUIC Backbone Monitoring Example

This example demonstrates how to use the monitoring features of the QUIC Backbone Network.

## Features Demonstrated

1. **Event Hooks**: Real-time event notifications
   - Client connect/disconnect events
   - Heartbeat timeout events
   - Message send/receive events
   - Reconnection events

2. **Metrics Collection**: Comprehensive system metrics
   - Connection metrics (connected clients, total connections)
   - Message metrics (sent/received, throughput, bytes)
   - Latency metrics (average, P50, P95, P99)
   - Heartbeat metrics (sent/received, timeouts)
   - Promise metrics (created/completed/timeout, active count)
   - Error metrics (encoding/decoding/network errors)
   - System metrics (uptime)

3. **Prometheus Export**: HTTP endpoint for Prometheus scraping
   - Text format metrics endpoint at `/metrics`
   - Compatible with Prometheus server

## Usage

### Build

```bash
go build -o monitoring-server ./examples/monitoring
```

### Run

```bash
# Basic usage (metrics enabled on port 9090)
./monitoring-server

# Custom addresses
./monitoring-server -addr :8080 -metrics :9091

# Disable metrics endpoint
./monitoring-server -enable-metrics=false

# Custom TLS certificates
./monitoring-server -cert path/to/cert.crt -key path/to/key.key
```

### Flags

- `-addr`: QUIC server address (default: `:8474`)
- `-cert`: TLS certificate file (default: `certs/server.crt`)
- `-key`: TLS key file (default: `certs/server.key`)
- `-metrics`: Prometheus metrics address (default: `:9090`)
- `-enable-metrics`: Enable Prometheus metrics endpoint (default: `true`)

## Monitoring Endpoints

### Metrics Endpoint

Access the Prometheus metrics at:

```
http://localhost:9090/metrics
```

Example output:

```
# HELP quic_backbone_connected_clients Current number of connected clients
# TYPE quic_backbone_connected_clients gauge
quic_backbone_connected_clients 5

# HELP quic_backbone_total_connections Total number of connections
# TYPE quic_backbone_total_connections counter
quic_backbone_total_connections 10

# HELP quic_backbone_messages_sent_total Total number of messages sent
# TYPE quic_backbone_messages_sent_total counter
quic_backbone_messages_sent_total 1234
...
```

### Root Endpoint

Access the monitoring dashboard at:

```
http://localhost:9090/
```

## Prometheus Configuration

To scrape metrics with Prometheus, add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'quic_backbone'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
```

## Example Output

The server will periodically print metrics to the console:

```
=== Metrics Snapshot ===
Connected Clients: 5
Total Connections: 10
Messages Sent: 1234
Messages Received: 5678
Throughput: 150 msg/s
Avg Latency: 12 ms
P50 Latency: 10 ms
P95 Latency: 25 ms
P99 Latency: 40 ms
Heartbeat Timeouts: 2
Active Promises: 15
Encoding Errors: 0
Decoding Errors: 0
Network Errors: 1
Uptime: 3600 seconds
========================
```

## Event Hooks Output

When clients connect or send messages, you'll see hook notifications:

```
[HOOK] Client connected: client-001
[HOOK] Message sent: msg-123 to client-001
[HOOK] Message received: msg-456 from client-001
[HOOK] Client disconnected: client-001, reason: client disconnect
```

## Integration with Grafana

You can visualize the metrics in Grafana:

1. Add Prometheus as a data source
2. Create a dashboard with panels for:
   - Connected clients (gauge)
   - Message throughput (graph)
   - Latency distribution (graph)
   - Error rates (graph)

Example Grafana queries:

```promql
# Connected clients
quic_backbone_connected_clients

# Message rate (per minute)
rate(quic_backbone_messages_sent_total[1m])

# P99 latency
quic_backbone_latency_p99_milliseconds

# Error rate
rate(quic_backbone_encoding_errors_total[1m])
```

## Notes

- The metrics endpoint uses the Prometheus text format (version 0.0.4)
- Metrics are updated in real-time as events occur
- The server prints a snapshot every 10 seconds
- All timestamps are in Unix milliseconds
