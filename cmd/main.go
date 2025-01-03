package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	monitorcontroller "github.com/sebastianrau/focusrite-mackie-control/pkg/monitor_controller"
)

const Version string = "v0.0.1"

// TODO: config file command line option
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
	controller := monitorcontroller.NewController(toMcu, fromMcu, fromController)

	go func() {
		time.Sleep(5 * time.Second)
		controller.SetMeter(monitorcontroller.MasterFader, 1.2)
	}()

	for {

		select {
		case fm := <-fromController:

			switch f := fm.(type) {
			case mcu.KeyMessage:
				fmt.Printf("%s: %v\n", reflect.TypeOf(fm), fm)

			case mcu.RawFaderMessage:
				fmt.Println(f)

			default:
				fmt.Printf("%s: %v\n", reflect.TypeOf(fm), fm)

			}

		case <-interrupt:
			os.Exit(0)
		}

	}
}

/*
func onExit() {
}

func onReady() {
	fromUser := make(chan interface{}, 100)
	systray.SetTemplateIcon(icon.Data, icon.Data)
	//systray.SetTitle("obs-mcu")
	systray.SetTooltip("obs-mcu")
	mOpenConfig := systray.AddMenuItem("Edit Config", "Open config file")
	systray.AddSeparator()
	mMidiInputs := systray.AddMenuItem("MIDI Input", "Select MIDI input (restart to apply)")
	mMidiOutputs := systray.AddMenuItem("MIDI Output", "Select MIDI output (restart to apply)")
	systray.AddSeparator()
	mSettings := systray.AddMenuItem("Settings", "Other Settings")
	mShowMeters := mSettings.AddSubMenuItemCheckbox("Show Meters", "Show meters on MCU (restart to apply)", config.Config.McuFaders.ShowMeters)
	mSimulateTouch := mSettings.AddSubMenuItemCheckbox("Simulate Touch", "Simulate touch on MCU for surfaces with no touch support (restart to apply)", config.Config.McuFaders.SimulateTouch)
	inputs := mcu.GetMidiInputs()
	inputItems := make([]*systray.MenuItem, len(inputs))
	for i, v := range inputs {
		selected := config.Config.Midi.PortIn == v
		item := mMidiInputs.AddSubMenuItemCheckbox(v, "", selected)
		inputItems[i] = item
		val := v
		go func() {
			for {
				<-item.ClickedCh
				fromUser <- msg.MidiInputSetting{PortName: val}
				for _, v := range inputItems {
					v.Uncheck()
				}
				item.Check()
			}
		}()
	}
	outputs := mcu.GetMidiOutputs()
	outputItems := make([]*systray.MenuItem, len(outputs))
	for i, v := range outputs {
		selected := config.Config.Midi.PortOut == v
		item := mMidiOutputs.AddSubMenuItemCheckbox(v, "", selected)
		outputItems[i] = item
		val := v
		go func() {
			for {
				<-item.ClickedCh
				fromUser <- msg.MidiOutputSetting{PortName: val}
				for _, v := range outputItems {
					v.Uncheck()
				}
				item.Check()
			}
		}()
	}
	systray.AddSeparator()
	mQuitOrig := systray.AddMenuItem("Quit", "Quit obs-mcu")
	go func() {
		for {
			select {
			case <-mQuitOrig.ClickedCh:
				systray.Quit()
			case <-mOpenConfig.ClickedCh:
				open.Run(config.GetConfigFilePath())
			case <-mShowMeters.ClickedCh:
				config.Config.McuFaders.ShowMeters = !config.Config.McuFaders.ShowMeters
				config.SaveConfig()
				if config.Config.McuFaders.ShowMeters {
					mShowMeters.Check()
				} else {
					mShowMeters.Uncheck()
				}
			case <-mSimulateTouch.ClickedCh:
				config.Config.McuFaders.SimulateTouch = !config.Config.McuFaders.SimulateTouch
				config.SaveConfig()
				if config.Config.McuFaders.SimulateTouch {
					mSimulateTouch.Check()
				} else {
					mSimulateTouch.Uncheck()
				}
			case message := <-fromUser:
				switch msg := message.(type) {
				case msg.MidiInputSetting:
					config.Config.Midi.PortIn = msg.PortName
					config.SaveConfig()
				case msg.MidiOutputSetting:
					config.Config.Midi.PortOut = msg.PortName
					config.SaveConfig()
				}
			}
		}
	}()
}

// check if we run headless
func isHeadless() bool {
	_, display := os.LookupEnv("DISPLAY")
	return runtime.GOOS != "windows" && runtime.GOOS != "darwin" && !display
}
*/
