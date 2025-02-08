package gui

import (
	"fmt"
	"image/color"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
)

const APP_TITLE string = "Monitor Controller"

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
	ID    ButtonID
	Name  string
	Color color.RGBA
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
		{ID: Spacer, Name: "", Color: BLACK},
		{ID: Mute, Name: "Mute", Color: RED},
		{ID: Dim, Name: "Dim", Color: YELLOW},
	}
)

type MainGui struct {
	fader           *AudioFader
	levelMeter      *AudioMeter
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
		{Value: -127, Color: color.RGBA{0, 50, 0, 255}},  // Dark green
		{Value: -15, Color: color.RGBA{0, 180, 0, 255}},  // Light Green
		{Value: -6, Color: color.RGBA{255, 255, 0, 255}}, // Yellow
		{Value: 0, Color: color.RGBA{255, 0, 0, 255}},    // Red
	})

	mainGui := &MainGui{
		guiEvents:          guiEvents,
		masterValueChanged: make(chan AudioLevelChanged, 100),
		buttonPressed:      make(chan ButtonEvent, 100),
	}

	mainGui.fader = NewAudioFaderMeter(-127, 0, -10, false, mainGui.masterValueChanged)
	mainGui.fader.SetLevel(-20)

	mainGui.levelMeter = NewAudioMeterBar(0)
	mainGui.levelMeter.SetGradient(colorGradient)
	mainGui.levelMeter.Decay = false

	mainGui.buttonContainer = container.NewVBox()

	// Action Buttons
	for _, b := range btnDefinition {

		//imgSize := mainGui.buttonContainer.Size().Width

		if b.ID == Spacer {
			img := canvas.NewImageFromFile("logo.png")
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(100, 100))
			mainGui.buttonContainer.Add(img)
		} else {
			btn := NewToggleButton(
				b.ID,
				b.Name,
				b.Color,
				mainGui.buttonPressed,
			)
			mainGui.buttonContainer.Add(btn)
		}

	}

	// Layouts
	content := container.NewBorder(
		nil, // top
		nil, // bot
		mainGui.fader,
		mainGui.levelMeter,
		mainGui.buttonContainer,
	)
	go mainGui.run()

	fmt.Printf("Fader Size w/h: %.0f %.0f ", mainGui.fader.MinSize().Width, mainGui.fader.MinSize().Height)

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
	g.levelMeter.SetValue(level)
}
func (g *MainGui) SetFader(level float64) {
	g.fader.SetLevel(level)
}
func (g *MainGui) SetButtonlabel(id ButtonID, label string) {
	b, ok := g.buttons[id]
	if !ok {
		return
	}
	b.Label = label
}
func (g *MainGui) SetButton(id ButtonID, state bool) {
	b, ok := g.buttons[id]
	if !ok {
		return
	}
	b.State = state
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
	w.SetFixedSize(true)
	w.SetFullScreen(false)
	w.Resize(fyne.NewSize(280, 300))
	return app, w, nil
}
