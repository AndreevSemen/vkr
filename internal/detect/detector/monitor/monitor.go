package monitor

import (
	"context"
	"gossip_emulation/internal/detect/types"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var (
	ErrNoHostsToMonitor = errors.New("no hosts to monitor")
)

type HTTPMonitorConfig struct {
	Hosts         []url.URL
	CheckInterval time.Duration
	HealthCheck   func(*http.Response) bool
	HTTPTimeout   time.Duration
	TCPTimeout    time.Duration
	ICMPTimeout   time.Duration
}

type HTTPMonitor struct {
	cfg HTTPMonitorConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	events chan types.Event
}

func NewHTTPMonitor(cfg HTTPMonitorConfig) (types.Detector, error) {
	if len(cfg.Hosts) == 0 {
		return nil, ErrNoHostsToMonitor
	}

	hosts := make([]url.URL, len(cfg.Hosts))
	copy(hosts, cfg.Hosts)
	rand.Shuffle(len(hosts), func(i, j int) {
		hosts[i], hosts[j] = hosts[j], hosts[i]
	})
	cfg.Hosts = hosts

	ctx, cancel := context.WithCancel(context.Background())
	m := &HTTPMonitor{
		cfg:    cfg,
		events: make(chan types.Event),
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.manage()
	}()

	return m, nil
}

func (m *HTTPMonitor) DetectorEvents() <-chan types.Event {
	return m.events
}

func (m *HTTPMonitor) Close() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

func (m *HTTPMonitor) manage() {
	states := make(map[string]types.AppState, len(m.cfg.Hosts))
	for _, u := range m.cfg.Hosts {
		states[u.Hostname()] = -1
	}

	ticker := time.NewTicker(m.cfg.CheckInterval)
	for {
		select {
		case <-m.ctx.Done():
			close(m.events)
			return

		case <-ticker.C:
			u := m.cfg.Hosts[0]
			m.cfg.Hosts = append(m.cfg.Hosts[1:], m.cfg.Hosts[0])

			hostname := u.Hostname()
			state := states[hostname]
			newState := m.check(u, state)

			if newState != state {
				e := types.Event{
					Hostname: hostname,
					State:    newState,
				}
				select {
				case <-m.ctx.Done():
					close(m.events)
					return
				case m.events <- e:
					states[hostname] = newState
				}
			}
		}
	}
}

func (m *HTTPMonitor) check(u url.URL, state types.AppState) types.AppState {
	switch state {
	case types.AppState_AppAvailable:
		if m.checkHTTP(u) {
			return types.AppState_AppAvailable
		}

		if m.checkPort(u) {
			return types.AppState_PortAvailable
		}

		if m.checkHost(u) {
			return types.AppState_HostAvailable
		}

		return types.AppState_Unavailable

	case types.AppState_PortAvailable:
		if m.checkPort(u) {
			if m.checkHTTP(u) {
				return types.AppState_AppAvailable
			} else {
				return types.AppState_PortAvailable
			}
		} else {
			if m.checkHost(u) {
				return types.AppState_HostAvailable
			} else {
				return types.AppState_Unavailable
			}
		}

	default:
		if !m.checkHost(u) {
			return types.AppState_Unavailable
		}

		if !m.checkPort(u) {
			return types.AppState_HostAvailable
		}

		if !m.checkHTTP(u) {
			return types.AppState_PortAvailable
		}

		return types.AppState_AppAvailable
	}
}

func (m *HTTPMonitor) checkHTTP(u url.URL) bool {
	cli := *http.DefaultClient
	cli.Timeout = m.cfg.HTTPTimeout

	resp, err := cli.Get(u.String())
	if err != nil {
		return false
	}

	return m.cfg.HealthCheck(resp)
}

func (m *HTTPMonitor) checkPort(u url.URL) bool {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), m.cfg.TCPTimeout)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}

	return false
}

func (m *HTTPMonitor) checkHost(u url.URL) bool {
	ip, err := net.ResolveIPAddr("ip4", u.Hostname())
	if err != nil {
		// log.Printf("Error on ResolveIPAddr %v", err)
		return false
	}
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		// log.Printf("Error on ListenPacket %v", err)
		return false
	}
	defer conn.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte(""),
		},
	}
	msg_bytes, err := msg.Marshal(nil)
	if err != nil {
		// log.Printf("Error on Marshal %v %v", msg_bytes, err)
		return false
	}

	// Write the message to the listening connection
	if _, err := conn.WriteTo(msg_bytes, &net.UDPAddr{IP: net.ParseIP(ip.String())}); err != nil {
		// log.Printf("Error on WriteTo %v", err)
		return false
	}

	err = conn.SetReadDeadline(time.Now().Add(m.cfg.ICMPTimeout))
	if err != nil {
		// log.Printf("Error on SetReadDeadline %v", err)
		return false
	}

	reply := make([]byte, 1500)
	n, _, err := conn.ReadFrom(reply)
	if err != nil {
		// log.Printf("Error on ReadFrom %v", err)
		return false
	}

	parsed_reply, err := icmp.ParseMessage(1, reply[:n])
	if err != nil {
		// log.Printf("Error on ParseMessage %v", err)
		return false
	}

	switch parsed_reply.Code {
	case 0:
		// Got a reply so we can save this
		return true
	case 3:
		// Given that we don't expect google to be unreachable, we can assume that our network is down
		return false
	case 11:
		// Time Exceeded so we can assume our network is slow
		return false
	default:
		// We don't know what this is so we can assume it's unreachable
		return false
	}
}
