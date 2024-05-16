package main

import (
	"gossip_emulation/internal/controller"
	"gossip_emulation/internal/detect/detector/detector"
	monitor "gossip_emulation/internal/monitor"
	"gossip_emulation/internal/spread/agent/gossip"
	"gossip_emulation/internal/spread/cluster/udp"
	"strings"

	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	bind := flag.String("frontend_addr", "0.0.0.0:8000", "frontend server address")
	addr := flag.String("agent_addr", "0.0.0.0:1111", "addr of current agent")
	seedAgents := flag.String("seed_agents", "0.0.0.0:1111", "list of seed agents separated with comma")

	flag.Parse()

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
			{
				Scheme: "http",
				Host:   "wikipedia.org",
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
	d, err := detector.NewHTTPDetector(dCfg)
	if err != nil {
		panic(err)
	}

	aCfg := gossip.GossipAgentConfig{
		Address:     *addr,
		SendTimeout: 200 * time.Millisecond,
		SeedAgents:  strings.Split(*seedAgents, ","),
		Fanout:      3,
		Heartbeat:   200 * time.Millisecond,
	}

	cl, err := udp.NewCluster(*addr)
	if err != nil {
		panic(err)
	}

	a, err := gossip.NewGossipAgent(aCfg, cl)
	if err != nil {
		panic(err)
	}

	m := monitor.NewMonitor(d, a)

	app := fiber.New()
	ctrl := controller.NewController(m)
	ctrl.Route(app)

	go func() {
		if err = app.Listen(*bind); err != nil {
			log.Panic("server error:", err)
		}
	}()

	<-context.Background().Done()
}
