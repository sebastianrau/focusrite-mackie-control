package ui

import (
	"fmt"
	"reflect"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/sebastianrau/gomcu"

	"github.com/sebastianrau/focusrite-mackie-control/internal/config"
	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite"
	"github.com/sebastianrau/focusrite-mackie-control/internal/mcu"
)

const none = "None"

// settingsWindow edits a draft that is applied on Save. Monitor settings are
// controller state and apply at once; the Focusrite and MIDI mappings are
// read at startup and need a restart.
type settingsWindow struct {
	opts Options
	win  fyne.Window

	live    domain.State
	devices []focusrite.DeviceInfo

	draft     config.Config
	monitor   domain.State
	focusrite *fyne.Container
	midi      *fyne.Container
}

func newSettingsWindow(app fyne.App, opts Options) *settingsWindow {
	s := &settingsWindow{opts: opts, win: app.NewWindow(appTitle + " Settings")}
	s.win.Resize(fyne.NewSize(460, 560))
	s.win.SetCloseIntercept(s.win.Hide)
	s.focusrite = container.NewVBox()
	s.midi = container.NewVBox()
	return s
}

// start follows the device list. Called once the app runs.
func (s *settingsWindow) start() {
	go func() {
		for list := range s.opts.Devices {
			fyne.Do(func() {
				s.devices = list
				if s.win.Content() != nil {
					s.buildFocusrite()
				}
			})
		}
	}()
}

func (s *settingsWindow) stateChanged(snap domain.Snapshot) { s.live = snap.State }

func (s *settingsWindow) show() {
	s.draft = s.opts.Store.Get()
	s.monitor = s.live
	s.buildFocusrite()
	s.buildMIDI()

	tabs := container.NewAppTabs(
		container.NewTabItem("Monitor", container.NewVScroll(s.monitorTab())),
		container.NewTabItem("Focusrite", container.NewVScroll(s.focusrite)),
		container.NewTabItem("MIDI", container.NewVScroll(s.midi)),
	)
	save := widget.NewButton("Save", s.save)
	save.Importance = widget.HighImportance
	version := widget.NewLabel("Version " + s.opts.Version)
	version.Importance = widget.LowImportance
	buttons := container.NewHBox(version, layout.NewSpacer(), widget.NewButton("Cancel", s.win.Hide), save)
	s.win.SetContent(container.NewBorder(nil, container.NewPadded(buttons), nil, nil, tabs))
	s.win.Show()
	s.win.RequestFocus()
}

func (s *settingsWindow) monitorTab() fyne.CanvasObject {
	dimLabel := widget.NewLabel("")
	showDim := func() { dimLabel.SetText(fmt.Sprintf("%d dB", s.monitor.Master.DimDB)) }
	dim := widget.NewSlider(0, domain.MaxDimDB)
	dim.Step = 1
	dim.Value = float64(s.monitor.Master.DimDB)
	dim.OnChanged = func(v float64) {
		s.monitor.Master.DimDB = int(v)
		showDim()
	}
	showDim()

	form := widget.NewForm(widget.NewFormItem("Dim", container.NewBorder(nil, nil, nil, dimLabel, dim)))
	for _, id := range domain.Speakers {
		spk := &s.monitor.Speakers[id]
		enabled := widget.NewCheck("Enabled", func(on bool) { spk.Disabled = !on })
		enabled.Checked = !spk.Disabled
		row := container.NewHBox(enabled)
		if !id.IsSub() {
			exclusive := widget.NewCheck("Exclusive", func(on bool) { spk.Exclusive = on })
			exclusive.Checked = spk.Exclusive
			row.Add(exclusive)
		}
		form.Append(spk.Name, row)
	}
	hint := widget.NewLabel("Selecting an exclusive speaker switches the other speakers off. The sub is switched separately.")
	hint.Wrapping = fyne.TextWrapWord
	hint.Importance = widget.LowImportance
	return container.NewVBox(form, hint)
}

func (s *settingsWindow) buildFocusrite() {
	fc := s.draft.FocusriteConfig()
	var current *focusrite.DeviceInfo
	options := []string{}
	label := func(d focusrite.DeviceInfo) string { return fmt.Sprintf("%s (%s)", d.Model, d.SerialNumber) }
	for i := range s.devices {
		options = append(options, label(s.devices[i]))
		if s.devices[i].SerialNumber == fc.SerialNumber {
			current = &s.devices[i]
		}
	}
	selected := ""
	switch {
	case current != nil:
		selected = label(*current)
	case fc.SerialNumber != "":
		selected = fc.SerialNumber + " (not connected)"
		options = append(options, selected)
	}

	device := widget.NewSelect(options, nil)
	device.PlaceHolder = "First interface found"
	device.Selected = selected
	device.OnChanged = func(opt string) {
		for _, d := range s.devices {
			if label(d) == opt && d.SerialNumber != fc.SerialNumber {
				s.draft.SetFocusrite(focusrite.AutoConfig(d))
			}
		}
		s.buildFocusrite()
	}

	form := widget.NewForm(widget.NewFormItem("Interface", device))
	if current != nil {
		controls := "not available"
		if current.MuteID != 0 && current.DimID != 0 {
			controls = "linked"
		}
		form.Append("Monitor mute/dim", widget.NewLabel(controls))
	}
	for _, id := range domain.Speakers {
		form.Append(id.String(), s.outputSelect(current, fc, id))
	}

	var hint string
	switch {
	case len(s.devices) == 0:
		hint = "Start Focusrite Control and connect the interface to choose outputs."
	case current == nil:
		hint = "Choose an interface to assign its outputs."
	}
	s.focusrite.Objects = []fyne.CanvasObject{form}
	if hint != "" {
		h := widget.NewLabel(hint)
		h.Wrapping = fyne.TextWrapWord
		h.Importance = widget.LowImportance
		s.focusrite.Add(h)
	}
	s.focusrite.Refresh()
}

