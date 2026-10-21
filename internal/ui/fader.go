package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// Fader is the vertical master volume slider. The value is shown separately
// in the channel strip header.
type Fader struct {
	widget.Slider
}

func NewFader(onChange func(db int)) *Fader {
	f := &Fader{}
	f.Min, f.Max, f.Step = domain.MinVolumeDB, domain.MaxVolumeDB, 1
	f.Orientation = widget.Vertical
	f.Value = domain.MinVolumeDB
	f.OnChanged = func(v float64) { onChange(int(v)) }
	f.ExtendBaseWidget(f)
	return f
}

// SetDB moves the slider without reporting a change.
func (f *Fader) SetDB(db int) {
	if int(f.Value) != db {
		f.Value = float64(db)
		f.Refresh()
	}
}

func (f *Fader) SetDisabled(disabled bool) {
	if disabled == f.Disabled() {
		return
	}
	if disabled {
		f.Disable()
	} else {
		f.Enable()
	}
}

func (f *Fader) MinSize() fyne.Size {
	return fyne.NewSize(28, f.Slider.MinSize().Height)
}
