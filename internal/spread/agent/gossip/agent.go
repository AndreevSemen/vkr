package gossip

import (
	"context"
	"gossip_emulation/internal/spread/types"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type spreadContext struct {
	Event  types.Event
	Agents map[string]struct{}
}

type GossipAgentConfig struct {
	Address     string
	SeedAgents  []string
	SendTimeout time.Duration
	Fanout      int
	Heartbeat   time.Duration
}

type GossipAgent struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	cfg        GossipAgentConfig
	cl         types.Cluster
	eventsSend chan<- types.Event
	eventsRecv <-chan types.Event
	senders    atomic.Int64

	spreadMx *sync.Mutex
	spread   map[uuid.UUID]spreadContext
	agents   map[string]struct{}
}

func NewGossipAgent(cfg GossipAgentConfig, cl types.Cluster) (types.Agent, error) {
	agents := make(map[string]struct{}, len(cfg.SeedAgents))
	for _, agent := range cfg.SeedAgents {
		if agent != cfg.Address {
			agents[agent] = struct{}{}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	eventsSend, eventsRecv := newEventQueue()
	a := &GossipAgent{
		ctx:        ctx,
		cancel:     cancel,
		wg:         &sync.WaitGroup{},
		cfg:        cfg,
		cl:         cl,
		eventsSend: eventsSend,
		eventsRecv: eventsRecv,
		spreadMx:   &sync.Mutex{},
		spread:     make(map[uuid.UUID]spreadContext),
		agents:     agents,
	}

	if err := cl.Join(a); err != nil {
		return nil, err
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.runtime()
	}()

	return a, nil
}

func (a *GossipAgent) Address() string {
	return a.address()
}

func (a *GossipAgent) Receive(m types.Message) error {
	return a.receive(m)
}

func (a *GossipAgent) GetEvents() <-chan types.Event {
	return a.getEvents()
}

func (a *GossipAgent) PublishEvent(e types.Event) error {
	return a.publishEvent(e)
}

func (a *GossipAgent) Close() error {
	return a.close()
}
