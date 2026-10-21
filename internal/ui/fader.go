package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// Fader is the vertical master volume slider with its dB readout.
type Fader struct {
	widget.BaseWidget
	slider *widget.Slider
	label  *widget.Label
}

func NewFader(onChange func(db int)) *Fader {
	f := &Fader{
		slider: widget.NewSlider(domain.MinVolumeDB, domain.MaxVolumeDB),
		label:  widget.NewLabel(""),
	}
	f.slider.Step = 1
	f.slider.Orientation = widget.Vertical
	f.slider.OnChanged = func(v float64) {
		f.showDB(int(v))
		onChange(int(v))
	}
	f.label.Alignment = fyne.TextAlignCenter
	f.label.TextStyle.Monospace = true
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

func (f *Fader) showDB(db int) { f.label.SetText(fmt.Sprintf("%d dB", db)) }

func (f *Fader) SetDisabled(disabled bool) {
	if disabled {
		f.slider.Disable()
		f.label.Importance = widget.LowImportance
	} else {
		f.slider.Enable()
		f.label.Importance = widget.MediumImportance
	}
	f.label.Refresh()
}

func (f *Fader) MinSize() fyne.Size { return fyne.NewSize(90, 160) }

func (f *Fader) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.CornerRadius = 5
	return widget.NewSimpleRenderer(container.NewStack(bg, container.NewBorder(f.label, nil, nil, nil, f.slider)))
}
