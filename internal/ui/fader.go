package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// Fader is the vertical master volume slider with its dB readout.
type Fader struct {
	widget.BaseWidget
	slider *widget.Slider
	text   binding.String
	label  *widget.Label
}

func NewFader(onChange func(db int)) *Fader {
	f := &Fader{
		slider: widget.NewSlider(domain.MinVolumeDB, domain.MaxVolumeDB),
		text:   binding.NewString(),
	}
	f.slider.Step = 1
	f.slider.Orientation = widget.Vertical
	f.slider.OnChanged = func(v float64) {
		f.showDB(int(v))
		onChange(int(v))
	}
	f.label = widget.NewLabelWithData(f.text)
	f.label.Alignment = fyne.TextAlignCenter
	f.ExtendBaseWidget(f)
	f.SetDB(domain.MinVolumeDB)
	return f
}

// SetDB moves the slider without reporting a change.
func (f *Fader) SetDB(db int) {
	if int(f.slider.Value) != db {
		f.slider.Value = float64(db)
		f.slider.Refresh()
	}
	f.showDB(db)
}

func (f *Fader) showDB(db int) { _ = f.text.Set(fmt.Sprintf("%.1f dB", float64(db))) }

func (f *Fader) LabelText() string {
	s, _ := f.text.Get()
	return s
}

func (f *Fader) SetDisabled(disabled bool) {
	if disabled == f.slider.Disabled() {
		return
	}
	if disabled {
		f.slider.Disable()
		f.label.Importance = widget.LowImportance
	} else {
		f.slider.Enable()
		f.label.Importance = widget.MediumImportance
	}
	f.label.Refresh()
}

func (f *Fader) MinSize() fyne.Size { return fyne.NewSize(90, 120) }

func (f *Fader) CreateRenderer() fyne.WidgetRenderer {
	border := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	border.CornerRadius = 5
	return widget.NewSimpleRenderer(container.NewStack(border, container.NewBorder(f.label, nil, nil, nil, f.slider)))
}
