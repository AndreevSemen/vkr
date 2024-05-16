package udp

import (
	"encoding/json"
	"log"
	"net"
	"sync/atomic"
	"time"

	"AndreevSemen/vkr/internal/spread/types"

	"github.com/pkg/errors"
)

type agentStat struct {
	send atomic.Int64
	recv atomic.Int64
}

type Cluster struct {
	stats    map[string]*agentStat
	messages atomic.Int64

	agent types.Agent
}

func NewCluster(addr string) (types.Cluster, error) {
	c := &Cluster{
		stats: make(map[string]*agentStat),
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	lis, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	go c.manage(lis)

	return c, nil
}

func (c *Cluster) Join(a types.Agent) error {
	c.agent = a
	return nil
}

func (c *Cluster) Send(msg types.Message, timeout time.Duration) error {
	conn, err := net.DialTimeout("udp", msg.DstAddress, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		err = errors.Wrap(err, "encode message")
		return err
	}

	if _, exists := c.stats[msg.DstAddress]; !exists {
		c.stats[msg.DstAddress] = new(agentStat)
	}
	c.stats[msg.DstAddress].send.Add(1)
	c.messages.Add(1)

	return nil
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

func (c *Cluster) manage(conn *net.UDPConn) {
	for {
		var msg types.Message
		if err := json.NewDecoder(conn).Decode(&msg); err != nil {
			log.Println("decode message:", err)
			return
		}

		if err := c.agent.Receive(msg); err != nil {
			log.Println("receive message:", err)
		}

		if _, exists := c.stats[msg.SrcAddress]; !exists {
			c.stats[msg.SrcAddress] = new(agentStat)
		}
		c.stats[msg.SrcAddress].recv.Add(1)
	}
}
