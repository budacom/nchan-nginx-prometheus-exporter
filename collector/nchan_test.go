package collector

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/budacom/nchan-prometheus-exporter/client"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestCollect139(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("../client/testdata/nchan_1.3.9.txt")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(New(client.New(srv.Client(), srv.URL), discard()))
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]*dto.MetricFamily{}
	for _, mf := range mfs {
		got[mf.GetName()] = mf
	}

	counters := []string{
		"nchan_messages_published_total",
		"nchan_messages_stored",
		"nchan_redis_commands_sent_total",
		"nchan_interprocess_alerts_received_total",
		"nchan_interprocess_send_delay_total",
		"nchan_interprocess_receive_delay_total",
	}
	gauges := []string{
		"nchan_shared_memory_used",
		"nchan_shared_memory_limit",
		"nchan_channels",
		"nchan_subscribers",
		"nchan_redis_pending_commands",
		"nchan_redis_connected_servers",
		"nchan_redis_unhealthy_upstreams",
		"nchan_interprocess_alerts_in_transit",
		"nchan_interprocess_queued_alerts",
		"nchan_version_info",
		"nchan_up",
	}
	if len(got) != len(counters)+len(gauges) {
		t.Fatalf("got %d metrics, want %d: %v", len(got), len(counters)+len(gauges), names(got))
	}
	for _, name := range counters {
		mf := got[name]
		if mf == nil || mf.GetType() != dto.MetricType_COUNTER {
			t.Fatalf("%s type = %v", name, metricType(mf))
		}
	}
	for _, name := range gauges {
		mf := got[name]
		if mf == nil || mf.GetType() != dto.MetricType_GAUGE {
			t.Fatalf("%s type = %v", name, metricType(mf))
		}
	}
	if got["nchan_shared_memory_used"].GetMetric()[0].GetGauge().GetValue() != 16 {
		t.Fatalf("shared memory used = %v", got["nchan_shared_memory_used"])
	}
	if got["nchan_shared_memory_limit"].GetMetric()[0].GetGauge().GetValue() != 131072 {
		t.Fatalf("shared memory limit = %v", got["nchan_shared_memory_limit"])
	}
	labels := got["nchan_version_info"].GetMetric()[0].GetLabel()
	if len(labels) != 1 || labels[0].GetName() != "version" || labels[0].GetValue() != "1.3.9" {
		t.Fatalf("version labels = %v", labels)
	}
	if n := len(got["nchan_interprocess_queued_alerts"].GetMetric()[0].GetLabel()); n != 0 {
		t.Fatalf("queued alerts has %d labels", n)
	}
	if got["nchan_up"].GetMetric()[0].GetGauge().GetValue() != 1 {
		t.Fatalf("nchan_up = %v", got["nchan_up"])
	}
}

func TestCollectDown(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(New(client.New(srv.Client(), srv.URL), discard()))
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	if len(mfs) != 1 || mfs[0].GetName() != "nchan_up" || mfs[0].GetMetric()[0].GetGauge().GetValue() != 0 {
		t.Fatalf("metrics = %v", mfs)
	}
}

func discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func names(mfs map[string]*dto.MetricFamily) []string {
	out := make([]string, 0, len(mfs))
	for name := range mfs {
		out = append(out, name)
	}
	return out
}

func metricType(mf *dto.MetricFamily) string {
	if mf == nil {
		return "missing"
	}
	return mf.GetType().String()
}
