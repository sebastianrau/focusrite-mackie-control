package gui

import (
	"image/color"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

var log *logger.CustomLogger = logger.WithPackage("gui-main")

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

	controllerChannel chan interface{}

	masterVolumeBuffer int
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

func NewAppWindow(
	myApp fyne.App,
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
		masterValueChanged: make(chan AudioLevelChanged, 100),
		buttonPressed:      make(chan ButtonEvent, 100),
		buttons:            make(map[ButtonID]*ToggleButton),
	}

	mainGui.fader = NewAudioFaderMeter(-127, 0, -10, false, mainGui.masterValueChanged)
	mainGui.fader.SetLevel(-20)

	mainGui.levelMeter = NewAudioMeterBar(0, true)
	mainGui.levelMeter.SetGradient(colorGradient)

	mainGui.buttonContainer = container.NewVBox()

	// Action Buttons
	for _, b := range btnDefinition {
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
			mainGui.buttons[b.ID] = btn
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
	return mainGui, content
}

func (g *MainGui) run() {
	for {
		select {
		case v := <-g.masterValueChanged:
			volume := int(v.Value)
			if g.masterVolumeBuffer != volume {
				g.masterVolumeBuffer = volume
				if g.controllerChannel != nil {
					g.controllerChannel <- monitorcontroller.RcSetVolume(v.Value)
					log.Debugf("Send new Volume: %d", int(v.Value))
				}
			}

		case v := <-g.buttonPressed:
			switch v.Button.ID {
			case SpeakerA,
				SpeakerB,
				SpeakerC,
				SpeakerD,
				Sub:
				g.controllerChannel <- monitorcontroller.RcSpeakerSelect{Id: monitorcontroller.SpeakerID(v.Button.ID), State: !v.Button.state}
			case Mute:
				g.controllerChannel <- monitorcontroller.RcSetMute(!v.Button.state)
			case Dim:
				g.controllerChannel <- monitorcontroller.RcSetDim(!v.Button.state)
			}
		}
	}
}

func (g *MainGui) SetLevelStereo(levelL, levelR float64) {
	g.levelMeter.SetValueStereo(levelL, levelR)
}

func (g *MainGui) SetFader(level float64) {
	g.fader.SetLevel(level)
}

func (g *MainGui) SetButtonlabel(id ButtonID, label string) {

	b, ok := g.buttons[id]
	if !ok {
		return
	}
	b.SetLabel(label)
}
func (g *MainGui) SetButton(id ButtonID, state bool) {
	log.Debugf("Setting Button: %d to %t", id, state)
	b, ok := g.buttons[id]
	if !ok {
		log.Errorf("Button not found %d", id)
		return
	}
	b.Set(state)
}

func (g *MainGui) SetButtonDisabled(id ButtonID, state bool) {
	log.Debugf("Setting Button: %d to %t", id, state)
	b, ok := g.buttons[id]
	if !ok {
		log.Errorf("Button not found %d", id)
		return
	}
	b.SetDisable(state)
}

func (g *MainGui) SetControlChannel(controllerChannel chan interface{}) {
	g.controllerChannel = controllerChannel
}

func (g *MainGui) HandleDeviceArrival(dev *focusritexml.Device) { /* TODO ignore for now */ }
func (g *MainGui) HandleDeviceRemoval()                         { /* TODO ignore for now */ }

// sNew Dim State
func (g *MainGui) HandleDim(state bool) {
	g.SetButton(Dim, state)
}

// New Mute State
func (g *MainGui) HandleMute(state bool) {
	g.SetButton(Mute, state)
}

// Volume -127 .. 0 dB
func (g *MainGui) HandleVolume(volume int) {
	g.SetFader(float64(volume))
}

// Meter Value in DB
func (g *MainGui) HandleMeter(left, right int) {
	g.SetLevelStereo(float64(left), float64(right))
}

// Speaker with given ID new selection State
func (g *MainGui) HandleSpeakerSelect(id monitorcontroller.SpeakerID, state bool) {
	log.Debugf("Speaker %d: %t", id, state)
	g.SetButton(ButtonID(id), state)
}

func (g *MainGui) HandleSpeakerName(id monitorcontroller.SpeakerID, name string) {
	g.SetButtonlabel(ButtonID(id), name)
}

func (g *MainGui) HandleSpeakerUpdate(id monitorcontroller.SpeakerID, spk *monitorcontroller.SpeakerState) {
	log.Debugf("Speaker Update (%d) %s : sel: %t", id, spk.Name, spk.Selected)

	g.SetButton(ButtonID(id), spk.Selected)
	g.SetButtonlabel(ButtonID(id), spk.Name)
	g.SetButtonDisabled(ButtonID(id), spk.Disabled)
}
func (g *MainGui) HandleMasterUpdate(master *monitorcontroller.MasterState) {
	g.SetFader(float64(master.VolumeDB))
	g.SetLevelStereo(float64(master.LevelLeft), float64(master.LevelRight))

	g.SetButton(Mute, master.Mute)
	g.SetButton(Dim, master.Dim)
}
