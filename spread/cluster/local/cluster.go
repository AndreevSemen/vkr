package local

import (
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"gossip_emulation/spread/types"
)

var (
	ErrAddressAlreadyInUse      = errors.New("address already in use")
	ErrSourceIsNotJoinedYet     = errors.New("source is not joined yet")
	ErrDestinationIsUnreachable = errors.New("destination is unreachable")
)

type agentStat struct {
	send atomic.Int64
	recv atomic.Int64
}

type Cluster struct {
	// address -> agent
	agents map[string]types.Agent
	stats  map[string]*agentStat

	messages atomic.Int64

	failures map[Edge]struct{}
}

type Edge struct {
	From string
	To   string
}

func NewCluster(failures map[Edge]struct{}) types.Cluster {
	return &Cluster{
		agents:   make(map[string]types.Agent),
		stats:    make(map[string]*agentStat),
		failures: failures,
	}
}

func (c *Cluster) Join(a types.Agent) error {
	addr := a.Address()

	if _, exists := c.agents[addr]; exists {
		return ErrAddressAlreadyInUse
	}

	c.agents[addr] = a
	c.stats[addr] = &agentStat{}

	return nil
}

func (c *Cluster) Send(msg types.Message, timeout time.Duration) error {
	if _, exists := c.agents[msg.DstAddress]; !exists {
		return ErrSourceIsNotJoinedYet
	}

	dst, exists := c.agents[msg.DstAddress]
	if !exists {
		return ErrDestinationIsUnreachable
	}

	c.stats[msg.SrcAddress].send.Add(1)
	c.stats[msg.DstAddress].recv.Add(1)

	c.messages.Add(1)

	edge1 := Edge{
		From: msg.SrcAddress,
		To:   msg.DstAddress,
	}

	edge2 := Edge{
		To:   msg.SrcAddress,
		From: msg.DstAddress,
	}

	if _, exists := c.failures[edge1]; exists {
		return nil
	}

	if _, exists := c.failures[edge2]; exists {
		return nil
	}

	return dst.Receive(msg)
}

func (c *Cluster) SentMessages() int {
	return int(c.messages.Load())
}

func (c *Cluster) MaxAgentMsgs() int {
	var maxSend int
	for _, stat := range c.stats {
		send := int(stat.send.Load())
		if send > maxSend {
			maxSend = send
		}
	}
	return maxSend
}
