package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sebastianrau/focusrite-mackie-control/internal/config"
	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite"
)

const (
	appID        = "com.github.sebastianrau.focusrite-mackie-control"
	appTitle     = "Monitor Controller"
	infoNoDevice = "No device connected"
	infoNoServer = "No Focusrite Control connection"
)

type Options struct {
	Version   string
	Submit    func(domain.Command)
	Snapshots <-chan domain.Snapshot
	Devices   <-chan []focusrite.DeviceInfo
	Store     *config.Store
	// Apply is called after the settings were saved and puts them into
	// effect. It must not block.
	Apply func()
	// OnStarted runs on the main thread once the app is running.
	OnStarted func()
}

// App is the desktop UI. Create it and call Run on the main goroutine.
type App struct {
	opts     Options
	fyne     fyne.App
	window   fyne.Window
	settings *settingsWindow

	fader    *Fader
	meter    *Meter
	speakers [domain.NumSpeakers]*ToggleButton
	mute     *ToggleButton
	dim      *ToggleButton
	status   *canvas.Text
	icon     *widget.Icon

	tray     *fyne.Menu
	trayMute *fyne.MenuItem
	trayDim  *fyne.MenuItem

	stop chan struct{}
}

func New(opts Options) *App { return newApp(app.NewWithID(appID), opts) }

func newApp(fa fyne.App, opts Options) *App {
	a := &App{opts: opts, fyne: fa, stop: make(chan struct{})}
	a.fyne.SetIcon(logo)
	a.fyne.Settings().SetTheme(darkTheme{theme.DefaultTheme()})

	a.window = a.fyne.NewWindow(appTitle)
	a.window.SetIcon(logo)
	a.window.SetMainMenu(fyne.NewMainMenu())
	a.window.SetMaster()
	a.window.SetFixedSize(true)

	a.settings = newSettingsWindow(a.fyne, opts)

	a.buildMain()
	a.buildTray()
	a.window.SetCloseIntercept(func() {
		a.window.Hide()
		if a.tray != nil {
			a.tray.Refresh()
		}
	})
	a.render(domain.Snapshot{State: opts.Store.Get().State(), Levels: domain.Silence})
	return a
}

func (a *App) buildMain() {
	a.fader = NewFader(func(db int) { a.opts.Submit(domain.SetVolume{DB: db}) })
	a.meter = NewMeter()

	buttons := container.NewVBox()
	for _, id := range domain.Speakers {
		a.speakers[id] = NewToggleButton(id.String(), green, func() { a.opts.Submit(domain.ToggleSpeaker{ID: id}) })
		buttons.Add(a.speakers[id])
	}
	img := canvas.NewImageFromResource(logo)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(100, 100))
	a.mute = NewToggleButton("Mute", red, func() { a.opts.Submit(domain.ToggleMute{}) })
	a.dim = NewToggleButton("Dim", yellow, func() { a.opts.Submit(domain.ToggleDim{}) })
	buttons.Add(img)
	buttons.Add(a.mute)
	buttons.Add(a.dim)

	a.icon = widget.NewIcon(theme.NewDisabledResource(theme.BrokenImageIcon()))
	a.status = canvas.NewText(infoNoDevice, theme.Color(theme.ColorNameDisabled))

	a.window.SetContent(container.NewBorder(
		nil,
		container.NewHBox(a.icon, a.status),
		a.fader,
		a.meter,
		buttons,
	))
	a.window.Resize(fyne.NewSize(280, 300))
}

func (a *App) buildTray() {
	desk, ok := a.fyne.(desktop.App)
	if !ok {
		return
	}
	a.trayMute = fyne.NewMenuItem("Mute", func() { a.opts.Submit(domain.SetMute{On: !a.trayMute.Checked}) })
	a.trayDim = fyne.NewMenuItem("Dim", func() { a.opts.Submit(domain.SetDim{On: !a.trayDim.Checked}) })
	exit := fyne.NewMenuItem("Exit", a.fyne.Quit)
	exit.IsQuit = true
	a.tray = fyne.NewMenu(appTitle,
		fyne.NewMenuItem("Show", a.window.Show),
		fyne.NewMenuItemSeparator(),
		a.trayDim,
		a.trayMute,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Configuration", a.settings.show),
		fyne.NewMenuItemSeparator(),
		exit,
	)
	desk.SetSystemTrayMenu(a.tray)
}

// Run shows the window and blocks until the app quits.
func (a *App) Run() {
	a.fyne.Lifecycle().SetOnStarted(func() {
		if a.opts.OnStarted != nil {
			a.opts.OnStarted()
		}
		a.meter.Start()
		go a.pump()
		a.settings.start()
	})
	a.fyne.Lifecycle().SetOnStopped(func() { close(a.stop) })
	a.window.ShowAndRun()
}

// Quit stops the app; safe to call from any goroutine.
func (a *App) Quit() { fyne.Do(a.fyne.Quit) }

// pump hands snapshots to the main thread. Waiting for each render keeps the
// UI from falling behind; the controller only keeps the latest snapshot.
func (a *App) pump() {
	for {
		select {
		case <-a.stop:
			return
		case s := <-a.opts.Snapshots:
			fyne.DoAndWait(func() { a.render(s) })
		}
	}
}

func (a *App) render(s domain.Snapshot) {
	st := s.State
	active := s.Device.ServerConnected && s.Device.Present

	a.fader.SetDB(st.Master.VolumeDB)
	a.fader.SetDisabled(!active)
	a.meter.SetLevels(s.Levels.LeftDB, s.Levels.RightDB)
	a.meter.SetDisabled(!active)

	for _, id := range domain.Speakers {
		spk := st.Speakers[id]
		b := a.speakers[id]
		b.SetText(spk.Name)
		b.SetEnabled(active)
		b.Set(spk.Selected)
		if spk.Disabled {
			b.Hide()
		} else {
			b.Show()
		}
	}
	a.mute.SetEnabled(active)
	a.mute.Set(st.Master.Mute)
	a.dim.SetEnabled(active)
	a.dim.Set(st.Master.Dim)

	switch {
	case !s.Device.ServerConnected:
		a.setStatus(infoNoServer, theme.Color(theme.ColorNameDisabled),
			theme.NewColoredResource(theme.BrokenImageIcon(), theme.ColorNameForegroundOnWarning))
	case !s.Device.Present:
		a.setStatus(infoNoDevice, theme.Color(theme.ColorNameDisabled), theme.NewDisabledResource(theme.WarningIcon()))
	default:
		a.setStatus(s.Device.Model, theme.Color(theme.ColorNamePrimary), theme.NewPrimaryThemedResource(theme.VolumeUpIcon()))
	}

	if a.tray != nil && (a.trayMute.Checked != st.Master.Mute || a.trayDim.Checked != st.Master.Dim) {
		a.trayMute.Checked, a.trayDim.Checked = st.Master.Mute, st.Master.Dim
		a.tray.Refresh()
	}
	a.settings.stateChanged(s)
}

func (a *App) setStatus(text string, c color.Color, icon fyne.Resource) {
	if a.status.Text == text {
		return
	}
	a.status.Text, a.status.Color = text, c
	a.status.Refresh()
	a.icon.SetResource(icon)
}
