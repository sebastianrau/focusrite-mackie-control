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

	fromMcu := make(chan interface{}, 100)
	toMcu := make(chan interface{}, 100)
	fromController := make(chan interface{}, 100)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	mcu.InitMcu(fromMcu, toMcu, interrupt, &waitGroup, *cfg)
	fc := focusriteclient.NewFocusriteClient(focusriteclient.UpdateRaw)

	monitorcontroller.NewController(toMcu, fromMcu, fromController)

	//syst := systray.CreateSystray(cfg)

	for {

		select {
		case frCon := <-fc.ConnectedChannel:
			log.Infof("Focusrite Connection State: %t\n", frCon)

		case newDevice := <-fc.DeviceArrivalChannel:
			log.Infof("New Device Connected %d (%s)\n", newDevice.ID, newDevice.SerialNumber)

		case frDevice := <-fc.DeviceUpdateChannel:
			log.Infof("Device Update %d (%s)\n", frDevice.ID, frDevice.SerialNumber)

		case r := <-fc.RawUpdateChannel:
			log.Infof("Raw Update %d len: %d\n", r.DevID, len(r.Items))

		case fm := <-fromController:
			switch f := fm.(type) {

			case monitorcontroller.TransportMessage:
				switch f.Key {
				case gomcu.Play:
					robotgo.KeyTap(robotgo.AudioPlay)
				case gomcu.Stop:
					robotgo.KeyTap(robotgo.AudioStop)
				case gomcu.FastFwd:
					robotgo.KeyTap(robotgo.AudioNext)
				case gomcu.Rewind:
					robotgo.KeyTap(robotgo.AudioPrev)
				}
			default:
				log.Warnf("unhandled message from monitor-controller %s: %v\n", reflect.TypeOf(fm), fm)

			}

		case <-interrupt:
			os.Exit(0)
		}

	}
}
