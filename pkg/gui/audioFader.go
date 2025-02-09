package gui

import (
	// Importiere das color-Paket

	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const FADER_MAX = 1000

type AudioLevelChanged struct {
	LevelMeter *AudioFader
	Value      float64
}

type AudioFader struct {
	widget.BaseWidget

	faderValueString binding.String
	minLevel         float64
	maxLevel         float64
	scaleLog         bool
	ValueChanged     chan AudioLevelChanged

	fader *widget.Slider
	label *widget.Label
}

func NewAudioFaderMeter(min, max, faderValue float64, scaleLog bool, valueChanged chan AudioLevelChanged) *AudioFader {
	f := &AudioFader{
		faderValueString: binding.NewString(),
		minLevel:         min,
		maxLevel:         max,
		ValueChanged:     valueChanged,
		scaleLog:         scaleLog,
	}
	f.ExtendBaseWidget(f)

	f.fader = widget.NewSlider(0, FADER_MAX)
	f.fader.Orientation = widget.Vertical

	if f.scaleLog {
		f.fader.SetValue(logToLinear(faderValue, min, max) * FADER_MAX)
	} else {
		f.fader.SetValue(normalise(faderValue, min, max) * FADER_MAX)
	}

	f.faderValueString.Set(fmt.Sprintf("%.1f dB", faderValue))

	f.fader.OnChanged = func(f *AudioFader) func(float64) {
		return func(value float64) {
			val := 0.0
			if f.scaleLog {
				val = linearToLog(value/FADER_MAX, min, max)
			} else {
				val = deNormalise(value/FADER_MAX, min, max)
			}
			f.faderValueString.Set(fmt.Sprintf("%.1f dB", val))
			f.ValueChanged <- AudioLevelChanged{LevelMeter: f, Value: val}
		}
	}(f)

	f.label = widget.NewLabelWithData(f.faderValueString)
	f.label.Alignment = fyne.TextAlignCenter

	return f
}

// FIXMME - Fader will no be updated
func (f *AudioFader) SetLevel(level float64) {
	if f.scaleLog {
		f.fader.Value = logToLinear(level, f.minLevel, f.maxLevel) * FADER_MAX
	} else {
		f.fader.Value = normalise(level, f.minLevel, f.maxLevel) * FADER_MAX
	}
	f.fader.Refresh()
	f.faderValueString.Set(fmt.Sprintf("%.1f dB", level))
}

func (f *AudioFader) CreateRenderer() fyne.WidgetRenderer {

	border := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	border.CornerRadius = 5

	c := container.NewStack(border, container.NewBorder(f.label, nil, nil, nil, f.fader))

	return widget.NewSimpleRenderer(c)
}

func (r *AudioFader) MinSize() fyne.Size {
	return fyne.NewSize(90, 120)
}
