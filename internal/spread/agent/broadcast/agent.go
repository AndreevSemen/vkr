package broadcast

import (
	"AndreevSemen/vkr/internal/spread/types"
	"time"

	"golang.org/x/sync/errgroup"
)

type BroadcastAgentConfig struct {
	Address     string
	SeedAgents  []string
	SendTimeout time.Duration
}

type BroadcastAgent struct {
	cfg    BroadcastAgentConfig
	cl     types.Cluster
	events chan types.Event
}

func NewBroadcastAgent(cfg BroadcastAgentConfig, cl types.Cluster) (types.Agent, error) {
	a := &BroadcastAgent{
		cfg:    cfg,
		cl:     cl,
		events: make(chan types.Event, 100),
	}

	cl.Join(a)

	return a, nil
}

func (a *BroadcastAgent) Address() string {
	return a.cfg.Address
}

func (a *BroadcastAgent) Receive(m types.Message) error {
	a.events <- m.Event
	return nil
}

func (a *BroadcastAgent) GetEvents() <-chan types.Event {
	return a.events
}

func (a *BroadcastAgent) PublishEvent(e types.Event) error {
	eg := &errgroup.Group{}
	eg.SetLimit(len(a.cfg.SeedAgents))

	a.events <- e

	for _, addr := range a.cfg.SeedAgents {
		if addr == a.cfg.Address {
			continue
		}

		m := types.Message{
			SrcAddress: a.cfg.Address,
			DstAddress: addr,
			Event:      e,
		}
		eg.Go(func() error {
			return a.cl.Send(m, a.cfg.SendTimeout)
		})
	}

	return eg.Wait()
}

func (a *BroadcastAgent) Close() error {
	close(a.events)
	return nil
}
