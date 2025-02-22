package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	guiconfig "github.com/sebastianrau/focusrite-mackie-control/pkg/gui-config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

var log *logger.CustomLogger = logger.WithPackage("gui-main")

const APP_TITLE string = "Monitor Controller"
const APP_TITLE_CFG string = "Monitor Controller Configuration"

const INFO_NO_DEVICE string = "No device connected"
const INFO_NO_CONNECTION string = "No Focusrite Control connection"

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
	//Color Set
	DARK_GREEN = color.RGBA{0, 102, 0, 225}
	GREEN      = color.RGBA{51, 255, 51, 255}  // Green
	YELLOW     = color.RGBA{255, 255, 51, 255} // Yellow
	ORANGE     = color.RGBA{255, 153, 51, 255} // Orange
	RED        = color.RGBA{255, 51, 0, 255}   // Red

	GREY  = color.RGBA{128, 128, 128, 255}
	BLACK = color.RGBA{0, 0, 0, 0}
)

var (
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

	infoLabel *canvas.Text
	infoIcon  *widget.Icon

	menuShow       *fyne.MenuItem
	menuMute       *fyne.MenuItem
	menuDim        *fyne.MenuItem
	menuConfig     *fyne.MenuItem
	menuSystemTray *fyne.Menu

	masterValueChanged chan AudioLevelChanged
	buttonPressed      chan ButtonEvent

	controllerChannel  chan interface{}
	masterVolumeBuffer int

	app          fyne.App
	window       fyne.Window
	windowConfig fyne.Window
}

func NewAppWindow(cfg *config.Config, closeFunction func()) (*MainGui, error) {

	colorGradient := NewGradient([]ColorValuePair{
		{Value: -127, Color: DARK_GREEN}, // Dark green
		{Value: -15, Color: GREEN},       // Light Green
		{Value: -6, Color: YELLOW},       // Yellow
		{Value: 0, Color: ORANGE},        // Red
	})

	mainGui := &MainGui{
		masterValueChanged: make(chan AudioLevelChanged, 100),
		buttonPressed:      make(chan ButtonEvent, 100),
		buttons:            make(map[ButtonID]*ToggleButton),
	}

	iconResource := resourceIconPng

	//App
	mainGui.app = app.NewWithID("com.github.sebastianrau.focusrite-mackie-control")
	mainGui.app.SetIcon(iconResource)

	//Window
	mainGui.window = mainGui.app.NewWindow(APP_TITLE)
	mainGui.window.SetFullScreen(false)
	mainGui.window.SetMainMenu(fyne.NewMainMenu())
	mainGui.window.SetIcon(iconResource) // Setzt das Icon für die App
	mainGui.window.SetMaster()
	mainGui.window.SetTitle(APP_TITLE)
	mainGui.window.SetFixedSize(true)
	mainGui.window.Resize(fyne.NewSize(280, 300))

	mainGui.windowConfig = mainGui.app.NewWindow(APP_TITLE_CFG)
	mainGui.windowConfig.Resize(fyne.NewSize(450, 600))

	cfgGui := guiconfig.NewConfigApp(mainGui.app, cfg, func() {
		mainGui.windowConfig.Hide()
	})
	mainGui.windowConfig.SetContent(cfgGui.Content)

	mainGui.fader = NewAudioFaderMeter(-127, 0, -10, false, mainGui.masterValueChanged)
	mainGui.fader.SetLevel(-20)

	mainGui.levelMeter = NewAudioMeterBar(true)
	mainGui.levelMeter.SetGradient(colorGradient)

	mainGui.buttonContainer = container.NewVBox()

	mainGui.infoIcon = widget.NewIcon(theme.NewDisabledResource(theme.BrokenImageIcon()))
	mainGui.infoLabel = canvas.NewText(INFO_NO_DEVICE, theme.Color(theme.ColorNameDisabled))

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

	// Context menu
	mainGui.menuMute = fyne.NewMenuItem("Mute", func() {
		mainGui.controllerChannel <- monitorcontroller.RcSetMute(!mainGui.menuMute.Checked)

	})

	mainGui.menuDim = fyne.NewMenuItem("Dim", func() {
		mainGui.controllerChannel <- monitorcontroller.RcSetDim(!mainGui.menuDim.Checked)
	})

	mainGui.menuShow = fyne.NewMenuItem("Show", func() {
		mainGui.window.Show()
		mainGui.menuShow.Disabled = true
		mainGui.menuSystemTray.Refresh()
	})

	//Gui shown initally
	mainGui.menuShow.Disabled = true

	mainGui.menuConfig = fyne.NewMenuItem("Configuration", func() {
		mainGui.windowConfig.Show()
	})

	if desk, ok := mainGui.app.(desktop.App); ok {

		exit := fyne.NewMenuItem("Exit", func() {
			if closeFunction != nil {
				closeFunction()
			}
			mainGui.app.Quit()
		})
		exit.IsQuit = true

		mainGui.menuSystemTray = fyne.NewMenu(APP_TITLE,
			mainGui.menuShow,
			fyne.NewMenuItemSeparator(),
			mainGui.menuDim,
			mainGui.menuMute,
			fyne.NewMenuItemSeparator(),
			mainGui.menuConfig,
			fyne.NewMenuItemSeparator(),
			exit,
		)

		desk.SetSystemTrayMenu(mainGui.menuSystemTray)
	}

	mainGui.window.SetCloseIntercept(func() {
		mainGui.window.Hide()
		mainGui.menuShow.Disabled = false
		mainGui.menuSystemTray.Refresh()
	})

	// Layouts
	content := container.NewBorder(
		nil, //top
		container.NewHBox(mainGui.infoIcon, mainGui.infoLabel), // bot
		mainGui.fader,           // left
		mainGui.levelMeter,      // right
		mainGui.buttonContainer, // center
	)
	mainGui.window.SetContent(content)

	// Worker
	go mainGui.run()

	return mainGui, nil
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

func (g *MainGui) UpdateStatusText(dev *monitorcontroller.DeviceInfo) {
	if dev.ConnectionState {
		if dev.DeviceId == 0 {
			g.infoLabel.Text = INFO_NO_DEVICE
			g.infoIcon.SetResource(theme.NewDisabledResource(theme.WarningIcon()))
		} else {
			g.infoLabel.Text = dev.Model
			g.infoIcon.SetResource(theme.NewDisabledResource(theme.VolumeUpIcon()))
		}
	} else {
		g.infoLabel.Text = INFO_NO_CONNECTION
		g.infoIcon.SetResource(theme.NewDisabledResource(theme.BrokenImageIcon()))
	}

	g.infoLabel.Refresh()
}

func (g *MainGui) SetControlChannel(controllerChannel chan interface{}) {
	g.controllerChannel = controllerChannel
}

func (g *MainGui) HandleDeviceUpdate(dev *monitorcontroller.DeviceInfo) {
	g.UpdateStatusText(dev)
}

// sNew Dim State
func (g *MainGui) HandleDim(state bool) {
	g.SetButton(Dim, state)
	g.menuDim.Checked = state
	g.menuSystemTray.Refresh()
}

// New Mute State
func (g *MainGui) HandleMute(state bool) {
	g.SetButton(Mute, state)
	g.menuMute.Checked = state
	g.menuSystemTray.Refresh()
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

	g.menuMute.Checked = master.Mute
	g.menuDim.Checked = master.Dim
	g.menuSystemTray.Refresh()
}

func (g *MainGui) ShowAndRun() {
	g.window.ShowAndRun()
}

func (g *MainGui) Lifecycle() fyne.Lifecycle {
	return g.app.Lifecycle()
}
