package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define the GaugeVec for our device points
var (
	deviceMetric = prometheus.NewDesc(
		prometheus.BuildFQName("nibe", "device", "point_value"),
		"Current value of a NIBE device data point",
		[]string{"id", "title", "unit"}, nil,
	)
	username string
	password string
)

type DeviceCollector struct {
	endpoint string
}

// Describe sends the super-type of all metrics we can collect
func (c *DeviceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- deviceMetric
}

// Collect is called by Prometheus on every scrape
func (c *DeviceCollector) Collect(ch chan<- prometheus.Metric) {
	// 1. Fetch data from your API
	data, err := fetchDeviceData(c.endpoint)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return
	}

	// 2. Loop through the map and push to Prometheus
	for id, point := range data {
		val := float64(point.Value.IntegerValue)

		// Ignore the "null" sensor value (-32768)
		if val == -32768 {
			continue
		}

		// Calculate actual value based on divisor
		actualValue := val

		if point.Metadata.Divisor > 0 {
			actualValue = val / float64(point.Metadata.Divisor)
		}

		ch <- prometheus.MustNewConstMetric(
			deviceMetric,
			prometheus.GaugeValue,
			actualValue,
			id, point.Title, point.Metadata.Unit,
		)
	}
}

// fetchDeviceData handles the HTTP logic we discussed earlier
func fetchDeviceData(url string) (map[string]Point, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}

	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var target map[string]Point
	err = json.NewDecoder(resp.Body).Decode(&target)
	return target, err
}

func main() {
	deviceSerial := requireEnv("DEVICE_SERIAL")
	username = requireEnv("USERNAME")
	password = requireEnv("PASSWORD")
	api_url := requireEnv("API_URL")
	metricsPort := getPort("METRICS_PORT", "9090")
	collector := &DeviceCollector{
		endpoint: api_url + "/api/v1/devices/" + deviceSerial + "/points",
	}

	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Exporter running on :" + metricsPort + "/metrics")
	log.Fatal(http.ListenAndServe(":"+metricsPort, nil))
}

func getPort(name string, defaultValue string) string {
	val := os.Getenv(name)
	if val == "" {
		val = defaultValue
	}

	// 1. Convert string to integer
	port, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("Invalid port %q for %s: must be a number", val, name)
	}

	// 2. Validate the port range (1 to 65535)
	if port < 1 || port > 65535 {
		log.Fatalf("Port %d is out of valid range (1-65535)", port)
	}

	return fmt.Sprintf("%d", port)
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatal("Missing environment variable: " + name)
	}
	return v
}
