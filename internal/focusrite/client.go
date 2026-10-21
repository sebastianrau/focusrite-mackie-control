// Package focusrite connects the controller to Focusrite Control: Client
// speaks the network protocol, Adapter maps it to domain commands and state.
package focusrite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite/protocol"
	"github.com/sebastianrau/focusrite-mackie-control/internal/logging"
)

var log = logging.New("focusrite")

const (
	// Focusrite Control remembers approved clients by hostname and key, so
	// these must stay stable across versions.
	clientHostname = "Monitor Controller"
	clientKey      = "123456789"

	retryDelay        = 5 * time.Second
	keepAliveInterval = 3 * time.Second
	// Outgoing sets are collected and sent at most this often; Focusrite
	// Control drops updates that arrive faster.
	flushInterval = 75 * time.Millisecond
)

// Event is something the client observed on the connection.
type Event interface{ isEvent() }

type ServerConnected struct{ Connected bool }
type Approved struct{ Authorised bool }
type DeviceArrived struct{ Device *protocol.Device }
type DeviceRemoved struct{ ID int }
type DeviceUpdated struct{ Set protocol.Set }

func (ServerConnected) isEvent() {}
func (Approved) isEvent()        {}
func (DeviceArrived) isEvent()   {}
func (DeviceRemoved) isEvent()   {}
func (DeviceUpdated) isEvent()   {}

// Client keeps a connection to Focusrite Control, reconnecting as needed.
type Client struct {
	connect    func(context.Context) (net.Conn, error)
	retryDelay time.Duration
	events     chan Event
	sends      chan protocol.Set
}

func NewClient() *Client {
	return &Client{
		connect:    dialFocusriteControl,
		retryDelay: retryDelay,
		events:     make(chan Event, 64),
		sends:      make(chan protocol.Set, 64),
	}
}

// Events delivers connection events. It is closed when Run returns.
func (c *Client) Events() <-chan Event { return c.events }

// Send queues a set for the device. Sets are dropped while disconnected; the
// adapter pushes the full state again when the device reappears.
func (c *Client) Send(set protocol.Set) {
	select {
	case c.sends <- set:
	default:
		log.Warn("send queue full, dropping set", "items", len(set.Items))
	}
}

// Run connects and serves until ctx is cancelled.
func (c *Client) Run(ctx context.Context) {
	defer close(c.events)
	failing := false
	for {
		conn, err := c.connect(ctx)
		if err == nil {
			failing = false
			log.Info("connected to Focusrite Control", "addr", conn.RemoteAddr())
			err = c.serve(ctx, conn)
			c.emit(ctx, ServerConnected{Connected: false})
		}
		if ctx.Err() != nil {
			return
		}
		if !failing {
			log.Info("Focusrite Control not reachable, retrying", "err", err, "every", c.retryDelay)
			failing = true
		} else {
			log.Debug("Focusrite Control still not reachable", "err", err)
		}
		if !c.wait(ctx) {
			return
		}
	}
}

// wait sleeps for the retry delay, discarding sets queued meanwhile.
func (c *Client) wait(ctx context.Context) bool {
	t := time.NewTimer(c.retryDelay)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
			return true
		case <-c.sends:
		}
	}
}

func (c *Client) emit(ctx context.Context, e Event) {
	select {
	case c.events <- e:
	case <-ctx.Done():
	}
}

func (c *Client) serve(ctx context.Context, conn net.Conn) error {
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	if err := write(conn, protocol.ClientDetails{Hostname: clientHostname, ClientKey: clientKey}); err != nil {
		return err
	}
	c.emit(ctx, ServerConnected{Connected: true})

	msgs := make(chan any)
	readErr := make(chan error, 1)
	done := make(chan struct{})
	defer close(done)
	go func() {
		r := protocol.NewReader(conn)
		for {
			body, err := r.ReadFrame()
			if err != nil {
				readErr <- err
				return
			}
			m, err := protocol.Decode(body)
			if err != nil {
				log.Debug("skipping message", "err", err)
				continue
			}
			select {
			case msgs <- m:
			case <-done:
				return
			}
		}
	}()

	keepAlive := time.NewTicker(keepAliveInterval)
	defer keepAlive.Stop()
	flush := time.NewTicker(flushInterval)
	defer flush.Stop()
	var out outbox

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-readErr:
			if errors.Is(err, io.EOF) {
				return errors.New("connection closed by Focusrite Control")
			}
			return err
		case m := <-msgs:
			if err := c.handle(ctx, conn, m); err != nil {
				return err
			}
		case set := <-c.sends:
			out.add(set)
		case <-flush.C:
			for _, set := range out.take() {
				if err := write(conn, set); err != nil {
					return err
				}
			}
		case <-keepAlive.C:
			if err := write(conn, protocol.KeepAlive{}); err != nil {
				return err
			}
		}
	}
}

