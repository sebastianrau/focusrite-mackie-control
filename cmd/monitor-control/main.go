package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	fcaudioconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-connector"
	guifix "github.com/sebastianrau/focusrite-mackie-control/pkg/gui-fix"
	mcuconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/mcu-connector"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

const Version string = "v0.0.1"

var log *logger.CustomLogger = logger.WithPackage("main")

// TODO MUC: Check reconnection
// TODO Config: add configuration gui

func main() {

	var (
		cfg     *config.Config
		closers []interface{ Close() error }
	)

	log.Infof("Monitor Controller %v", Version)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	cfg, err := config.Load()

	if err != nil {
		log.Errorln("Loading configuration failed. Loading default values")
		cfg = config.Default()
		err := cfg.Save()
		if err != nil {
			log.Errorln("Configuration could not be stored")
		}
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go cfg.RunAutoSave(ctx)

	var mainGui *gui.MainGui

	mainGui, err = gui.NewAppWindow(
		cfg,
		// On Close
		func() {
			err := cfg.Save()
			if err != nil {
				log.Error(err.Error())
			}
		})

	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	mainGui.Lifecycle().SetOnStarted(func() {
		guifix.SetActivationPolicy()
	})

	mcu, err := mcuconnector.NewMcuConnector(&cfg.Midi)
	if err != nil {
		log.Warnf("could not open Midi System. Midi system disabled.")
	} else {
		closers = append(closers, mcu)
	}

	fc, err := fcaudioconnector.NewAudioDeviceConnector(&cfg.FocusriteDevice)
	if err != nil {
		log.Errorf("Could not load Audio Connector")
		return
	}
	closers = append(closers, fc)

	mc, err := monitorcontroller.NewController(fc, &cfg.MonitorController)
	if err != nil {
		log.Errorf("Could not load monitor Controller")
		return
	}
	closers = append(closers, mc)

	if mcu != nil {
		mc.RegisterRemoteController(mcu)
	}

	if mainGui != nil {
		mc.RegisterRemoteController(mainGui)
	}

	go func() {
		for range interrupt {
			err := cfg.Save()
			if err != nil {
				log.Error(err.Error())
			}
			os.Exit(0)
		}
	}()

	go func() {
		<-ctx.Done()

		if err := cfg.Save(); err != nil {
			log.Error(err.Error())
		}

		for i := len(closers) - 1; i >= 0; i-- {
			_ = closers[i].Close()
		}

		if mainGui != nil {
			mainGui.Quit()
		}
	}()

	mainGui.ShowAndRun()
}
