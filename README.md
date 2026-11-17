# NIBE Prometheus exporter

Exposes data points from a NIBE heat pump's local REST API as Prometheus metrics.

### Requirements
* The NIBE local REST API must be enabled on the heat pump.

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `API_URL` | yes | | Heat pump API URL, e.g. `https://192.168.1.200:8443` |
| `DEVICE_SERIAL` | yes | | Device serial/ID |
| `USERNAME` | yes | | API username |
| `PASSWORD` | yes | | API password |
| `METRICS_PORT` | no | `9090` | Port to serve metrics on |
| `TLS_INSECURE_SKIP_VERIFY` | no | `true` | Skip TLS certificate verification (the heat pump uses a self-signed certificate) |

## Metrics

| Metric | Description |
|--------|-------------|
| `nibe_device_point_value{id,title,unit}` | Current value of each data point (already divided by the point's divisor) |
| `nibe_up` | `1` if the last fetch from the heat pump succeeded, `0` otherwise |
| `nibe_scrape_duration_seconds` | Time taken to fetch data from the heat pump |

Standard Go runtime and process metrics are also exposed. A liveness endpoint is available at `/healthz`.

Example alert for an unreachable heat pump:
```yaml
- alert: NibeExporterDown
  expr: nibe_up == 0
  for: 10m
```

## Example usage with docker
```bash
docker run -it \
  -e API_URL=https://<NIBE_IPADDRESS>:<NIBE_PORT> \
  -e DEVICE_SERIAL=<NIBE_SERIAL> \
  -e USERNAME=<USERNAME> \
  -e PASSWORD=<PASSWORD> \
  -p 9090:9090 \
  rehn/nibe-prometheus-exporter:latest
```
Metrics will be available at http://localhost:9090/metrics

## Install in Kubernetes with Helm
```bash
helm repo add nibe-exporter https://rehn.github.io/nibe-prometheus-exporter/
helm repo update
helm install my-release nibe-exporter/nibe-prometheus-exporter \
  --set nibe.api_url=https://192.168.1.200:8443 \
  --set nibe.device_serial=<NIBE_SERIAL> \
  --set nibe.username=<USERNAME> \
  --set nibe.password=<PASSWORD>
```

Credentials are stored in a Kubernetes Secret. To use a Secret you manage yourself (keys `USERNAME` and `PASSWORD`), set `nibe.existingSecret=<secret-name>` instead of `nibe.username`/`nibe.password`.

See the [chart README](charts/nibe-prometheus-exporter/README.md) for all values.

## Development
```bash
go test ./...
go build .
```
