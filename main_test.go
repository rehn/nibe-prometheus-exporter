package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

const samplePoints = `{
  "40004": {"title": "outdoor temp", "metadata": {"unit": "°C", "divisor": 10}, "value": {"isOk": true, "integerValue": -35}},
  "1234":  {"title": "no divisor", "metadata": {"unit": "", "divisor": 0}, "value": {"isOk": true, "integerValue": 7}},
  "9999":  {"title": "missing", "metadata": {"unit": "", "divisor": 1}, "value": {"isOk": true, "integerValue": -32768}}
}`

func TestCollectSuccess(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "user" || p != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(samplePoints))
	}))
	defer srv.Close()

	c := NewDeviceCollector(srv.URL, "user", "pass", true)

	expected := `
# HELP nibe_device_point_value Current value of a NIBE device data point
# TYPE nibe_device_point_value gauge
nibe_device_point_value{id="1234",title="no divisor",unit=""} 7
nibe_device_point_value{id="40004",title="outdoor temp",unit="°C"} -3.5
# HELP nibe_up Whether the last fetch from the NIBE API succeeded (1) or failed (0)
# TYPE nibe_up gauge
nibe_up 1
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected), "nibe_device_point_value", "nibe_up"); err != nil {
		t.Fatal(err)
	}
}

func TestCollectHTTPError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewDeviceCollector(srv.URL, "user", "wrong", true)

	expected := `
# HELP nibe_up Whether the last fetch from the NIBE API succeeded (1) or failed (0)
# TYPE nibe_up gauge
nibe_up 0
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected), "nibe_device_point_value", "nibe_up"); err != nil {
		t.Fatal(err)
	}
}

func TestCollectRejectsSelfSignedWhenVerifying(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(samplePoints))
	}))
	defer srv.Close()

	c := NewDeviceCollector(srv.URL, "user", "pass", false)
	if got := testutil.ToFloat64(upOnly{c}); got != 0 {
		t.Fatalf("expected nibe_up 0 with TLS verification enabled, got %v", got)
	}
}

// upOnly exposes just the nibe_up metric so testutil.ToFloat64 can read it.
type upOnly struct{ c *DeviceCollector }

func (u upOnly) Describe(ch chan<- *prometheus.Desc) { ch <- upMetric }
func (u upOnly) Collect(ch chan<- prometheus.Metric) {
	all := make(chan prometheus.Metric)
	go func() { u.c.Collect(all); close(all) }()
	for m := range all {
		if m.Desc() == upMetric {
			ch <- m
		}
	}
}