func (s *settingsWindow) outputSelect(dev *focusrite.DeviceInfo, fc focusrite.Config, id domain.SpeakerID) fyne.CanvasObject {
	assigned := fc.Outputs[id]
	if dev == nil {
		text := none
		if assigned != (focusrite.Output{}) {
			text = "assigned"
		}
		return widget.NewLabel(text)
	}
	options := []string{none}
	selected := none
	for _, out := range dev.Outputs {
		options = append(options, out.Name)
		if assigned.MuteID != 0 && out.MuteID == assigned.MuteID {
			selected = out.Name
		}
	}
	sel := widget.NewSelect(options, nil)
	sel.Selected = selected
	sel.OnChanged = func(name string) {
		next := s.draft.FocusriteConfig()
		next.Outputs[id] = focusrite.Output{}
		for _, out := range dev.Outputs {
			if out.Name == name {
				next.Outputs[id] = out.Output
			}
		}
		s.draft.SetFocusrite(next)
	}
	return sel
}

func (s *settingsWindow) buildMIDI() {
	m := &s.draft.MIDI
	port := func(ports []string, cur *string) *widget.Select {
		options := append([]string{none}, ports...)
		selected := none
		if *cur != "" {
			selected = *cur
			if !slices.Contains(ports, *cur) {
				options = append(options, *cur)
			}
		}
		sel := widget.NewSelect(options, func(v string) {
			if v == none {
				v = ""
			}
			*cur = v
		})
		sel.Selected = selected
		return sel
	}
	key := func(cur *config.Key) *widget.Select {
		sel := widget.NewSelect(gomcu.Names, func(v string) {
			if id, ok := gomcu.IDs[v]; ok {
				*cur = config.Key(id)
			}
		})
		if int(*cur) < len(gomcu.Names) {
			sel.Selected = gomcu.Names[*cur]
		}
		return sel
	}

	channel := widget.NewSelect(gomcu.ChannelNames, func(v string) {
		if id, ok := gomcu.ChannelIDs[v]; ok {
			m.FaderChannel = config.Channel(id)
		}
	})
	if int(m.FaderChannel) < len(gomcu.ChannelNames) {
		channel.Selected = gomcu.ChannelNames[m.FaderChannel]
	}
	logFader := widget.NewCheck("Logarithmic fader scale", func(on bool) { m.LogarithmicFade = on })
	logFader.Checked = m.LogarithmicFade

	form := widget.NewForm(
		widget.NewFormItem("Input port", port(mcu.InPorts(), &m.InputPort)),
		widget.NewFormItem("Output port", port(mcu.OutPorts(), &m.OutputPort)),
		widget.NewFormItem("", widget.NewButton("Refresh ports", s.buildMIDI)),
		widget.NewFormItem("Volume fader", channel),
		widget.NewFormItem("", logFader),
		widget.NewFormItem("Mute key", key(&m.MuteKey)),
		widget.NewFormItem("Dim key", key(&m.DimKey)),
	)
	for i := range m.SpeakerKeys {
		form.Append(domain.SpeakerID(i).String()+" key", key(&m.SpeakerKeys[i]))
	}
	s.midi.Objects = []fyne.CanvasObject{form}
	s.midi.Refresh()
}

func (s *settingsWindow) save() {
	for _, id := range domain.Speakers {
		want, have := s.monitor.Speakers[id], s.live.Speakers[id]
		if want.Disabled != have.Disabled || want.Exclusive != have.Exclusive {
			s.opts.Submit(domain.ConfigureSpeaker{ID: id, Disabled: want.Disabled, Exclusive: want.Exclusive})
		}
	}
	if s.monitor.Master.DimDB != s.live.Master.DimDB {
		s.opts.Submit(domain.SetDimDB{DB: s.monitor.Master.DimDB})
	}

	saved := s.opts.Store.Get()
	if reflect.DeepEqual(saved.Focusrite, s.draft.Focusrite) && reflect.DeepEqual(saved.MIDI, s.draft.MIDI) {
		s.win.Hide()
		return
	}
	s.opts.Store.Update(func(c *config.Config) {
		c.Focusrite = s.draft.Focusrite
		c.MIDI = s.draft.MIDI
	})
	if err := s.opts.Store.Save(); err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	dialog.ShowConfirm("Restart required",
		"The Focusrite and MIDI settings apply after a restart. Restart now?",
		func(ok bool) {
			s.win.Hide()
			if ok && s.opts.Restart != nil {
				s.opts.Restart()
			}
		}, s.win)
}
