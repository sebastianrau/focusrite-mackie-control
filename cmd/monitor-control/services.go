package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/config"
	"github.com/sebastianrau/focusrite-mackie-control/internal/controller"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite"
	"github.com/sebastianrau/focusrite-mackie-control/internal/mcu"
)

// service is a group of goroutines that can be stopped and replaced while
// the app keeps running.
type service struct {
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	cleanup []func()
}

func startService(parent context.Context, runs ...func(context.Context)) *service {
	ctx, cancel := context.WithCancel(parent)
	s := &service{cancel: cancel}
	for _, run := range runs {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			run(ctx)
		}()
	}
	return s
}

// stop cancels the service and waits for it. A nil service is a no-op.
func (s *service) stop() {
	if s == nil {
		return
	}
	s.cancel()
	waitTimeout(&s.wg, shutdownTimeout)
	for _, f := range s.cleanup {
		f()
	}
}

// catalog is the device list for the settings window. It outlives the
// Focusrite service, which is replaced when its settings change.
type catalog chan []focusrite.DeviceInfo

// publish keeps only the latest list. Only one Focusrite service runs at a
// time, so there is a single sender.
func (c catalog) publish(list []focusrite.DeviceInfo) {
	select {
	case <-c:
	default:
	}
	c <- list
}

// services runs the Focusrite and control surface connections and restarts
// them when their settings change.
type services struct {
	ctx   context.Context
	store *config.Store

	startFocusrite func(context.Context, focusrite.Config) *service
	startMCU       func(context.Context, mcu.Config) *service

	mu        sync.Mutex
	fc        *service
	mcu       *service
	mcuConfig mcu.Config
	// fcConfig is what the running Focusrite service uses. It changes when
	// the service adopts an interface, from the service's own goroutine.
	fcConfig atomic.Pointer[focusrite.Config]
}

func newServices(ctx context.Context, store *config.Store, ctrl *controller.Controller, devices catalog) *services {
	s := &services{ctx: ctx, store: store}
	s.startFocusrite = func(ctx context.Context, cfg focusrite.Config) *service {
		client := focusrite.NewClient()
		sub := ctrl.Subscribe()
		a := focusrite.NewAdapter(cfg, client.Events(), client, ctrl.Submit, sub.C())
		a.OnAdopt = func(c focusrite.Config) {
			s.fcConfig.Store(&c)
			store.Update(func(cfg *config.Config) { cfg.SetFocusrite(c) })
			if err := store.Save(); err != nil {
				log.Error("saving adopted interface failed", "err", err)
			}
		}
		forward := func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case list := <-a.Devices():
					devices.publish(list)
				}
			}
		}
		svc := startService(ctx, client.Run, a.Run, forward)
		svc.cleanup = append(svc.cleanup, func() { ctrl.Unsubscribe(sub) })
		return svc
	}
	s.startMCU = func(ctx context.Context, cfg mcu.Config) *service {
		if cfg.InputPort == "" || cfg.OutputPort == "" {
			log.Info("no MIDI ports configured, control surface disabled")
			return nil
		}
		surface := mcu.NewMidiSurface(cfg.InputPort, cfg.OutputPort)
		sub := ctrl.Subscribe()
		svc := startService(ctx, surface.Run, mcu.NewAdapter(cfg, surface, ctrl.Submit, sub.C()).Run)
		svc.cleanup = append(svc.cleanup, func() { ctrl.Unsubscribe(sub) })
		return svc
	}
	return s
}

// apply starts the services, or restarts those whose settings changed.
func (s *services) apply() {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.store.Get()
	first := s.fc == nil

	if fc := cfg.FocusriteConfig(); first || *s.fcConfig.Load() != fc {
		if !first {
			log.Info("Focusrite settings changed, reconnecting")
		}
		s.fc.stop()
		s.fcConfig.Store(&fc)
		s.fc = s.startFocusrite(s.ctx, fc)
	}
	if mc := cfg.MCUConfig(); first || mc != s.mcuConfig {
		if !first {
			log.Info("MIDI settings changed, reconnecting")
		}
		s.mcu.stop()
		s.mcuConfig = mc
		s.mcu = s.startMCU(s.ctx, mc)
	}
}

func (s *services) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fc.stop()
	s.mcu.stop()
}

func waitTimeout(wg *sync.WaitGroup, d time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(d):
		log.Warn("stopping timed out, continuing anyway")
	}
}
