package monitor

import (
	"gossip_emulation/detect/detector/detector"
	"gossip_emulation/spread/cluster/local"
	"log"

	"gossip_emulation/spread/agent/gossip"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	dCfg := detector.HTTPDetectorConfig{
		Hosts: []url.URL{
			{
				Scheme: "http",
				Host:   "ya.ru",
			},
			{
				Scheme: "http",
				Host:   "google.com",
			},
		},
		CheckInterval: time.Second,
		HealthCheck: func(r *http.Response) bool {
			return http.StatusOK == r.StatusCode
		},
		HTTPTimeout: 5 * time.Second,
		TCPTimeout:  5 * time.Second,
		ICMPTimeout: 5 * time.Second,
	}
	d1, err := detector.NewHTTPDetector(dCfg)
	require.NoError(t, err)

	d2, err := detector.NewHTTPDetector(dCfg)
	require.NoError(t, err)

	aCfg := gossip.GossipAgentConfig{
		SendTimeout: 200 * time.Millisecond,
		SeedAgents: []string{
			"1",
			"2",
		},
		Fanout:    3,
		Heartbeat: 10 * time.Millisecond,
	}

	cl := local.NewCluster(nil)

	aCfg.Address = "1"
	a1, err := gossip.NewGossipAgent(aCfg, cl)
	require.NoError(t, err)

	aCfg.Address = "2"
	a2, err := gossip.NewGossipAgent(aCfg, cl)
	require.NoError(t, err)

	m1 := NewMonitor(d1, a1)
	go func() {
		for spreadEvent := range m1.WatchEvents() {
			log.Printf(
				"monitor1 got event (%s from %s): %s, %d",
				spreadEvent.UUID.String(),
				spreadEvent.Source,
				spreadEvent.DetectEvent.Hostname,
				spreadEvent.DetectEvent.State,
			)
		}
	}()

	m2 := NewMonitor(d2, a2)
	go func() {
		for spreadEvent := range m2.WatchEvents() {
			log.Printf(
				"monitor2 got event (%s from %s): %s, %d",
				spreadEvent.UUID.String(),
				spreadEvent.Source,
				spreadEvent.DetectEvent.Hostname,
				spreadEvent.DetectEvent.State,
			)
		}
	}()

	time.Sleep(5 * time.Second)

	err = m1.Close()
	require.NoError(t, err)

	err = m2.Close()
	require.NoError(t, err)
}
