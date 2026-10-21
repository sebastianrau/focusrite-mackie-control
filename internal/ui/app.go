package ui

import (
	"fmt"
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

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

	volume   *canvas.Text
	volBox   *fyne.Container // re-laid out when the number's width changes
	volNote  *canvas.Text
	fader    *Fader
	meter    *Meter
	speakers [domain.NumSpeakers]*blockButton
	mute     *blockButton
	dim      *blockButton
	dot      *canvas.Rectangle
	status   *canvas.Text
	statBox  *fyne.Container

	tray      *fyne.Menu
	trayMute  *fyne.MenuItem
	trayDim   *fyne.MenuItem
	trayState fyne.Resource

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
		a.meter.Pause()
		if a.tray != nil {
			a.tray.Refresh()
		}
	})
	a.render(domain.Snapshot{State: opts.Store.Get().State(), Levels: domain.Silence})
	return a
}

func (a *App) buildMain() {
	a.mute = newBlockButton("MUTE", fyne.TextAlignCenter, func() { a.opts.Submit(domain.ToggleMute{}) })
	a.dim = newBlockButton("DIM", fyne.TextAlignCenter, func() { a.opts.Submit(domain.ToggleDim{}) })
	a.mute.label.TextSize, a.dim.label.TextSize = 13, 13
	switches := container.NewGridWithColumns(2, a.mute, a.dim)

	right := container.NewBorder(nil, switches, nil, nil, a.speakerList())
	a.window.SetContent(container.NewBorder(nil, a.statusBar(), nil, nil,
		container.New(layout.NewCustomPaddedLayout(4, 10, 10, 10),
			container.NewBorder(nil, nil, a.channelStrip(), nil, right))))
	a.window.Resize(fyne.NewSize(320, 440))
}

// channelStrip is the left column: volume, fader and meter.
func (a *App) channelStrip() fyne.CanvasObject {
	a.volume = canvas.NewText("", colText)
	a.volume.TextSize = 30
	a.volume.TextStyle.Bold = true
	unit := canvas.NewText("dB", colTextDim)
	unit.TextSize = 13
	a.volNote = canvas.NewText("", colYellow)
	a.volNote.TextSize = 11
	a.volNote.Alignment = fyne.TextAlignCenter
	a.volBox = container.NewCenter(container.NewHBox(a.volume, container.NewVBox(layout.NewSpacer(), unit)))
	header := container.NewVBox(a.volBox, a.volNote)

	a.fader = NewFader(func(db int) { a.opts.Submit(domain.SetVolume{DB: db}) })
	a.meter = NewMeter()
	center := container.NewHBox(layout.NewSpacer(), a.fader, newMeterScale(), a.meter, layout.NewSpacer())

	bg := canvas.NewRectangle(colPanel)
	bg.CornerRadius = 14
	bg.SetMinSize(fyne.NewSize(128, 0))
	return container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(12, 10, 10, 10),
		container.NewBorder(header, nil, nil, nil, center)))
}

// speakerList is the right column above Mute and Dim: one row per speaker,
// the sub apart.
func (a *App) speakerList() fyne.CanvasObject {
	list := container.NewVBox(sectionHeading("SPEAKERS"))
	for _, id := range domain.Speakers {
		row := newBlockButton(id.String(), fyne.TextAlignLeading, func() { a.opts.Submit(domain.ToggleSpeaker{ID: id}) })
		key := string(rune('A' + int(id)))
		if id.IsSub() {
			key = "S"
			list.Add(sectionHeading("SUBWOOFER"))
		}
		a.speakers[id] = row.withKey(key)
		list.Add(row)
	}
	return list
}

func sectionHeading(text string) fyne.CanvasObject {
	t := canvas.NewText(text, colTextDim)
	t.TextSize = 11
	t.TextStyle.Bold = true
	return container.New(layout.NewCustomPaddedLayout(6, 0, 4, 0), t)
}

