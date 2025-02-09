package main

import (
	"os"
	"os/signal"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
	debuggerconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/remote-controller/debugger-connector"
)

const Version string = "v0.0.1"

var log *logger.CustomLogger = logger.WithPackage("main")

// TODO : config file command line option
// TODO : Context Menu

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

	mcu := mcu.InitMcu(cfg.Midi)
	// TODO add mcu remote controller
	// TODO add gui remote controller
	dbg := debuggerconnector.NewDebuggerConnector()

	mc := monitorcontroller.NewController(
		mcu.ToMcu, mcu.FromMcu,
		cfg.Controller)

	mc.
		RegisterRemoteController(dbg)

	for {
		select {
		/*case fm := <-control.FromController:
		switch f := fm.(type) {




		case monitorcontroller.MuteMessage:
			log.Infof("Mute: %t", f)
		case monitorcontroller.DimMessage:
			log.Infof("Dim: %t", f)
		case monitorcontroller.SpeakerEnabledMessage:
			log.Infof("Speaker %d set to %t", f.SpeakerID, f.SpeakerEnabled)

		default:
			log.Warnf("unhandled message from monitor-controller %s: %v\n", reflect.TypeOf(fm), fm)

		}
		*/
		case <-interrupt:

			os.Exit(0)
		}

	}
}
