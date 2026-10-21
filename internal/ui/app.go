package ui

import (
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
	appID    = "com.github.sebastianrau.focusrite-mackie-control"
	appTitle = "Monitor Controller"
)

type Options struct {
	Version   string
	Submit    func(domain.Command)
	Snapshots <-chan domain.Snapshot
	Devices   <-chan []focusrite.DeviceInfo
	Store     *config.Store
	// Restart is called when saved settings need a restart to apply.
	Restart func()
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
	speakers [domain.NumSpeakers]*widget.Button
	mute     *widget.Button
	dim      *widget.Button
	status   *widget.Label
	icon     *widget.Icon

	tray     *fyne.Menu
	trayMute *fyne.MenuItem
	trayDim  *fyne.MenuItem

	snap domain.Snapshot
	stop chan struct{}
}

func New(opts Options) *App { return newApp(app.NewWithID(appID), opts) }

func newApp(fa fyne.App, opts Options) *App {
	a := &App{opts: opts, fyne: fa, stop: make(chan struct{})}
	a.fyne.SetIcon(logo)
	a.fyne.Settings().SetTheme(darkTheme{theme.DefaultTheme()})

	a.window = a.fyne.NewWindow(appTitle)
	a.window.SetIcon(logo)
	a.window.SetMaster()
	a.window.SetFixedSize(true)
	a.window.SetCloseIntercept(a.window.Hide)

	a.settings = newSettingsWindow(a.fyne, opts)

	a.buildMain()
	a.buildTray()
	a.render(domain.Snapshot{State: opts.Store.Get().State(), Levels: domain.Silence})
	return a
}

func (a *App) buildMain() {
	a.fader = NewFader(func(db int) { a.opts.Submit(domain.SetVolume{DB: db}) })
	a.meter = NewMeter()

	buttons := container.NewVBox()
	for _, id := range domain.Speakers {
		a.speakers[id] = widget.NewButton(id.String(), func() { a.opts.Submit(domain.ToggleSpeaker{ID: id}) })
		buttons.Add(a.speakers[id])
	}
	img := canvas.NewImageFromResource(logo)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(80, 80))
	a.mute = widget.NewButton("Mute", func() { a.opts.Submit(domain.ToggleMute{}) })
	a.dim = widget.NewButton("Dim", func() { a.opts.Submit(domain.ToggleDim{}) })
	buttons.Add(img)
	buttons.Add(a.mute)
	buttons.Add(a.dim)

	a.icon = widget.NewIcon(theme.WarningIcon())
	a.status = widget.NewLabel("")
	a.status.Truncation = fyne.TextTruncateEllipsis

	a.window.SetContent(container.NewBorder(
		nil,
		container.NewBorder(nil, nil, a.icon, nil, a.status),
		a.fader,
		a.meter,
		buttons,
	))
	a.window.Resize(fyne.NewSize(300, 380))
}

func (a *App) buildTray() {
	desk, ok := a.fyne.(desktop.App)
	if !ok {
		return
	}
	a.trayMute = fyne.NewMenuItem("Mute", func() { a.opts.Submit(domain.ToggleMute{}) })
	a.trayDim = fyne.NewMenuItem("Dim", func() { a.opts.Submit(domain.ToggleDim{}) })
	quit := fyne.NewMenuItem("Quit", a.fyne.Quit)
	quit.IsQuit = true
	a.tray = fyne.NewMenu(appTitle,
		fyne.NewMenuItem("Show", a.window.Show),
		fyne.NewMenuItemSeparator(),
		a.trayMute,
		a.trayDim,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Settings…", a.settings.show),
		fyne.NewMenuItemSeparator(),
		quit,
	)
	desk.SetSystemTrayMenu(a.tray)
	desk.SetSystemTrayIcon(logo)
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
	a.snap = s
	st := s.State
	active := s.Device.Present

	a.fader.SetDB(st.Master.VolumeDB)
	a.fader.SetDisabled(!active)
	a.meter.SetLevels(s.Levels.LeftDB, s.Levels.RightDB)
	a.meter.SetDisabled(!active)

	for _, id := range domain.Speakers {
		spk := st.Speakers[id]
		b := a.speakers[id]
		setButton(b, spk.Name, spk.Selected, widget.SuccessImportance, active)
		if spk.Disabled {
			b.Hide()
		} else {
			b.Show()
		}
	}
	setButton(a.mute, "Mute", st.Master.Mute, widget.DangerImportance, active)
	setButton(a.dim, "Dim", st.Master.Dim, widget.WarningImportance, active)

	switch {
	case !s.Device.ServerConnected:
		a.setStatus("Focusrite Control not running", theme.NewWarningThemedResource(theme.WarningIcon()))
	case !active:
		a.setStatus("No interface connected", theme.NewWarningThemedResource(theme.WarningIcon()))
	default:
		text := s.Device.Model
		if s.Device.SampleRate != "" {
			text += " · " + s.Device.SampleRate
		}
		a.setStatus(text, theme.NewSuccessThemedResource(theme.VolumeUpIcon()))
	}

	if a.tray != nil && (a.trayMute.Checked != st.Master.Mute || a.trayDim.Checked != st.Master.Dim) {
		a.trayMute.Checked, a.trayDim.Checked = st.Master.Mute, st.Master.Dim
		a.tray.Refresh()
	}
	a.settings.stateChanged(s)
}

func (a *App) setStatus(text string, icon fyne.Resource) {
	if a.status.Text != text {
		a.status.SetText(text)
		a.icon.SetResource(icon)
	}
}

func setButton(b *widget.Button, label string, on bool, onImportance widget.Importance, enabled bool) {
	imp := widget.MediumImportance
	if on {
		imp = onImportance
	}
	changed := b.Text != label || b.Importance != imp
	b.Text, b.Importance = label, imp
	if enabled {
		b.Enable()
	} else {
		b.Disable()
	}
	if changed {
		b.Refresh()
	}
}
