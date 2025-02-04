package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"

	"github.com/go-vgo/robotgo"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	focusriteclient "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-client"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	monitorcontroller "github.com/sebastianrau/focusrite-mackie-control/pkg/monitor_controller"
	"github.com/sirupsen/logrus"
)

const Version string = "v0.0.1"

var log *logrus.Entry = logger.WithPackage("main")

// TODO : config file command line option
// TODO : Focusrite Control
// TODO : Context Menu

func main() {
	var (
		showMidi, configureMidi, showHelp bool
		cfg                               *config.Config
		waitGroup                         sync.WaitGroup
	)

	//	log := logger.Log.WithFields(logrus.Fields{"package": "main"})

	flag.BoolVar(&showMidi, "l", false, "List all installed MIDI devices")
	flag.BoolVar(&configureMidi, "c", false, "Configure and start")
	flag.BoolVar(&showHelp, "h", false, "Show Help")
	flag.Parse()
	log.Infof("Monitor Controller %v", Version)

	if showHelp {
		fmt.Println("Usage: monitor-controller [options]")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if configureMidi {
		c, success := config.UserConfigure()
		if !success {
			log.Errorln("Configuration failed.")
			os.Exit(1)
		}
		cfg = c
	}

	if cfg == nil {
		c, err := config.Load()
		if err != nil {
			log.Errorln("Loading configuration failed.")
			os.Exit(1)
		}
		cfg = c
	}

	if cfg.MidiInputPort == "" {
		log.Errorln("No Midi port configured.")
		os.Exit(-1)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	mcu := mcu.InitMcu(interrupt, &waitGroup, *cfg)

	fc := focusriteclient.NewFocusriteClient(focusriteclient.UpdateRaw)

	control := monitorcontroller.NewController(
		mcu.ToMcu, mcu.FromMcu,
		fc.ToFocusrite, fc.FromFocusrite)

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
				case gomcu.Stop:
					// err := robotgo.KeyTap(robotgo.Audio)
					// if err != nil {
					// 	log.Errorf("Keytab error %s", err.Error())
					// }
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
				}

			case monitorcontroller.MuteMessage:
				log.Infof("Mute: %t", f)
			case monitorcontroller.DimMessage:
				log.Infof("Dim: %t", f)
			default:
				log.Warnf("unhandled message from monitor-controller %s: %v\n", reflect.TypeOf(fm), fm)

			}

		case <-interrupt:
			os.Exit(0)
		}

	}
}
