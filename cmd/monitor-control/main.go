package main

import (
	"os"
	"os/signal"

	fcaudioconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-connector"
	mcuconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/mcu-connector"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
	"github.com/sebastianrau/focusrite-mackie-control/internal/platform"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

var log *logger.CustomLogger = logger.WithPackage("main")

// TODO MUC: Check reconnection
// TODO Config: add configuration gui

func main() {
	var (
		cfg *config.Config
	)

	log.Infof("Monitor Controller %v", version)

	cfg, err := config.Load()
	go cfg.RunAutoSave()

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
		platform.HideFromDock()
	})

	mcu := mcuconnector.NewMcuConnector(&cfg.Midi)
	if mcu == nil {
		log.Warnf("could not open Midi System")
	}

	fc := fcaudioconnector.NewAudioDeviceConnector(&cfg.FocusriteDevice)
	if fc == nil {
		log.Errorf("Could not load Audio Connector")
		os.Exit(-1)
	}

	mc := monitorcontroller.NewController(fc, &cfg.MonitorController)
	if mc == nil {
		log.Errorf("Could not load monitor Controller")
		os.Exit(-3)
	}

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

	mainGui.ShowAndRun()
}
