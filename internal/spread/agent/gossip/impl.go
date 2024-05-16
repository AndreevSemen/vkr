package gossip

import (
	"log"
	"time"

	"gossip_emulation/internal/spread/types"
)

func (a *GossipAgent) address() string {
	return a.cfg.Address
}

func (a *GossipAgent) receive(m types.Message) error {
	select {
	case <-a.ctx.Done():
		return nil
	default:
	}

	a.senders.Add(1)
	defer a.senders.Add(-1)

	a.spreadMx.Lock()
	if _, exists := a.agents[m.SrcAddress]; !exists {
		a.agents[m.SrcAddress] = struct{}{}
	}
	a.spreadMx.Unlock()

	select {
	case <-a.ctx.Done():
		return nil

	case a.eventsSend <- m.Event:
		a.spreadMx.Lock()
		if ctx, exists := a.spread[m.Event.UUID]; !exists {
			a.spread[m.Event.UUID] = a.makeSpreadContext(m.Event)
		} else {
			delete(ctx.Agents, m.SrcAddress)
			if len(ctx.Agents) == 0 {
				delete(a.spread, m.Event.UUID)
			}
		}
		a.spreadMx.Unlock()
	}

	return nil
}

func (a *GossipAgent) getEvents() <-chan types.Event {
	return a.eventsRecv
}

func (a *GossipAgent) publishEvent(e types.Event) error {
	select {
	case <-a.ctx.Done():
		return nil
	default:
	}

	a.senders.Add(1)
	defer a.senders.Add(-1)

	select {
	case <-a.ctx.Done():
		return nil

	case a.eventsSend <- e:
		a.spreadMx.Lock()
		ctx := a.makeSpreadContext(e)
		if _, exists := a.spread[e.UUID]; !exists {
			a.spread[e.UUID] = ctx
		}
		a.spreadMx.Unlock()
		a.spreadEvent(ctx)
	}

	return nil
}

func (a *GossipAgent) close() error {
	eventsSend := a.eventsSend
	a.eventsSend = nil

	a.cancel()
	a.wg.Wait()

	for !a.senders.CompareAndSwap(0, 0) {
	}

	close(eventsSend)

	return nil
}

func (a *GossipAgent) getDestinations(ctx spreadContext) []string {
	a.spreadMx.Lock()
	defer a.spreadMx.Unlock()

	dsts := make([]string, 0, a.cfg.Fanout)
	for addr := range ctx.Agents {
		if len(dsts) >= a.cfg.Fanout {
			break
		}

		delete(ctx.Agents, addr)
		dsts = append(dsts, addr)
	}

	return dsts
}

func (a *GossipAgent) spreadEvent(ctx spreadContext) {
	dsts := a.getDestinations(ctx)

	if len(dsts) == 0 {
		a.spreadMx.Lock()
		delete(a.spread, ctx.Event.UUID)
		a.spreadMx.Unlock()
		return
	}

	for _, dst := range dsts {
		m := types.Message{
			SrcAddress: a.cfg.Address,
			DstAddress: dst,
			Event:      ctx.Event,
		}
		go func() {
			err := a.cl.Send(m, a.cfg.SendTimeout)
			if err != nil {
				log.Printf("send to %s: %v", m.DstAddress, err)
			}
		}()
	}
}

func (a *GossipAgent) makeSpreadContext(e types.Event) spreadContext {
	agents := make(map[string]struct{}, len(a.agents))
	for agent := range a.agents {
		agents[agent] = struct{}{}
	}

	return spreadContext{
		Event:  e,
		Agents: agents,
	}
}

func (a *GossipAgent) runtime() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(a.cfg.Heartbeat):
		}

		a.spreadMx.Lock()
		events := make([]spreadContext, 0, len(a.spread))
		for _, ctx := range a.spread {
			events = append(events, ctx)
		}
		a.spreadMx.Unlock()

		for _, e := range events {
			a.spreadEvent(e)
		}
	}
}