func (a *App) statusBar() fyne.CanvasObject {
	a.dot = canvas.NewRectangle(colTextDisabled)
	a.dot.CornerRadius = 3.5
	a.dot.SetMinSize(fyne.NewSize(7, 7))
	a.status = canvas.NewText("", colTextIdle)
	a.status.TextSize = 12
	line := canvas.NewRectangle(colLine)
	line.SetMinSize(fyne.NewSize(0, 1))
	a.statBox = container.NewHBox(container.NewCenter(a.dot), a.status)
	return container.NewBorder(line, nil, nil, nil,
		container.New(layout.NewCustomPaddedLayout(6, 6, 14, 14), a.statBox))
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
		fyne.NewMenuItem("Show", a.show),
		fyne.NewMenuItemSeparator(),
		a.trayDim,
		a.trayMute,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Configuration", a.settings.show),
		fyne.NewMenuItemSeparator(),
		exit,
	)
	desk.SetSystemTrayMenu(a.tray)
	a.setTrayIcon(trayNormal)
}

func (a *App) show() {
	a.window.Show()
	a.meter.Resume()
}

func (a *App) setTrayIcon(icon fyne.Resource) {
	desk, ok := a.fyne.(desktop.App)
	if !ok || a.trayState == icon {
		return
	}
	a.trayState = icon
	desk.SetSystemTrayIcon(icon)
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

	if setText(a.volume, strconv.Itoa(st.Master.VolumeDB), pick(active, colText, colTextDisabled)) {
		a.volBox.Refresh()
	}
	switch {
	case st.Master.Mute:
		setText(a.volNote, "muted", colRedText)
	case st.Master.Dim:
		setText(a.volNote, fmt.Sprintf("dimmed −%d dB", st.Master.DimDB), colYellow)
	default:
		setText(a.volNote, "", colYellow)
	}
	a.fader.SetDB(st.Master.VolumeDB)
	a.fader.SetDisabled(!active)
	a.meter.SetLevels(s.Levels.LeftDB, s.Levels.RightDB)
	a.meter.SetDisabled(!active)

	for _, id := range domain.Speakers {
		spk, row := st.Speakers[id], a.speakers[id]
		switch {
		case spk.Disabled:
			row.setText(spk.Name + " · disabled")
			row.style(color.Transparent, nil, colTextDisabled, false)
		case !active:
			row.setText(spk.Name)
			row.style(colPanel, nil, colTextDisabled, spk.Selected)
		case spk.Selected:
			row.setText(spk.Name)
			row.style(colGreenBg, colGreen, color.White, true)
		default:
			row.setText(spk.Name)
			row.style(colPanel, nil, colTextIdle, false)
		}
		row.setEnabled(active && !spk.Disabled)
	}
	styleSwitch(a.mute, st.Master.Mute, active, colRed, color.White)
	styleSwitch(a.dim, st.Master.Dim, active, colYellow, colDark)

	switch {
	case !s.Device.ServerConnected:
		a.setStatus(infoNoServer, colTextDisabled, colTextDim)
	case !s.Device.Present:
		a.setStatus(infoNoDevice, colYellow, colTextDim)
	default:
		a.setStatus(s.Device.Model, colGreen, colTextIdle)
	}

	if a.tray != nil {
		a.setTrayIcon(trayIconFor(st.Master.Mute, st.Master.Dim))
		if a.trayMute.Checked != st.Master.Mute || a.trayDim.Checked != st.Master.Dim {
			a.trayMute.Checked, a.trayDim.Checked = st.Master.Mute, st.Master.Dim
			a.tray.Refresh()
		}
	}
	a.settings.stateChanged(s)
}

func styleSwitch(b *blockButton, on, active bool, fill, text color.Color) {
	switch {
	case !active:
		b.style(colRaised, nil, colTextDisabled, true)
	case on:
		b.style(fill, nil, text, true)
	default:
		b.style(colRaised, nil, colTextIdle, true)
	}
	b.setEnabled(active)
}

func (a *App) setStatus(text string, dot, fg color.Color) {
	if a.dot.FillColor != dot {
		a.dot.FillColor = dot
		a.dot.Refresh()
	}
	if setText(a.status, text, fg) {
		a.statBox.Refresh()
	}
}

// setText updates t and reports whether anything changed. Containers do not
// re-lay out by themselves when a text's width changes.
func setText(t *canvas.Text, s string, c color.Color) bool {
	if t.Text == s && t.Color == c {
		return false
	}
	t.Text, t.Color = s, c
	t.Refresh()
	return true
}

func pick(cond bool, yes, no color.Color) color.Color {
	if cond {
		return yes
	}
	return no
}
