package main

import (
	"os"
	"os/signal"

	fcaudioconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-connector"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
	mcuconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/mcu-connector"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

int
SetActivationPolicy(void) {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    return 0;
}
*/
import "C"

// Workaround for hiding app symbol and having only system tray
func setActivationPolicy() {
	log.Debugln("Setting ActivationPolicy")
	C.SetActivationPolicy()
}

const Version string = "v0.0.1"

var log *logger.CustomLogger = logger.WithPackage("main")

// TODO MUC: Check reconnection
// TODO Config: add configuration gui

func main() {
	var (
		cfg *config.Config
	)

	log.Infof("Monitor Controller %v", Version)

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
		setActivationPolicy()
	})

	mcu := mcuconnector.NewMcuConnector(cfg.Midi)
	if mcu == nil {
		log.Warnf("could not open Midi System")
	}

	fc := fcaudioconnector.NewAudioDeviceConnector(cfg.FocusriteDevice)
	if fc == nil {
		log.Errorf("Could not load Audio Connector")
		os.Exit(-1)
	}

	mc := monitorcontroller.NewController(fc, cfg.MonitorController)
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
