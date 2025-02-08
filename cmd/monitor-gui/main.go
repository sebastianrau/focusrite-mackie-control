package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
)

const ()

func main() {

	guiEvent := make(chan interface{}, 100)

	app, window, err := gui.MakeApp()
	if err != nil {
		fyne.LogError("Loading App error: ", err)
		os.Exit(-1)
	}

	mainGui, content := gui.NewAppWindow(app, guiEvent, -127, 0)

	go func() {
		for {
			select {
			case v := <-guiEvent:

				switch ev := v.(type) {
				case gui.ButtonEvent:
					//toggle Button for showcase

					newValue := !ev.Button.State
					ev.Button.Set(newValue)

					switch ev.Button.ID {
					case gui.SpeakerA,
						gui.SpeakerB,
						gui.SpeakerC,
						gui.SpeakerD,
						gui.Sub:
						fmt.Printf("Speaker %s (%d) Changed to: %t\n", ev.Label, int(ev.Button.ID), newValue)
					case gui.Dim:
						fmt.Printf("Dim: %t\n", newValue)
					case gui.Mute:
						fmt.Printf("Mute: %t\n", newValue)
					default:
						fmt.Printf("Unknown Button (%d) was pressed: %t\n", ev.Button.ID, newValue)
					}

				case gui.AudioLevelChanged:
					// HACK mainGui.SetLevel(ev.Value)
					fmt.Printf("Value Changed on Slider: %.1f\n", ev.Value)
				}
			}
		}
	}()

	// TODO Remove Audio value Simulation

	go func() {
		for {
			time.Sleep(time.Second / 5)
			v := rand.Float64() * -127
			mainGui.SetLevel(v)
		}
	}()

	window.SetContent(content)
	window.ShowAndRun()
}
