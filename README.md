# Nibe prometheus exporter

### Requirements
* This exporter require nibe rest-API to be enabled

## environment variables that is needed is 
* API_URL
* DEVICE_SERIAL
* USERNAME
* PASSWORD
* METRICS_PORT (optional default: 9090)


## Example usage with docker
```bash
docker run -i \
-e API_URL=https://<NIBE_IPADDRESS:NIBE_PORT> \
-e DEVICE_SERIAL=<NIBE_SERIAL> \
-e USERNAME=<USERNAME> \
-e PASSWORD=<PASSWORD> \
-p 9090:9090 \
-t rehn/nibe-prometheus-exporter:latest 
```
Will be accessible from  http://localhost:9090/metrics


## Install in kubernetes with helm
```bash
helm repo add nibe-exporter https://rehn.github.io/nibe-prometheus-exporter/
helm repo update
helm install my-release nibe-exporter/nibe-prometheus-exporter
```
