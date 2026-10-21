package osc

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/logging"
)

var log = logging.New("osc")

const (
	// DefaultPort is a common OSC port; TouchOSC and others suggest it.
	DefaultPort = 9000

	meterInterval   = 50 * time.Millisecond
	refreshInterval = 5 * time.Second
	subscriptionTTL = 60 * time.Second
	retryDelay      = 5 * time.Second
)

type Config struct {
	Port int
	// AllowNetwork listens on all interfaces instead of only on this computer.
	// OSC has no authentication, so this is off by default.
	AllowNetwork bool
	// Targets always receive the state, as "host:port".
	Targets []string
}

func DefaultConfig() Config { return Config{Port: DefaultPort} }

// Adapter is an OSC server. All fields are owned by the Run goroutine.
type Adapter struct {
	cfg    Config
	submit func(domain.Command)
	snaps  <-chan domain.Snapshot

	listening chan net.Addr // receives the local address once listening

	conn    net.PacketConn
	targets []net.Addr
	subs    map[string]subscriber
	snap    domain.Snapshot
	ready   bool
	sent    map[string]string // last value sent per address
	meter   domain.Levels     // last levels sent
}

type subscriber struct {
	addr    net.Addr
	expires time.Time
}

type packet struct {
	data []byte
	from net.Addr
}

func NewAdapter(cfg Config, submit func(domain.Command), snaps <-chan domain.Snapshot) *Adapter {
	return &Adapter{
		cfg:       cfg,
		submit:    submit,
		snaps:     snaps,
		listening: make(chan net.Addr, 1),
		subs:      make(map[string]subscriber),
		sent:      make(map[string]string),
		meter:     domain.Silence,
	}
}

// Listening delivers the local address once the server is bound.
func (a *Adapter) Listening() <-chan net.Addr { return a.listening }

func (a *Adapter) listenAddr() string {
	host := "127.0.0.1"
	if a.cfg.AllowNetwork {
		host = ""
	}
	return net.JoinHostPort(host, strconv.Itoa(a.cfg.Port))
}

func (a *Adapter) Run(ctx context.Context) {
	for {
		var lc net.ListenConfig
		conn, err := lc.ListenPacket(ctx, "udp", a.listenAddr())
		if err == nil {
			a.serve(ctx, conn)
			return
		}
		log.Error("OSC port unavailable, retrying", "addr", a.listenAddr(), "err", err, "every", retryDelay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
		}
	}
}

func (a *Adapter) serve(ctx context.Context, conn net.PacketConn) {
	a.conn = conn
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	a.resolveTargets()
	log.Info("OSC server listening", "addr", conn.LocalAddr(), "targets", len(a.targets))
	a.listening <- conn.LocalAddr()

	packets := make(chan packet, 64)
	go func() {
		buf := make([]byte, 65536)
		for {
			n, from, err := conn.ReadFrom(buf)
			if err != nil {
				close(packets)
				return
			}
			select {
			case packets <- packet{data: append([]byte(nil), buf[:n]...), from: from}:
			default:
				log.Warn("OSC input queue full, dropping packet")
			}
		}
	}()

	meter := time.NewTicker(meterInterval)
	defer meter.Stop()
	refresh := time.NewTicker(refreshInterval)
	defer refresh.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case p, ok := <-packets:
			if !ok {
				return
			}
			a.receive(p)
		case s := <-a.snaps:
			a.snap, a.ready = s, true
			a.publishChanges()
		case <-meter.C:
			a.publishMeter()
		case <-refresh.C:
			a.expireSubscribers()
			a.sendAll(a.recipients())
		}
	}
}

func (a *Adapter) resolveTargets() {
	a.targets = nil
	for _, t := range a.cfg.Targets {
		addr, err := net.ResolveUDPAddr("udp", t)
		if err != nil {
			log.Warn("ignoring invalid OSC target", "target", t, "err", err)
			continue
		}
		if !a.cfg.AllowNetwork && !addr.IP.IsLoopback() {
			log.Warn("ignoring OSC target on another computer while network access is off", "target", t)
			continue
		}
		a.targets = append(a.targets, addr)
	}
}

func (a *Adapter) receive(p packet) {
	msgs, err := Decode(p.data)
	if err != nil {
		log.Debug("invalid OSC packet", "from", p.from, "err", err)
		return
	}
	for _, m := range msgs {
		switch m.Address {
		case AddrSync:
			a.sendAll(append(a.recipients(), p.from))
		case AddrSubscribe:
			addr := replyAddr(p.from, m)
			_, known := a.subs[addr.String()]
			a.subs[addr.String()] = subscriber{addr: addr, expires: time.Now().Add(subscriptionTTL)}
			if !known {
				log.Info("OSC client subscribed", "addr", addr)
				a.sendAll([]net.Addr{addr})
			}
		case AddrUnsubscribe:
			delete(a.subs, replyAddr(p.from, m).String())
		default:
			if cmd, ok := command(m); ok {
				a.submit(cmd)
			} else {
				log.Debug("unhandled OSC message", "msg", m.String(), "from", p.from)
			}
		}
	}
}

// replyAddr is the sender's address, with the port from the first argument
// if given (clients often receive on another port than they send from).
func replyAddr(from net.Addr, m Message) net.Addr {
	udp, ok := from.(*net.UDPAddr)
	if !ok {
		return from
	}
	reply := *udp
	if port, ok := m.Float(); ok && port > 0 && port < 65536 {
		reply.Port = int(port)
	}
	return &reply
}

func (a *Adapter) expireSubscribers() {
	now := time.Now()
	for k, s := range a.subs {
		if now.After(s.expires) {
			log.Info("OSC client subscription expired", "addr", s.addr)
			delete(a.subs, k)
		}
	}
}

func (a *Adapter) recipients() []net.Addr {
	seen := map[string]bool{}
	var out []net.Addr
	add := func(addr net.Addr) {
		if !seen[addr.String()] {
			seen[addr.String()] = true
			out = append(out, addr)
		}
	}
	for _, t := range a.targets {
		add(t)
	}
	for _, s := range a.subs {
		add(s.addr)
	}
	return out
}

// publishChanges sends the state messages whose values changed.
func (a *Adapter) publishChanges() {
	var changed []Message
	for _, m := range stateMessages(a.snap) {
		v := fmt.Sprint(m.Args...)
		if a.sent[m.Address] != v {
			a.sent[m.Address] = v
			changed = append(changed, m)
		}
	}
	a.send(a.recipients(), changed)
}

func (a *Adapter) publishMeter() {
	if !a.ready || a.snap.Levels == a.meter {
		return
	}
	a.meter = a.snap.Levels
	a.send(a.recipients(), meterMessages(a.meter))
}

// sendAll sends the complete state, including the meter, to the given
// recipients.
func (a *Adapter) sendAll(to []net.Addr) {
	if !a.ready {
		return
	}
	a.send(to, append(stateMessages(a.snap), meterMessages(a.snap.Levels)...))
}

func (a *Adapter) send(to []net.Addr, msgs []Message) {
	for _, m := range msgs {
		b, err := Encode(m)
		if err != nil {
			log.Error("encoding OSC message failed", "msg", m.String(), "err", err)
			continue
		}
		for _, addr := range to {
			if _, err := a.conn.WriteTo(b, addr); err != nil {
				log.Debug("sending OSC message failed", "to", addr, "err", err)
			}
		}
	}
}