func (c *Client) handle(ctx context.Context, conn net.Conn, m any) error {
	switch m := m.(type) {
	case protocol.DeviceArrival:
		dev := m.Device
		log.Info("device arrived", "model", dev.Model, "serial", dev.SerialNumber, "id", dev.ID)
		if err := write(conn, protocol.SubscribeMessage{DeviceID: dev.ID, Subscribe: true}); err != nil {
			return err
		}
		c.emit(ctx, DeviceArrived{Device: &dev})
	case protocol.DeviceRemoval:
		log.Info("device removed", "id", m.ID)
		c.emit(ctx, DeviceRemoved{ID: m.ID})
	case protocol.Set:
		c.emit(ctx, DeviceUpdated{Set: m})
	case protocol.Approval:
		c.emit(ctx, Approved{Authorised: m.Authorised})
	case protocol.ClientDetails:
		log.Debug("client id assigned", "id", m.ID)
	case protocol.KeepAlive:
	default:
		log.Debug("ignoring message", "type", fmt.Sprintf("%T", m))
	}
	return nil
}

func write(conn net.Conn, v any) error {
	b, err := protocol.Encode(v)
	if err != nil {
		return err
	}
	_, err = conn.Write(b)
	return err
}

// outbox merges queued sets per device: the last value per item ID wins and
// keeps the position of its first occurrence.
type outbox struct {
	sets []protocol.Set
}

func (o *outbox) add(set protocol.Set) {
	if set.DevID == 0 {
		return
	}
	i := 0
	for i < len(o.sets) && o.sets[i].DevID != set.DevID {
		i++
	}
	if i == len(o.sets) {
		o.sets = append(o.sets, protocol.Set{DevID: set.DevID})
	}
	target := &o.sets[i]
next:
	for _, it := range set.Items {
		if it.ID == 0 {
			continue
		}
		for j := range target.Items {
			if target.Items[j].ID == it.ID {
				target.Items[j].Value = it.Value
				continue next
			}
		}
		target.Items = append(target.Items, it)
	}
}

func (o *outbox) take() []protocol.Set {
	sets := o.sets
	o.sets = nil
	return sets
}

var discoveryPorts = []int{30096, 30097, 30098}

func dialFocusriteControl(ctx context.Context) (net.Conn, error) {
	port, err := discover(ctx, discoveryPorts, time.Second)
	if err != nil {
		return nil, err
	}
	var d net.Dialer
	return d.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
}

// discover asks Focusrite Control for its TCP port on the local discovery
// ports.
func discover(ctx context.Context, ports []int, timeout time.Duration) (int, error) {
	req, err := protocol.Encode(protocol.DiscoveryRequest{App: "SAFFIRE-CONTROL", Version: "4", Device: "iOS"})
	if err != nil {
		return 0, err
	}
	var errs []error
	for _, p := range ports {
		port, err := discoverOn(ctx, p, req, timeout)
		if err == nil {
			return port, nil
		}
		errs = append(errs, err)
	}
	return 0, fmt.Errorf("discovery failed: %w", errors.Join(errs...))
}

func discoverOn(ctx context.Context, port int, req []byte, timeout time.Duration) (int, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return 0, err
	}
	if _, err := conn.Write(req); err != nil {
		return 0, err
	}
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, err
	}
	m, err := protocol.DecodeFrame(buf[:n])
	if err != nil {
		return 0, err
	}
	a, ok := m.(protocol.ServerAnnouncement)
	if !ok || a.Port == 0 {
		return 0, fmt.Errorf("unexpected discovery reply %T", m)
	}
	return a.Port, nil
}
