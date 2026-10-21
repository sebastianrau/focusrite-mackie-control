// Command monitor-control is a monitor controller for Focusrite interfaces,
// operated from a Mackie Control surface, a Stream Deck or its own window.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/config"
	"github.com/sebastianrau/focusrite-mackie-control/internal/controller"
	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite"
	"github.com/sebastianrau/focusrite-mackie-control/internal/logging"
	"github.com/sebastianrau/focusrite-mackie-control/internal/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/internal/platform"
	"github.com/sebastianrau/focusrite-mackie-control/internal/ui"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	autoSaveInterval = 30 * time.Second
	shutdownTimeout  = 3 * time.Second
)

var log = logging.New("main")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "monitor-control:", err)
		os.Exit(1)
	}
}

func run() error {
	defaultPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	path := flag.String("config", defaultPath, "settings file")
	flag.Parse()
	log.Info("starting Monitor Controller", "version", version, "config", *path)

	store, err := config.Open(*path)
	if err != nil {
		return err
	}
	cfg := store.Get()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup
	start := func(run func(context.Context)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(ctx)
		}()
	}

	ctrl := controller.New(cfg.State())
	start(ctrl.Run)
	start(persist(store, ctrl.Subscribe()))
	start(func(ctx context.Context) { store.AutoSave(ctx, autoSaveInterval) })

	client := focusrite.NewClient()
	fc := focusrite.NewAdapter(cfg.FocusriteConfig(), client.Events(), client, ctrl.Submit, ctrl.Subscribe().C())
	fc.OnAdopt = func(c focusrite.Config) {
		store.Update(func(cfg *config.Config) { cfg.SetFocusrite(c) })
		if err := store.Save(); err != nil {
			log.Error("saving adopted interface failed", "err", err)
		}
	}
	start(client.Run)
	start(fc.Run)

	if mc := cfg.MCUConfig(); mc.InputPort != "" && mc.OutputPort != "" {
		surface := mcu.NewMidiSurface(mc.InputPort, mc.OutputPort)
		start(surface.Run)
		start(mcu.NewAdapter(mc, surface, ctrl.Submit, ctrl.Subscribe().C()).Run)
	} else {
		log.Info("no MIDI ports configured, control surface disabled")
	}

	var restart atomic.Bool
	app := ui.New(ui.Options{
		Version:   version,
		Submit:    ctrl.Submit,
		Snapshots: ctrl.Subscribe().C(),
		Devices:   fc.Devices(),
		Store:     store,
		OnStarted: platform.HideFromDock,
		Restart: func() {
			restart.Store(true)
			cancel()
		},
	})
	stopUI := context.AfterFunc(ctx, app.Quit)
	defer stopUI()

	app.Run() // blocks until quit from the tray, a signal or a restart
	cancel()
	waitTimeout(&wg, shutdownTimeout)

	store.Update(func(c *config.Config) { c.SetState(ctrl.Snapshot().State) })
	if err := store.Save(); err != nil {
		return err
	}
	if restart.Load() {
		return relaunch()
	}
	return nil
}

// persist copies every state change into the settings store.
func persist(store *config.Store, sub *controller.Subscription) func(context.Context) {
	return func(ctx context.Context) {
		var last domain.State
		for {
			select {
			case <-ctx.Done():
				return
			case s := <-sub.C():
				if s.State != last {
					last = s.State
					store.Update(func(c *config.Config) { c.SetState(s.State) })
				}
			}
		}
	}
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
		log.Warn("shutdown timed out, exiting anyway")
	}
}

func relaunch() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	log.Info("restarting", "exe", exe)
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Start()
}
