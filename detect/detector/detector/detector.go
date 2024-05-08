package detector

import (
	"context"
	"gossip_emulation/detect/types"
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
	ErrNoHostsToDetect = errors.New("no hosts to detect")
)

type HTTPDetectorConfig struct {
	Hosts         []url.URL
	CheckInterval time.Duration
	HealthCheck   func(*http.Response) bool
	HTTPTimeout   time.Duration
	TCPTimeout    time.Duration
	ICMPTimeout   time.Duration
}

type HTTPDetector struct {
	cfg HTTPDetectorConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	events chan types.Event
}

func NewHTTPDetector(cfg HTTPDetectorConfig) (types.Detector, error) {
	if len(cfg.Hosts) == 0 {
		return nil, ErrNoHostsToDetect
	}

	hosts := make([]url.URL, len(cfg.Hosts))
	copy(hosts, cfg.Hosts)
	rand.Shuffle(len(hosts), func(i, j int) {
		hosts[i], hosts[j] = hosts[j], hosts[i]
	})
	cfg.Hosts = hosts

	ctx, cancel := context.WithCancel(context.Background())
	d := &HTTPDetector{
		cfg:    cfg,
		events: make(chan types.Event),
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.manage()
	}()

	return d, nil
}

func (d *HTTPDetector) DetectorEvents() <-chan types.Event {
	return d.events
}

func (d *HTTPDetector) Close() error {
	d.cancel()
	d.wg.Wait()
	return nil
}

func (d *HTTPDetector) manage() {
	states := make(map[string]types.AppState, len(d.cfg.Hosts))
	for _, u := range d.cfg.Hosts {
		states[u.Hostname()] = -1
	}

	ticker := time.NewTicker(d.cfg.CheckInterval)
	for {
		select {
		case <-d.ctx.Done():
			close(d.events)
			return

		case <-ticker.C:
			u := d.cfg.Hosts[0]
			d.cfg.Hosts = append(d.cfg.Hosts[1:], d.cfg.Hosts[0])

			hostname := u.Hostname()
			state := states[hostname]
			newState := d.check(u, state)

			if newState != state {
				e := types.Event{
					Hostname: hostname,
					State:    newState,
				}
				select {
				case <-d.ctx.Done():
					close(d.events)
					return
				case d.events <- e:
					states[hostname] = newState
				}
			}
		}
	}
}

func (d *HTTPDetector) check(u url.URL, state types.AppState) types.AppState {
	if u.Hostname() == "wikipedia.org" {
		i := 0
		i++
	}

	switch state {
	case types.AppState_AppAvailable:
		if d.checkHTTP(u) {
			return types.AppState_AppAvailable
		}

		if d.checkPort(u) {
			return types.AppState_PortAvailable
		}

		if d.checkHost(u) {
			return types.AppState_HostAvailable
		}

		return types.AppState_Unavailable

	case types.AppState_PortAvailable:
		if d.checkPort(u) {
			if d.checkHTTP(u) {
				return types.AppState_AppAvailable
			} else {
				return types.AppState_PortAvailable
			}
		} else {
			if d.checkHost(u) {
				return types.AppState_HostAvailable
			} else {
				return types.AppState_Unavailable
			}
		}

	default:
		if !d.checkHost(u) {
			return types.AppState_Unavailable
		}

		if !d.checkPort(u) {
			return types.AppState_HostAvailable
		}

		if !d.checkHTTP(u) {
			return types.AppState_PortAvailable
		}

		return types.AppState_AppAvailable
	}
}

func (d *HTTPDetector) checkHTTP(u url.URL) bool {
	cli := *http.DefaultClient
	cli.Timeout = d.cfg.HTTPTimeout

	resp, err := cli.Get(u.String())
	if err != nil {
		return false
	}

	return d.cfg.HealthCheck(resp)
}

func (d *HTTPDetector) checkPort(u url.URL) bool {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), d.cfg.TCPTimeout)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}

	return false
}

func (d *HTTPDetector) checkHost(u url.URL) bool {
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

	err = conn.SetReadDeadline(time.Now().Add(d.cfg.ICMPTimeout))
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
