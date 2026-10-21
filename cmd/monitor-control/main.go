// Command monitor-control is a monitor controller for Focusrite interfaces,
// operated from a Mackie Control surface, a Stream Deck or its own window.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/config"
	"github.com/sebastianrau/focusrite-mackie-control/internal/controller"
	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
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

	devices := make(catalog, 1)
	svc := newServices(ctx, store, ctrl, devices)
	svc.apply()

	app := ui.New(ui.Options{
		Version:   version,
		Submit:    ctrl.Submit,
		Snapshots: ctrl.Subscribe().C(),
		Devices:   devices,
		Store:     store,
		OnStarted: platform.HideFromDock,
		Apply:     func() { go svc.apply() },
	})
	stopUI := context.AfterFunc(ctx, app.Quit)
	defer stopUI()

	app.Run() // blocks until quit from the tray or a signal
	cancel()
	svc.stop()
	waitTimeout(&wg, shutdownTimeout)
	mcu.CloseDriver()

	store.Update(func(c *config.Config) { c.SetState(ctrl.Snapshot().State) })
	return store.Save()
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
