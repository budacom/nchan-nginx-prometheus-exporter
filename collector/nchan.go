// Package collector exports Nchan stub_status as Prometheus metrics.
package collector

import (
	"log/slog"
	"sync"

	"github.com/budacom/nchan-prometheus-exporter/client"
	"github.com/prometheus/client_golang/prometheus"
)

// Collector implements prometheus.Collector for one Nchan status URL.
type Collector struct {
	client  *client.Client
	logger  *slog.Logger
	samples []sample
	version *prometheus.Desc
	up      prometheus.Gauge
	mu      sync.Mutex
}

type sample struct {
	desc  *prometheus.Desc
	kind  prometheus.ValueType
	value func(*client.Stats) float64
}

// New returns a collector. Metric names match the 0.2.0 exporter.
func New(c *client.Client, logger *slog.Logger) *Collector {
	return &Collector{
		client: c,
		logger: logger,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "nchan",
			Name:      "up",
			Help:      "Status of the last metric scrape.",
		}),
		version: prometheus.NewDesc(
			"nchan_version_info",
			"Nchan version.",
			[]string{"version"},
			nil,
		),
		samples: []sample{
			newSample("messages_published_total", "Number of messages published to all channels through this Nchan server.", prometheus.CounterValue, func(s *client.Stats) float64 { return s.MessagesPublished }),
			newSample("messages_stored", "Number of messages currently buffered in memory.", prometheus.CounterValue, func(s *client.Stats) float64 { return s.MessagesStored }),
			newSample("shared_memory_used", "Total shared memory used for buffering messages, storing channel information, and other purposes, in kilobytes.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.SharedMemoryUsed }),
			newSample("shared_memory_limit", "Shared memory limit for buffering messages, storing channel information, and other purposes, in kilobytes.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.SharedMemoryLimit }),
			newSample("channels", "Number of channels present on this Nchan server.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.Channels }),
			newSample("subscribers", "Number of subscribers to all channels on this Nchan server.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.Subscribers }),
			newSample("redis_pending_commands", "Number of commands sent to Redis that are awaiting a reply.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.RedisPendingCommands }),
			newSample("redis_connected_servers", "Number of redis servers to which Nchan is currently connected.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.RedisConnectedServers }),
			newSample("redis_unhealthy_upstreams", "Number of Redis upstreams Nchan considers unhealthy.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.RedisUnhealthyUpstreams }),
			newSample("redis_commands_sent_total", "Number of commands sent to Redis.", prometheus.CounterValue, func(s *client.Stats) float64 { return s.RedisCommandsSent }),
			newSample("interprocess_alerts_received_total", "Number of interprocess communication packets transmitted between Nginx workers processes for Nchan.", prometheus.CounterValue, func(s *client.Stats) float64 { return s.InterprocessAlertsReceived }),
			newSample("interprocess_alerts_in_transit", "Number of interprocess communication packets in transit between Nginx workers.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.InterprocessAlertsInTransit }),
			newSample("interprocess_queued_alerts", "Number of interprocess communication packets waiting to be sent.", prometheus.GaugeValue, func(s *client.Stats) float64 { return s.InterprocessQueuedAlerts }),
			newSample("interprocess_send_delay_total", "Total amount of time interprocess communication packets spend being queued if delayed.", prometheus.CounterValue, func(s *client.Stats) float64 { return s.InterprocessSendDelay }),
			newSample("interprocess_receive_delay_total", "Total amount of time interprocess communication packets spend in transit if delayed.", prometheus.CounterValue, func(s *client.Stats) float64 { return s.InterprocessReceiveDelay }),
		},
	}
}

func newSample(name, help string, kind prometheus.ValueType, value func(*client.Stats) float64) sample {
	return sample{
		desc:  prometheus.NewDesc("nchan_"+name, help, nil, nil),
		kind:  kind,
		value: value,
	}
}

// Describe sends all metric descriptors.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up.Desc()
	ch <- c.version
	for _, s := range c.samples {
		ch <- s.desc
	}
}

// Collect fetches Nchan and sends metrics. A failed scrape sets nchan_up to 0.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats, err := c.client.GetStats()
	if err != nil {
		c.up.Set(0)
		ch <- c.up
		c.logger.Error("error getting stats", "uri", c.client.Endpoint(), "error", err)
		return
	}

	c.up.Set(1)
	ch <- c.up
	ch <- prometheus.MustNewConstMetric(c.version, prometheus.GaugeValue, 1, stats.Version)
	for _, s := range c.samples {
		ch <- prometheus.MustNewConstMetric(s.desc, s.kind, s.value(stats))
	}
}
