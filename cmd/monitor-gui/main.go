package main

import (
	"math/rand/v2"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
)

const ()

func main() {

	app, window, err := gui.MakeApp()
	if err != nil {
		fyne.LogError("Loading App error: ", err)
		os.Exit(-1)
	}

	mainGui, content := gui.NewAppWindow(app, -127, 0)

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
