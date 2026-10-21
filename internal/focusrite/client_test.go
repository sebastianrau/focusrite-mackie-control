package focusrite

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite/protocol"
)

// fakeServer plays Focusrite Control on a local TCP port.
type fakeServer struct {
	t     *testing.T
	ln    net.Listener
	conns chan net.Conn
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeServer{t: t, ln: ln, conns: make(chan net.Conn, 4)}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			s.conns <- c
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *fakeServer) client() *Client {
	c := NewClient()
	c.retryDelay = 20 * time.Millisecond
	c.connect = func(ctx context.Context) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "tcp", s.ln.Addr().String())
	}
	return c
}

func (s *fakeServer) accept() (net.Conn, *protocol.Reader) {
	s.t.Helper()
	select {
	case c := <-s.conns:
		s.t.Cleanup(func() { c.Close() })
		return c, protocol.NewReader(c)
	case <-time.After(2 * time.Second):
		s.t.Fatal("client did not connect")
		return nil, nil
	}
}

func send(t *testing.T, c net.Conn, v any) {
	t.Helper()
	b, err := protocol.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write(b); err != nil {
		t.Fatal(err)
	}
}

// expect reads messages until one of type T arrives, skipping keep-alives.
func expect[T any](t *testing.T, r *protocol.Reader) T {
	t.Helper()
	for {
		m, err := r.Next()
		if err != nil {
			t.Fatalf("waiting for %T: %v", *new(T), err)
		}
		if v, ok := m.(T); ok {
			return v
		}
	}
}

func event[T Event](t *testing.T, c *Client) T {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case e := <-c.Events():
			if v, ok := e.(T); ok {
				return v
			}
		case <-timeout:
			t.Fatalf("no %T event", *new(T))
		}
	}
}

func TestClientSession(t *testing.T) {
	srv := newFakeServer(t)
	c := srv.client()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)

	conn, r := srv.accept()
	if _, err := r.ReadFrame(); err != nil { // client-details
		t.Fatal(err)
	}
	if !event[ServerConnected](t, c).Connected {
		t.Fatal("expected connected")
	}

	arrival, err := os.ReadFile("protocol/testdata/device-arrival-18i20.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(frameBytes(arrival)); err != nil {
		t.Fatal(err)
	}
	if sub := expect[protocol.SubscribeMessage](t, r); sub.DeviceID != 3 || !sub.Subscribe {
		t.Errorf("subscribe = %+v", sub)
	}
	if d := event[DeviceArrived](t, c).Device; d.SerialNumber != "P9EAC6K250F325" {
		t.Errorf("arrived %q", d.SerialNumber)
	}

	send(t, conn, protocol.Set{DevID: 3, Items: []protocol.Item{protocol.BoolItem(1678, true)}})
	if set := event[DeviceUpdated](t, c).Set; set.Items[0].ID != 1678 {
		t.Errorf("update = %+v", set)
	}

	// Two quick sends are merged into one set with the last value.
	c.Send(protocol.Set{DevID: 3, Items: []protocol.Item{protocol.IntItem(1458, -30), protocol.BoolItem(1453, true)}})
	c.Send(protocol.Set{DevID: 3, Items: []protocol.Item{protocol.IntItem(1458, -20)}})
	got := expect[protocol.Set](t, r)
	want := []protocol.Item{protocol.IntItem(1458, -20), protocol.BoolItem(1453, true)}
	if len(got.Items) != 2 || got.Items[0] != want[0] || got.Items[1] != want[1] {
		t.Errorf("sent %+v, want %+v", got.Items, want)
	}

	// Losing the connection is reported and followed by a reconnect.
	conn.Close()
	if event[ServerConnected](t, c).Connected {
		t.Fatal("expected disconnected")
	}
	srv.accept()
	if !event[ServerConnected](t, c).Connected {
		t.Fatal("expected reconnected")
	}

	cancel()
	for range c.Events() {
	}
}

func TestOutbox(t *testing.T) {
	var o outbox
	o.add(protocol.Set{DevID: 0, Items: []protocol.Item{protocol.IntItem(1, 1)}}) // no device
	o.add(protocol.Set{DevID: 1, Items: []protocol.Item{protocol.IntItem(1, 1), protocol.IntItem(0, 9), protocol.IntItem(2, 2)}})
	o.add(protocol.Set{DevID: 2, Items: []protocol.Item{protocol.IntItem(5, 5)}})
	o.add(protocol.Set{DevID: 1, Items: []protocol.Item{protocol.IntItem(1, 3)}})

	sets := o.take()
	if len(sets) != 2 || sets[0].DevID != 1 || sets[1].DevID != 2 {
		t.Fatalf("sets = %+v", sets)
	}
	if want := []protocol.Item{protocol.IntItem(1, 3), protocol.IntItem(2, 2)}; len(sets[0].Items) != 2 || sets[0].Items[0] != want[0] || sets[0].Items[1] != want[1] {
		t.Errorf("device 1 items = %+v, want %+v", sets[0].Items, want)
	}
	if len(o.take()) != 0 {
		t.Error("take did not empty the outbox")
	}
}

func TestDiscover(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	go func() {
		buf := make([]byte, 1024)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil || !strings.Contains(string(buf[:n]), `<client-discovery app="SAFFIRE-CONTROL"`) {
			return
		}
		reply, _ := protocol.Encode(protocol.ServerAnnouncement{App: "SAFFIRE-CONTROL", Version: "4", Port: 55145})
		_, _ = pc.WriteTo(reply, addr)
	}()

	unused := freeUDPPort(t)
	port, err := discover(context.Background(), []int{unused, pc.LocalAddr().(*net.UDPAddr).Port}, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if port != 55145 {
		t.Errorf("port = %d", port)
	}
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port
	pc.Close()
	return port
}

func frameBytes(body []byte) []byte { return fmt.Appendf(nil, "Length=%06X %s", len(body), body) }
