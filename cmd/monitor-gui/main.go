package main

import (
	"fmt"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
)

func main() {

	masterValueChanged := make(chan gui.AudioLevelChanged, 100)
	buttonPressed := make(chan gui.ButtonEvent, 100)

	myApp := app.New()

	myWindow := myApp.NewWindow("Audio Monitor Controller")
	myWindow.FixedSize()
	myWindow.Resize(fyne.NewSize(200, 300))

	mainGui, content := gui.NewAppWindow(myApp, masterValueChanged, buttonPressed, -127, 0)

	go func() {
		for {
			select {
			case v := <-masterValueChanged:
				fmt.Printf("Value Changed on Slider: %.1f\n", v.Value)
			case b := <-buttonPressed:
				fmt.Printf("Button Pressed %s\n", b.Label)
				b.Button.Set(!b.Button.State)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second / 10)
			v := rand.Float64() * -127
			mainGui.SetLevel(v)
		}
	}()

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}
