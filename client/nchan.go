// Package client fetches Nchan stub_status text.
package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Stats is one nchan_stub_status scrape.
// Shared memory fields are kilobytes: the trailing K is removed and the number is kept.
type Stats struct {
	MessagesPublished           float64
	MessagesStored              float64
	SharedMemoryUsed            float64
	SharedMemoryLimit           float64
	Channels                    float64
	Subscribers                 float64
	RedisPendingCommands        float64
	RedisConnectedServers       float64
	RedisUnhealthyUpstreams     float64
	RedisCommandsSent           float64
	InterprocessAlertsReceived  float64
	InterprocessAlertsInTransit float64
	InterprocessQueuedAlerts    float64
	InterprocessSendDelay       float64
	InterprocessReceiveDelay    float64
	Version                     string
}

// Client fetches the nchan_stub_status page.
type Client struct {
	endpoint string
	http     *http.Client
}

// New returns a client for endpoint. httpClient may be nil.
func New(httpClient *http.Client, endpoint string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{endpoint: endpoint, http: httpClient}
}

// Endpoint returns the status URL.
func (c *Client) Endpoint() string {
	return c.endpoint
}

// GetStats fetches and parses the status page.
func (c *Client) GetStats() (*Stats, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", c.endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", c.endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: status %s", c.endpoint, resp.Status)
	}
	stats, err := parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", c.endpoint, err)
	}
	return stats, nil
}

type field struct {
	kb  bool
	set func(*Stats, float64)
}

// Keys nchan has printed. Unknown keys are ignored so a newer status page still scrapes.
var fields = map[string]field{
	"total published messages":           {set: func(s *Stats, n float64) { s.MessagesPublished = n }},
	"stored messages":                    {set: func(s *Stats, n float64) { s.MessagesStored = n }},
	"shared memory used":                 {kb: true, set: func(s *Stats, n float64) { s.SharedMemoryUsed = n }},
	"shared memory limit":                {kb: true, set: func(s *Stats, n float64) { s.SharedMemoryLimit = n }},
	"channels":                           {set: func(s *Stats, n float64) { s.Channels = n }},
	"subscribers":                        {set: func(s *Stats, n float64) { s.Subscribers = n }},
	"redis pending commands":             {set: func(s *Stats, n float64) { s.RedisPendingCommands = n }},
	"redis connected servers":            {set: func(s *Stats, n float64) { s.RedisConnectedServers = n }},
	"redis unhealthy upstreams":          {set: func(s *Stats, n float64) { s.RedisUnhealthyUpstreams = n }},
	"total redis commands sent":          {set: func(s *Stats, n float64) { s.RedisCommandsSent = n }},
	"total interprocess alerts received": {set: func(s *Stats, n float64) { s.InterprocessAlertsReceived = n }},
	"interprocess alerts in transit":     {set: func(s *Stats, n float64) { s.InterprocessAlertsInTransit = n }},
	"interprocess queued alerts":         {set: func(s *Stats, n float64) { s.InterprocessQueuedAlerts = n }},
	"total interprocess send delay":      {set: func(s *Stats, n float64) { s.InterprocessSendDelay = n }},
	"total interprocess receive delay":   {set: func(s *Stats, n float64) { s.InterprocessReceiveDelay = n }},
}

func parse(body string) (*Stats, error) {
	stats := &Stats{}
	seen := 0
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "nchan version" {
			stats.Version = value
			seen++
			continue
		}
		f, ok := fields[key]
		if !ok {
			continue
		}
		raw := value
		if f.kb {
			raw = strings.TrimSuffix(raw, "K")
		}
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: parse %q: %w", key, value, err)
		}
		f.set(stats, n)
		seen++
	}
	if seen == 0 {
		return nil, fmt.Errorf("no nchan stats in %q", body)
	}
	return stats, nil
}
