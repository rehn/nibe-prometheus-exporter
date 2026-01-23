FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN echo "Bygger för OS: $TARGETOS, Arch: $TARGETARCH" && CGO_ENABLED=0 GOOS=linux GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o nibe-prometheus-exporter .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 exportergroup && \
    adduser -u 1000 -G exportergroup -D exporteruser

WORKDIR /home/exporteruser
COPY --from=builder /app/nibe-prometheus-exporter .
RUN chown exporteruser:exportergroup nibe-prometheus-exporter
USER 1000
EXPOSE 9090
ENV METRICS_PORT=9090
CMD ["./nibe-prometheus-exporter"]