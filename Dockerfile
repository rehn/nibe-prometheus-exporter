# --- STAGE 1: Build ---
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Här bygger vi filen 'nibe-prometheus-exporter'
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o nibe-prometheus-exporter .

# --- STAGE 2: Run ---
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
# KORRIGERAD: Kopierar nu rätt filnamn från builder
COPY --from=builder /app/nibe-prometheus-exporter .
EXPOSE 9090
ENV METRICS_PORT=9090
CMD ["./nibe-prometheus-exporter"]