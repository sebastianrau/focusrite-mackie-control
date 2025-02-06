package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type MainGui struct {
	fader           *AudioLevelMeter
	buttonContainer *fyne.Container
	buttons         map[string]*ToggleButton
}

func NewAppWindow(
	myApp fyne.App,
	masterValueChanged chan AudioLevelChanged,
	buttonPressed chan ButtonEvent,
	minLevel float64,
	maxLevel float64,
) (*MainGui, *fyne.Container) {

	mainGui := &MainGui{
		fader:           NewAudioFaderMeter(minLevel, maxLevel, minLevel, 0, "Master", "dB", masterValueChanged),
		buttonContainer: container.NewVBox(),
	}

	// Speaker and Sub Buttons

	speakerButtons := []string{"Speaker A", "Speaker B", "Speaker C", "Speaker D", "Sub"}
	for _, v := range speakerButtons {
		btn := NewToggleButton(
			v,
			color.RGBA{0, 255, 0, 255}, //Green
			buttonPressed,
		)
		//mainGui.buttons[v] = btn
		mainGui.buttonContainer.Add(btn)
	}

	spacerButton := widget.NewButton("", nil)
	spacerButton.Importance = widget.LowImportance
	spacerButton.Disable()
	mainGui.buttonContainer.Add(spacerButton)

	muteBtn := NewToggleButton(
		"Mute",
		color.RGBA{255, 0, 0, 255}, // Red
		buttonPressed,
	)
	mainGui.buttonContainer.Add(muteBtn)

	dimBtn := NewToggleButton(
		"Dim",
		color.RGBA{255, 255, 0, 255}, // Yellow
		buttonPressed,
	)
	mainGui.buttonContainer.Add(dimBtn)

	// Layout
	content := container.NewGridWithColumns(2,
		mainGui.fader,
		mainGui.buttonContainer,
	)
	return mainGui, content
}

func (g *MainGui) SetLevel(level float64) {

	scaledLevel := (level - g.fader.minLevel) / (g.fader.maxLevel - g.fader.minLevel)

	if scaledLevel < 0.0 {
		scaledLevel = 0.0
	} else if scaledLevel > 1.0 {
		scaledLevel = 1.0
	}
	g.fader.SetLevel(scaledLevel)
}

/*
func (g *MainGui) SetButton(button string, state bool) {
	b, ok := g.buttons[button]
	if !ok {
		return
	}
	b.Set(state)
}*/

func (g *MainGui) SetButtonlabel(button string, label string) {
	b, ok := g.buttons[button]
	if !ok {
		return
	}
	b.Label = label
}
