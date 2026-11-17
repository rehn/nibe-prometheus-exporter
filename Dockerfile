FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o nibe-prometheus-exporter .

# Static binary: no shell or package manager needed at runtime.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app/nibe-prometheus-exporter /nibe-prometheus-exporter
# Keep UID 1000 to match the Helm chart securityContext.
USER 1000:1000
EXPOSE 9090
ENV METRICS_PORT=9090
ENTRYPOINT ["/nibe-prometheus-exporter"]
