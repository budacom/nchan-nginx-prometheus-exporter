package client

import (
	"os"
	"strings"
	"testing"
)

func TestParseFixtures(t *testing.T) {
	t.Parallel()

	v12, err := os.ReadFile("testdata/nchan_1.2.x.txt")
	if err != nil {
		t.Fatal(err)
	}
	// The 0.2.0 parser required strings.Split to return 15 parts (14 lines plus the trailing newline).
	if got := len(strings.Split(string(v12), "\n")); got != 15 {
		t.Fatalf("1.2.x fixture split into %d parts, want 15", got)
	}

	v139, err := os.ReadFile("testdata/nchan_1.3.9.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(strings.TrimSuffix(string(v139), "\n"), "\n") + 1; got != 16 {
		t.Fatalf("1.3.9 fixture has %d lines, want 16", got)
	}

	tests := []struct {
		name    string
		body    string
		want    Stats
		wantErr bool
	}{
		{
			name: "1.2.x",
			body: string(v12),
			want: Stats{
				MessagesPublished:           123,
				MessagesStored:              54353,
				SharedMemoryUsed:            12,
				SharedMemoryLimit:           131072,
				Channels:                    34,
				Subscribers:                 5434535,
				RedisPendingCommands:        48,
				RedisConnectedServers:       65,
				InterprocessAlertsReceived:  43,
				InterprocessAlertsInTransit: 654,
				InterprocessQueuedAlerts:    765,
				InterprocessSendDelay:       534,
				InterprocessReceiveDelay:    46,
				Version:                     "1.1.5",
			},
		},
		{
			name: "1.3.9",
			body: string(v139),
			want: Stats{
				SharedMemoryUsed:  16,
				SharedMemoryLimit: 131072,
				Version:           "1.3.9",
			},
		},
		{
			name: "new fields and unknown key",
			body: "" +
				"total published messages: 7\n" +
				"shared memory used: 16.5K\n" +
				"redis unhealthy upstreams: 3\n" +
				"total redis commands sent: 99\n" +
				"future field: 1\n" +
				"nchan version: 1.3.9\n",
			want: Stats{
				MessagesPublished:       7,
				SharedMemoryUsed:        16.5,
				RedisUnhealthyUpstreams: 3,
				RedisCommandsSent:       99,
				Version:                 "1.3.9",
			},
		},
		{
			name:    "not status text",
			body:    "invalid-stats",
			wantErr: true,
		},
		{
			name:    "known key with a bad number",
			body:    "total published messages: nope\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parse(tt.body)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if *got != tt.want {
				t.Fatalf("got %+v, want %+v", *got, tt.want)
			}
		})
	}
}
