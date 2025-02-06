package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
)

const (
	APP_TITLE string = "Monitor Controller"
)

func main() {

	guiEvent := make(chan interface{}, 100)

	app, window, err := MakeApp()
	if err != nil {
		fyne.LogError("Fehler beim Laden des Icons", err)
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

func MakeApp() (fyne.App, fyne.Window, error) {
	iconPath := "Icon.png" // Stelle sicher, dass der Pfad stimmt
	iconFile, err := os.ReadFile(iconPath)
	if err != nil {
		return nil, nil, err
	}
	iconResource := fyne.NewStaticResource("appIcon", iconFile)

	app := app.NewWithID("com.github.sebastianrau.focusrite-mackie-control")
	app.SetIcon(iconResource)

	w := app.NewWindow(APP_TITLE)

	if desk, ok := app.(desktop.App); ok {
		m := fyne.NewMenu(APP_TITLE,
			fyne.NewMenuItem("Show", func() {
				w.Show()
			}))
		desk.SetSystemTrayMenu(m)
	}

	w.SetCloseIntercept(func() {
		w.Hide()
	})

	w.SetFullScreen(false)

	w.SetMainMenu(fyne.NewMainMenu())
	w.SetIcon(iconResource) // Setzt das Icon für die App
	w.SetMaster()

	w.SetTitle(APP_TITLE)
	w.FixedSize()
	w.Resize(fyne.NewSize(200, 300))
	return app, w, nil
}
