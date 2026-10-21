package ui

import (
	"fmt"

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

const none = "none"

// settingsWindow edits a draft that "Save" applies at once: monitor settings
// go to the controller, the Focusrite and MIDI mappings to the settings file,
// after which the affected connections are restarted.
type settingsWindow struct {
	opts Options
	win  fyne.Window

	live    domain.State
	devices []focusrite.DeviceInfo

	draft     config.Config
	monitor   domain.State
	focusrite *fyne.Container
	built     bool // the window content exists
}

func newSettingsWindow(app fyne.App, opts Options) *settingsWindow {
	s := &settingsWindow{opts: opts, win: app.NewWindow(appTitle + " Configuration")}
	s.win.Resize(fyne.NewSize(450, 600))
	s.win.SetCloseIntercept(s.win.Hide)
	s.focusrite = container.NewVBox()
	return s
}

// start follows the device list. Called once the app runs.
func (s *settingsWindow) start() {
	go func() {
		for list := range s.opts.Devices {
			fyne.Do(func() {
				s.devices = list
				if s.built {
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
	s.built = true
	s.buildFocusrite()

	sections := widget.NewAccordion(
		widget.NewAccordionItem("Monitor Controller", s.monitorSection()),
		widget.NewAccordionItem("Midi:", s.midiSection()),
		widget.NewAccordionItem("Focusrite:", s.focusrite),
	)
	sections.Open(0)

	save := widget.NewButton("Save", s.save)
	closeBtn := widget.NewButton("Close", func() {
		dialog.ShowConfirm("Close Config", "Your Settings will not be saved. Will you Close?", func(ok bool) {
			if ok {
				s.win.Hide()
			}
		}, s.win)
	})

	s.win.SetContent(container.NewBorder(
		nil,
		container.NewGridWithColumns(3, save, layout.NewSpacer(), closeBtn),
		nil, nil,
		container.NewVScroll(sections),
	))
	s.win.Show()
	s.win.RequestFocus()
}

// newSelect sets the placeholder itself: Fyne has no translation for its
// default placeholder in some languages and logs an error for every select.
func newSelect(options []string, changed func(string)) *widget.Select {
	sel := &widget.Select{Options: options, OnChanged: changed, PlaceHolder: "–"}
	sel.ExtendBaseWidget(sel)
	return sel
}

func heading(text string) *widget.Label {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func (s *settingsWindow) monitorSection() fyne.CanvasObject {
	dimLabel := widget.NewLabel("")
	showDim := func() { dimLabel.SetText(fmt.Sprintf("Dim: %d dB", s.monitor.Master.DimDB)) }
	dim := widget.NewSlider(0, domain.MaxDimDB)
	dim.Step = 5
	dim.Value = float64(s.monitor.Master.DimDB)
	dim.OnChanged = func(v float64) {
		s.monitor.Master.DimDB = int(v)
		showDim()
	}
	showDim()

	form := container.New(layout.NewFormLayout(),
		heading("Master:"), layout.NewSpacer(),
		dimLabel, dim,
		heading("Speaker:"), layout.NewSpacer(),
	)
	for _, id := range domain.Speakers {
		spk := &s.monitor.Speakers[id]
		disabled := widget.NewCheck("Disabled", func(on bool) { spk.Disabled = on })
		disabled.Checked = spk.Disabled
		row := container.NewHBox(disabled)
		if !id.IsSub() {
			exclusive := widget.NewCheck("Exclusive", func(on bool) { spk.Exclusive = on })
			exclusive.Checked = spk.Exclusive
			row.Add(exclusive)
		}
		form.Add(widget.NewLabel(speakerLabel(id)))
		form.Add(row)
	}
	return form
}

func speakerLabel(id domain.SpeakerID) string {
	if id.IsSub() {
		return "Subwoofer:"
	}
	return id.String() + ":"
}

func (s *settingsWindow) midiSection() fyne.CanvasObject {
	m := &s.draft.MIDI
	port := func(ports []string, cur *string) *widget.Select {
		options := append([]string{none}, ports...)
		sel := newSelect(options, func(v string) {
			if v == none {
				v = ""
			}
			*cur = v
		})
		sel.Selected = *cur
		return sel
	}
	key := func(cur *config.Key) *widget.Select {
		sel := newSelect(gomcu.Names, func(v string) {
			if id, ok := gomcu.IDs[v]; ok {
				*cur = config.Key(id)
			}
		})
		if int(*cur) < len(gomcu.Names) {
			sel.Selected = gomcu.Names[*cur]
		}
		return sel
	}
	channel := newSelect(gomcu.ChannelNames, func(v string) {
		if id, ok := gomcu.ChannelIDs[v]; ok {
			m.FaderChannel = config.Channel(id)
		}
	})
	if int(m.FaderChannel) < len(gomcu.ChannelNames) {
		channel.Selected = gomcu.ChannelNames[m.FaderChannel]
	}

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("Input Port:"), port(mcu.InPorts(), &m.InputPort),
		widget.NewLabel("Output Port:"), port(mcu.OutPorts(), &m.OutputPort),
		heading("Master:"), layout.NewSpacer(),
		widget.NewLabel("Fader:"), channel,
		widget.NewLabel("Mute:"), key(&m.MuteKey),
		widget.NewLabel("Dim:"), key(&m.DimKey),
		heading("Speaker:"), layout.NewSpacer(),
	)
	for i := range m.SpeakerKeys {
		form.Add(widget.NewLabel(speakerLabel(domain.SpeakerID(i))))
		form.Add(key(&m.SpeakerKeys[i]))
	}
	return form
}

func (s *settingsWindow) buildFocusrite() {
	fc := s.draft.FocusriteConfig()
	label := func(d focusrite.DeviceInfo) string { return fmt.Sprintf("%s (%s)", d.Model, d.SerialNumber) }
	var current *focusrite.DeviceInfo
	options := []string{}
	for i := range s.devices {
		options = append(options, label(s.devices[i]))
		if s.devices[i].SerialNumber == fc.SerialNumber {
			current = &s.devices[i]
		}
	}

	device := newSelect(options, nil)
	if current != nil {
		device.Selected = label(*current)
	}
	device.OnChanged = func(opt string) {
		for _, d := range s.devices {
			if label(d) == opt && d.SerialNumber != fc.SerialNumber {
				s.draft.SetFocusrite(focusrite.AutoConfig(d))
			}
		}
		s.buildFocusrite()
	}

	serial := fc.SerialNumber
	if serial == "" {
		serial = "-"
	}
	mute := widget.NewCheck("", nil)
	mute.Checked = current != nil && fc.MuteID != 0
	mute.Disable()
	dim := widget.NewCheck("", nil)
	dim.Checked = current != nil && fc.DimID != 0
	dim.Disable()

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("Device:"), device,
		widget.NewLabel("Serial Number"), widget.NewLabel(serial),
		heading("Master:"), layout.NewSpacer(),
		widget.NewLabel("Master Mute"), mute,
		widget.NewLabel("Master Dim"), dim,
		heading("Speaker:"), layout.NewSpacer(),
	)
	for _, id := range domain.Speakers {
		form.Add(widget.NewLabel(speakerLabel(id)))
		form.Add(s.outputSelect(current, fc, id))
	}
	s.focusrite.Objects = []fyne.CanvasObject{form}
	s.focusrite.Refresh()
}

func (s *settingsWindow) outputSelect(dev *focusrite.DeviceInfo, fc focusrite.Config, id domain.SpeakerID) fyne.CanvasObject {
	sel := newSelect(nil, nil)
	if dev == nil {
		return sel
	}
	assigned := fc.Outputs[id]
	options := []string{none}
	selected := none
	for _, out := range dev.Outputs {
		options = append(options, out.Name)
		if assigned.MuteID != 0 && out.MuteID == assigned.MuteID {
			selected = out.Name
		}
	}
	sel.Options = options
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
	s.opts.Store.Update(func(c *config.Config) {
		c.Focusrite = s.draft.Focusrite
		c.MIDI = s.draft.MIDI
	})
	if err := s.opts.Store.Save(); err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	s.win.Hide()
	if s.opts.Apply != nil {
		s.opts.Apply()
	}
}
