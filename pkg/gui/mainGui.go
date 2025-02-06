package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	SpeakerA ButtonID = iota
	SpeakerB
	SpeakerC
	SpeakerD
	Sub
	Spacer
	Mute
	Dim

	LEN
)

type buttonConfig struct {
	ID       ButtonID
	Name     string
	Color    color.RGBA
	Disabled bool
}

var (
	GREEN  = color.RGBA{0, 255, 0, 255}
	YELLOW = color.RGBA{255, 255, 0, 255} // Yellow
	RED    = color.RGBA{255, 0, 0, 255}   // Red
	BLACK  = color.RGBA{0, 0, 0, 0}

	btnDefinition = []buttonConfig{
		{ID: SpeakerA, Name: "Speaker A", Color: GREEN},
		{ID: SpeakerB, Name: "Speaker B", Color: GREEN},
		{ID: SpeakerC, Name: "Speaker B", Color: GREEN},
		{ID: SpeakerD, Name: "Speaker D", Color: GREEN},
		{ID: Sub, Name: "Sub", Color: GREEN},
		{ID: Spacer, Name: "", Color: BLACK, Disabled: true},
		{ID: Mute, Name: "Mute", Color: RED},
		{ID: Dim, Name: "Dim", Color: YELLOW},
	}
)

type MainGui struct {
	fader           *AudioLevelMeter
	buttonContainer *fyne.Container
	buttons         map[ButtonID]*ToggleButton

	masterValueChanged chan AudioLevelChanged
	buttonPressed      chan ButtonEvent
	guiEvents          chan interface{}
}

func NewAppWindow(
	myApp fyne.App,
	guiEvents chan interface{},
	minLevel float64,
	maxLevel float64,
) (*MainGui, *fyne.Container) {

	colorGradient := NewGradient([]ColorValuePair{
		{Value: -127, Color: color.RGBA{0, 50, 0, 255}},   // Dark green
		{Value: -5, Color: color.RGBA{255, 255, 0, 255}},  // Yellow
		{Value: -15, Color: color.RGBA{255, 255, 0, 255}}, // Light Green
		{Value: 0, Color: color.RGBA{255, 0, 0, 255}},     // Red
	})

	mainGui := &MainGui{
		guiEvents:          guiEvents,
		masterValueChanged: make(chan AudioLevelChanged, 100),
		buttonPressed:      make(chan ButtonEvent, 100),
	}

	mainGui.fader = NewAudioFaderMeter(minLevel, maxLevel, minLevel, 0, "Master", "dB", mainGui.masterValueChanged)
	mainGui.fader.SetGradient(colorGradient)

	mainGui.buttonContainer = container.NewVBox()

	// Action Buttons
	for _, b := range btnDefinition {
		if !b.Disabled {
			btn := NewToggleButton(
				b.ID,
				b.Name,
				b.Color,
				mainGui.buttonPressed,
			)
			mainGui.buttonContainer.Add(btn)
		} else {
			btn := widget.NewButton("", nil)
			btn.Importance = widget.LowImportance
			btn.Disable()
			mainGui.buttonContainer.Add(btn)
		}
	}

	// Layouts
	content := container.NewGridWithColumns(2,
		mainGui.fader,
		mainGui.buttonContainer,
	)
	go mainGui.run()

	return mainGui, content
}

func (g *MainGui) run() {
	for {
		select {
		case v := <-g.masterValueChanged:
			g.guiEvents <- v
		case v := <-g.buttonPressed:
			g.guiEvents <- v
		}
	}
}

func (g *MainGui) SetLevel(level float64) {
	g.fader.SetLevel(level)
}

func (g *MainGui) SetButtonlabel(id ButtonID, label string) {
	b, ok := g.buttons[id]
	if !ok {
		return
	}
	b.Label = label
}
