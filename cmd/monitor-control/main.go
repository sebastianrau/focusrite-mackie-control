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
)

const Version string = "v0.0.1"

var log *logger.CustomLogger = logger.WithPackage("main")

// TODO MUC: Check reconnection
// TODO Config: add configuration gui
// TODO Gui: Context Menu --> Select, Mute, Dim

func main() {
	var (
		cfg *config.Config
	)

	log.Infof("Monitor Controller %v", Version)

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

	mainGui, err := gui.NewAppWindow(
		func() {
			cfg.Save()
		},
		-127,
		0)
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	mcu := mcuconnector.NewMcuConnector(cfg.Midi)
	fc := fcaudioconnector.NewAudioDeviceConnector(cfg.FocusriteDevice)

	mc := monitorcontroller.NewController(fc, cfg.MonitorController)

	mc.
		RegisterRemoteController(mcu).
		RegisterRemoteController(mainGui)

	go func() {
		for range interrupt {
			cfg.Save()
			os.Exit(0)
		}
	}()

	mainGui.ShowAndRun()
}
