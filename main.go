package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// invalidValue is what the NIBE API reports for points without a reading.
	invalidValue = -32768
	fetchTimeout = 10 * time.Second
)

var (
	deviceMetric = prometheus.NewDesc(
		prometheus.BuildFQName("nibe", "device", "point_value"),
		"Current value of a NIBE device data point",
		[]string{"id", "title", "unit"}, nil,
	)
	upMetric = prometheus.NewDesc(
		prometheus.BuildFQName("nibe", "", "up"),
		"Whether the last fetch from the NIBE API succeeded (1) or failed (0)",
		nil, nil,
	)
	scrapeDurationMetric = prometheus.NewDesc(
		prometheus.BuildFQName("nibe", "scrape", "duration_seconds"),
		"Time taken to fetch data from the NIBE API",
		nil, nil,
	)
)

type DeviceCollector struct {
	endpoint string
	username string
	password string
	client   *http.Client
}

func NewDeviceCollector(endpoint, username, password string, insecureSkipVerify bool) *DeviceCollector {
	return &DeviceCollector{
		endpoint: endpoint,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: fetchTimeout,
			Transport: &http.Transport{
				// The NIBE local API uses a self-signed certificate by default.
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
			},
		},
	}
}

func (c *DeviceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- deviceMetric
	ch <- upMetric
	ch <- scrapeDurationMetric
}

func (c *DeviceCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	data, err := c.fetch()
	ch <- prometheus.MustNewConstMetric(scrapeDurationMetric, prometheus.GaugeValue, time.Since(start).Seconds())

	if err != nil {
		log.Printf("Error fetching data: %v", err)
		ch <- prometheus.MustNewConstMetric(upMetric, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(upMetric, prometheus.GaugeValue, 1)

	for id, point := range data {
		if point.Value.IntegerValue == invalidValue {
			continue
		}
		ch <- prometheus.MustNewConstMetric(
			deviceMetric,
			prometheus.GaugeValue,
			point.GetActualValue(),
			id, point.Title, point.Metadata.Unit,
		)
	}
}

func (c *DeviceCollector) fetch() (map[string]Point, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("unexpected status %s from %s: %s", resp.Status, c.endpoint, strings.TrimSpace(string(body)))
	}

	var target map[string]Point
	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return target, nil
}

func main() {
	deviceSerial := requireEnv("DEVICE_SERIAL")
	username := requireEnv("USERNAME")
	password := requireEnv("PASSWORD")
	apiURL := strings.TrimRight(requireEnv("API_URL"), "/")
	metricsPort := getPort("METRICS_PORT", "9090")
	insecure := getBool("TLS_INSECURE_SKIP_VERIFY", true)

	collector := NewDeviceCollector(apiURL+"/api/v1/devices/"+deviceSerial+"/points", username, password, insecure)

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collector,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	server := &http.Server{
		Addr:              ":" + metricsPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Exporter running on :%s/metrics", metricsPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
}

func getPort(name string, defaultValue string) string {
	val := os.Getenv(name)
	if val == "" {
		val = defaultValue
	}

	port, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("Invalid port %q for %s: must be a number", val, name)
	}

	if port < 1 || port > 65535 {
		log.Fatalf("Port %d is out of valid range (1-65535)", port)
	}

	return strconv.Itoa(port)
}

func getBool(name string, defaultValue bool) bool {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		log.Fatalf("Invalid boolean %q for %s", val, name)
	}
	return b
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatal("Missing environment variable: " + name)
	}
	return v
}
