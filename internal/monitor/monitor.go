package monitor

import (
	detect_types "gossip_emulation/internal/detect/types"
	spread_types "gossip_emulation/internal/spread/types"

	"context"
	"log"
	"sync"

	"github.com/google/uuid"
)

type Monitor struct {
	d detect_types.Detector
	a spread_types.Agent

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	events chan spread_types.Event
}

func NewMonitor(d detect_types.Detector, a spread_types.Agent) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Monitor{
		d:      d,
		a:      a,
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
		events: make(chan spread_types.Event),
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.manage()
	}()

	return m
}

func (m *Monitor) WatchEvents() <-chan spread_types.Event {
	return m.a.GetEvents()
}

func (m *Monitor) Close() error {
	m.cancel()
	m.wg.Wait()

	detectorErr := m.d.Close()
	agentErr := m.a.Close()

	if agentErr != nil {
		return agentErr
	}

	if detectorErr != nil {
		return detectorErr
	}

	return nil
}

func (m *Monitor) manage() {
	for {
		select {
		case <-m.ctx.Done():
			return

		case detectEvent := <-m.d.DetectorEvents():
			spreadEvent := spread_types.Event{
				UUID:        uuid.New(),
				Source:      m.a.Address(),
				DetectEvent: detectEvent,
			}

			if err := m.a.PublishEvent(spreadEvent); err != nil {
				log.Println("publish spread event:", err)
				continue
			}
		}
	}
}
