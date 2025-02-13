package main

import (
	"os"
	"os/signal"

	"fyne.io/fyne/v2"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	fcaudioconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-connector"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	mcuconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/mcu-connector"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

const Version string = "v0.0.1"

var log *logger.CustomLogger = logger.WithPackage("main")

// TODO Add Load balacer to FC Interface
// TODO Update Config File and use

// TODO Config: store and reload last state

// TODO Gui: Context Menu
// TODO Gui: add Hotkeys 1..5, D, M + Space

func main() {
	var (
		cfg *config.Config
	)

	log.Infof("Monitor Controller %v", Version)

	cfg, err := config.Load()

	if err != nil {
		log.Errorln("Loading configuration failed. Loading default values")
		cfg = config.Default()
		err := cfg.Save() // HACK fix me later
		if err != nil {
			log.Errorln("Configuration could not be stored")
		}
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	app, window, err := gui.MakeApp()
	if err != nil {
		fyne.LogError("Loading App error: ", err)
		os.Exit(-1)
	}

	mainGui, content := gui.NewAppWindow(app, -127, 0)
	mcu := mcuconnector.NewMcuConnector(mcuconnector.DefaultConfiguration())                //HACK remove default config
	fc := fcaudioconnector.NewAudioDeviceConnector(fcaudioconnector.DefaultConfiguration()) //HACK remove default config

	mc := monitorcontroller.NewController(fc)

	mc.
		RegisterRemoteController(mcu).
		RegisterRemoteController(mainGui)

	go func() {
		for range interrupt {

			os.Exit(0)
		}
	}()

	window.SetContent(content)
	window.ShowAndRun()
}
