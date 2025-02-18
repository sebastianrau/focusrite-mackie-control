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

	mainGui, err := gui.NewAppWindow(nil, -127, 0)
	if err != nil {
		fyne.LogError("Loading App error: ", err)
		os.Exit(-1)
	}
	go func() {
		for {
			time.Sleep(time.Second / 5)
			l := rand.Float64() * -127
			r := rand.Float64() * -127
			mainGui.SetLevelStereo(l, r)
		}
	}()

	mainGui.ShowAndRun()
}
