package main

import (
	"context"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/sebastianrau/focusrite-mackie-control/internal/config"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite"
	"github.com/sebastianrau/focusrite-mackie-control/internal/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/internal/osc"
)

// fakeServices records starts and stops instead of opening connections.
type fakeServices struct {
	mu      sync.Mutex
	started []string
	stopped []string
}

func (f *fakeServices) service(ctx context.Context, name string) *service {
	f.mu.Lock()
	f.started = append(f.started, name)
	f.mu.Unlock()
	return startService(ctx, func(ctx context.Context) {
		<-ctx.Done()
		f.mu.Lock()
		f.stopped = append(f.stopped, name)
		f.mu.Unlock()
	})
}

func (f *fakeServices) take() (started, stopped []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	started, stopped = f.started, f.stopped
	f.started, f.stopped = nil, nil
	return started, stopped
}

func newTestServices(t *testing.T) (*services, *config.Store, *fakeServices) {
	t.Helper()
	store, err := config.Open(filepath.Join(t.TempDir(), "settings.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	fake := &fakeServices{}
	s := &services{ctx: ctx, store: store}
	s.startFocusrite = func(ctx context.Context, cfg focusrite.Config) *service {
		return fake.service(ctx, "focusrite:"+cfg.SerialNumber)
	}
	s.startMCU = func(ctx context.Context, cfg mcu.Config) *service {
		return fake.service(ctx, "mcu:"+cfg.InputPort)
	}
	s.startOSC = func(ctx context.Context, cfg osc.Config) *service {
		return fake.service(ctx, "osc:"+strconv.Itoa(cfg.Port))
	}
	return s, store, fake
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestApplyRestartsOnlyChangedServices(t *testing.T) {
	s, store, fake := newTestServices(t)

	s.apply()
	started, _ := fake.take()
	if want := []string{"focusrite:", "mcu:IAC StreamDeckToController", "osc:9000"}; !equal(started, want) {
		t.Fatalf("first apply started %v, want %v", started, want)
	}

	s.apply() // nothing changed
	if started, stopped := fake.take(); len(started)+len(stopped) != 0 {
		t.Errorf("unchanged settings restarted %v / stopped %v", started, stopped)
	}

	store.Update(func(c *config.Config) { c.MIDI.InputPort = "Surface" })
	s.apply()
	started, stopped := fake.take()
	if !equal(started, []string{"mcu:Surface"}) || !equal(stopped, []string{"mcu:IAC StreamDeckToController"}) {
		t.Errorf("MIDI change: started %v, stopped %v", started, stopped)
	}

	store.Update(func(c *config.Config) { c.Focusrite.SerialNumber = "X1" })
	s.apply()
	started, stopped = fake.take()
	if !equal(started, []string{"focusrite:X1"}) || !equal(stopped, []string{"focusrite:"}) {
		t.Errorf("Focusrite change: started %v, stopped %v", started, stopped)
	}

	store.Update(func(c *config.Config) { c.OSC.Port = 8000 })
	s.apply()
	started, stopped = fake.take()
	if !equal(started, []string{"osc:8000"}) || !equal(stopped, []string{"osc:9000"}) {
		t.Errorf("OSC change: started %v, stopped %v", started, stopped)
	}

	store.Update(func(c *config.Config) { c.OSC.Enabled = false })
	s.apply()
	started, stopped = fake.take()
	if len(started) != 0 || !equal(stopped, []string{"osc:8000"}) {
		t.Errorf("OSC off: started %v, stopped %v", started, stopped)
	}

	s.stop()
	if _, stopped := fake.take(); len(stopped) != 2 {
		t.Errorf("stop stopped %v", stopped)
	}
}

// Adopting an interface stores its configuration; that must not count as a
// change on the next apply.
func TestAdoptionIsNotAChange(t *testing.T) {
	s, store, fake := newTestServices(t)
	s.apply()
	fake.take()

	adopted := focusrite.Config{SerialNumber: "P9EAC6K250F325", MuteID: 1679}
	s.fcConfig.Store(&adopted) // what OnAdopt does
	store.Update(func(c *config.Config) { c.SetFocusrite(adopted) })

	s.apply()
	if started, _ := fake.take(); len(started) != 0 {
		t.Errorf("adoption restarted %v", started)
	}
}

func TestCatalogKeepsLatest(t *testing.T) {
	c := make(catalog, 1)
	c.publish([]focusrite.DeviceInfo{{SerialNumber: "A"}})
	c.publish([]focusrite.DeviceInfo{{SerialNumber: "B"}})
	if got := <-c; got[0].SerialNumber != "B" {
		t.Errorf("got %v", got)
	}
}
