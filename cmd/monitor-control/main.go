package main

import (
	"os"
	"os/signal"
	"reflect"
	"sync"

	"github.com/go-vgo/robotgo"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/focusriteclient"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
	"github.com/sebastianrau/gomcu"

	"github.com/sirupsen/logrus"
)

const Version string = "v0.0.1"

var log *logrus.Entry = logger.WithPackage("main")

// TODO : config file command line option
// TODO : Context Menu

func main() {
	var (
		cfg       *config.Config
		waitGroup sync.WaitGroup
	)

	log.Infof("Monitor Controller %v", Version)

	c, err := config.Load()
	if err == nil {
		cfg = c
	} else {
		log.Errorln("Loading configuration failed. Loading default values")
		cfg = config.Default()
		err := cfg.Save()
		if err != nil {
			log.Errorln("Configuration could not be stored")
		}
	}

	if cfg.Midi.MidiInputPort == "" {
		log.Errorln("No Midi port configured.")
		os.Exit(-1)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	mcu := mcu.InitMcu(interrupt, &waitGroup, *cfg.Midi)
	fc := focusriteclient.NewFocusriteClient(focusriteclient.UpdateRaw)

	control := monitorcontroller.NewController(
		mcu.ToMcu, mcu.FromMcu,
		fc.ToFocusrite, fc.FromFocusrite,
		cfg.Controller)

	for {
		select {
		case fm := <-control.FromController:
			switch f := fm.(type) {

			case monitorcontroller.TransportMessage:

				switch gomcu.Switch(f) {
				case gomcu.Play:
					err := robotgo.KeyTap(robotgo.AudioPlay)
					if err != nil {
						log.Errorf("Keytab error %s", err.Error())
					}
				case gomcu.FastFwd:
					err := robotgo.KeyTap(robotgo.AudioNext)
					if err != nil {
						log.Errorf("Keytab error %s", err.Error())
					}
				case gomcu.Rewind:
					err := robotgo.KeyTap(robotgo.AudioPrev)
					if err != nil {
						log.Errorf("Keytab error %s", err.Error())
					}
				case gomcu.Stop:
					// Ignore Stop
				}

			case monitorcontroller.MuteMessage:
				log.Infof("Mute: %t", f)
			case monitorcontroller.DimMessage:
				log.Infof("Dim: %t", f)
			case monitorcontroller.SpeakerEnabledMessage:
				log.Infof("Speaker %d set to %t", f.SpeakerID, f.SpeakerEnabled)

			default:
				log.Warnf("unhandled message from monitor-controller %s: %v\n", reflect.TypeOf(fm), fm)

			}

		case <-interrupt:
			os.Exit(0)
		}

	}
}
