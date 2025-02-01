package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/go-vgo/robotgo"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	focusriteclient "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-client"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	monitorcontroller "github.com/sebastianrau/focusrite-mackie-control/pkg/monitor_controller"
)

const Version string = "v0.0.1"

// TODO : config file command line option
// TODO : Focusrite Control
// TODO : Context Menu

func main() {
	var (
		showMidi, configureMidi, showHelp bool
		cfg                               *config.Config
		waitGroup                         sync.WaitGroup
	)

	flag.BoolVar(&showMidi, "l", false, "List all installed MIDI devices")
	flag.BoolVar(&configureMidi, "c", false, "Configure and start")
	flag.BoolVar(&showHelp, "h", false, "Show Help")
	flag.Parse()

	log.Printf("Monitor Controller %v", Version)

	if showHelp {
		fmt.Println("Usage: monitor-controller [options]")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if configureMidi {
		c, success := config.UserConfigure()
		if !success {
			fmt.Println("Configuration failed.")
			os.Exit(1)
		}
		cfg = c
	}

	if cfg == nil {
		c, err := config.Load()
		if err != nil {
			fmt.Println("Loading configuration failed.")
			os.Exit(1)
		}
		cfg = c
	}

	if cfg.MidiInputPort == "" {
		fmt.Println("No Midi port configured.")
		os.Exit(-1)
	}

	fromMcu := make(chan interface{}, 100)
	toMcu := make(chan interface{}, 100)
	fromController := make(chan interface{}, 100)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	mcu.InitMcu(fromMcu, toMcu, interrupt, &waitGroup, *cfg)
	fc := focusriteclient.NewFocusriteClient()

	monitorcontroller.NewController(toMcu, fromMcu, fromController)

	//syst := systray.CreateSystray(cfg)

	for {

		select {
		//case frCon := <-fc.ConnectedChannel:
		case <-fc.ConnectedChannel:
			//log.Printf("Focusrite Connection State: %t\n", frCon)

		//case frDevice := <-fc.DataChannel:
		case <-fc.DataChannel:
			//fmt.Printf("Device Update %d (%s)\n", frDevice.ID, frDevice.SerialNumber)

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
				//fmt.Printf("%s: %v\n", reflect.TypeOf(fm), fm)

			}

		case <-interrupt:
			os.Exit(0)
		}

	}
}
