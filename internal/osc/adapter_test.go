package osc

import (
	"context"
	"net"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// client is a test OSC peer on localhost.
type client struct {
	t    *testing.T
	conn net.PacketConn
	srv  net.Addr
}

func newClient(t *testing.T, srv net.Addr) *client {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return &client{t: t, conn: conn, srv: srv}
}

func (c *client) port() int { return c.conn.LocalAddr().(*net.UDPAddr).Port }

func (c *client) send(addr string, args ...any) {
	c.t.Helper()
	b, err := Encode(Message{Address: addr, Args: args})
	if err != nil {
		c.t.Fatal(err)
	}
	if _, err := c.conn.WriteTo(b, c.srv); err != nil {
		c.t.Fatal(err)
	}
}

// await reads until a message for addr arrives and returns its arguments.
func (c *client) await(addr string) []any {
	c.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	buf := make([]byte, 65536)
	for {
		_ = c.conn.SetReadDeadline(deadline)
		n, _, err := c.conn.ReadFrom(buf)
		if err != nil {
			c.t.Fatalf("no %s received: %v", addr, err)
		}
		msgs, err := Decode(buf[:n])
		if err != nil {
			c.t.Fatal(err)
		}
		for _, m := range msgs {
			if m.Address == addr {
				return m.Args
			}
		}
	}
}

type harness struct {
	a     *Adapter
	srv   net.Addr
	snaps chan domain.Snapshot
	mu    sync.Mutex
	cmds  []domain.Command
}

func start(t *testing.T, cfg Config) *harness {
	t.Helper()
	h := &harness{snaps: make(chan domain.Snapshot, 1)}
	h.a = NewAdapter(cfg, func(c domain.Command) {
		h.mu.Lock()
		h.cmds = append(h.cmds, c)
		h.mu.Unlock()
	}, h.snaps)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { h.a.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case h.srv = <-h.a.Listening():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}
	h.snaps <- domain.Snapshot{State: domain.DefaultState(), Levels: domain.Silence,
		Device: domain.Device{ServerConnected: true, Present: true, Model: "Scarlett 18i20 (3rd Gen)"}}
	return h
}

func (h *harness) waitCmds(t *testing.T, n int) []domain.Command {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		got := slices.Clone(h.cmds)
		h.mu.Unlock()
		if len(got) >= n {
			return got
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("expected %d commands", n)
	return nil
}

func TestInputBecomesCommands(t *testing.T) {
	h := start(t, Config{})
	c := newClient(t, h.srv)

	c.send(AddrVolume, float32(-20))
	c.send(AddrVolumeNormalized, float32(1))
	c.send(AddrMute, int32(0))
	c.send(AddrDim, true)
	c.send(AddrMuteToggle, int32(1))
	c.send(AddrMuteToggle, int32(0)) // release of a momentary button: ignored
	c.send(SpeakerAddr(domain.SpeakerB), int32(1))
	c.send(SpeakerToggle(domain.Sub), float32(1))
	c.send("/monitor/speaker/zz", int32(1)) // unknown speaker: ignored
	c.send("/something/else")

	want := []domain.Command{
		domain.SetVolume{DB: -20},
		domain.SetVolume{DB: 0},
		domain.SetMute{On: false},
		domain.SetDim{On: true},
		domain.ToggleMute{},
		domain.SelectSpeaker{ID: domain.SpeakerB, On: true},
		domain.ToggleSpeaker{ID: domain.Sub},
	}
	got := h.waitCmds(t, len(want))
	time.Sleep(50 * time.Millisecond) // nothing else may follow
	h.mu.Lock()
	defer h.mu.Unlock()
	if !slices.Equal(h.cmds, want) {
		t.Errorf("commands\n%#v\nwant\n%#v", got, want)
	}
}

func TestSubscribeGetsStateAndChanges(t *testing.T) {
	h := start(t, Config{})
	c := newClient(t, h.srv)
	c.send(AddrSubscribe) // reply to the sending port

	if got := c.await(AddrDeviceModel); got[0] != "Scarlett 18i20 (3rd Gen)" {
		t.Errorf("model = %v", got)
	}
	if got := c.await(SpeakerNameAddr(domain.SpeakerC)); got[0] != "Speaker C" {
		t.Errorf("speaker C name = %v", got)
	}

	s := domain.Snapshot{State: domain.DefaultState(), Device: domain.Device{ServerConnected: true, Present: true}}
	s.State.Master.VolumeDB = -30
	s.Levels = domain.Levels{LeftDB: -12, RightDB: -60}
	h.snaps <- s
	if got := c.await(AddrVolume); got[0] != float32(-30) {
		t.Errorf("volume = %v", got)
	}
	if got := c.await(AddrMeterLeftNormalized); got[0] != float32(0.8) {
		t.Errorf("meter left = %v, want 0.8", got)
	}
}

func TestSubscribeWithReplyPort(t *testing.T) {
	h := start(t, Config{})
	sender := newClient(t, h.srv)
	receiver := newClient(t, h.srv)
	sender.send(AddrSubscribe, int32(receiver.port()))
	receiver.await(AddrMute)
}

func TestStaticTargetAndSync(t *testing.T) {
	target := newClient(t, nil)
	h := start(t, Config{Targets: []string{"127.0.0.1:" + strconv.Itoa(target.port()), "not a target"}})
	target.srv = h.srv

	other := newClient(t, h.srv)
	other.send(AddrSync)
	other.await(AddrVolumeNormalized)  // the sender gets the state
	target.await(AddrVolumeNormalized) // and so does the target
}

func TestRemoteTargetsNeedNetworkAccess(t *testing.T) {
	a := NewAdapter(Config{Targets: []string{"192.0.2.1:9001", "127.0.0.1:9001"}}, nil, nil)
	a.resolveTargets()
	if len(a.targets) != 1 {
		t.Errorf("targets = %v, want only the local one", a.targets)
	}
	a.cfg.AllowNetwork = true
	a.resolveTargets()
	if len(a.targets) != 2 {
		t.Errorf("targets = %v, want both", a.targets)
	}
}

func TestVolumeScaling(t *testing.T) {
	for _, db := range []int{domain.MinVolumeDB, -64, -1, 0} {
		if got := denormalizeVolume(float64(normalizeVolume(db))); got != db {
			t.Errorf("%d dB -> %v -> %d dB", db, normalizeVolume(db), got)
		}
	}
	if denormalizeVolume(-1) != domain.MinVolumeDB || denormalizeVolume(2) != domain.MaxVolumeDB {
		t.Error("out-of-range values must be clamped")
	}
}
