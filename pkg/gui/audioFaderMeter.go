package gui

import (
	"fmt"
	"image/color" // Importiere das color-Paket

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type AudioLevelChanged struct {
	LevelMeter *AudioLevelMeter
	Value      float64
}

// AudioLevelMeter ist unser benutzerdefiniertes Widget.
type AudioLevelMeter struct {
	widget.BaseWidget

	faderValue float64
	level      float64
	minLevel   float64
	maxLevel   float64

	title string
	unit  string

	titleLabel *canvas.Text
	fader      *widget.Slider
	faderLabel *canvas.Text
	levelMeter *VerticalBar

	ValueChanged chan AudioLevelChanged
}

func NewAudioFaderMeter(min, max float64, faderValue float64, level float64, title string, unit string, valueChanged chan AudioLevelChanged) *AudioLevelMeter {
	alm := &AudioLevelMeter{
		minLevel:     min,
		maxLevel:     max,
		level:        level,
		faderValue:   faderValue,
		title:        title,
		unit:         unit,
		ValueChanged: valueChanged,
	}
	alm.ExtendBaseWidget(alm)

	alm.titleLabel = canvas.NewText(alm.title, color.White)
	alm.titleLabel.Alignment = fyne.TextAlignCenter

	alm.faderLabel = canvas.NewText("--- "+alm.unit, color.White)
	alm.faderLabel.Alignment = fyne.TextAlignCenter

	alm.fader = widget.NewSlider(min, max)
	alm.fader.Value = alm.faderValue
	alm.fader.Orientation = widget.Vertical

	alm.levelMeter = NewVerticalBar(10)

	alm.fader.OnChanged = func(alm *AudioLevelMeter) func(float64) {
		return func(value float64) {
			alm.faderValue = level
			alm.ValueChanged <- AudioLevelChanged{LevelMeter: alm, Value: value}
			alm.faderLabel.Text = fmt.Sprintf("%.1f %s", alm.faderValue, alm.unit)
			alm.faderLabel.Refresh()
		}
	}(alm)

	return alm
}

func (alm *AudioLevelMeter) SetLevel(level float64) {
	alm.levelMeter.SetValue(level)
}

func (alm *AudioLevelMeter) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(alm.titleLabel, alm.faderLabel, alm.levelMeter, alm.fader, nil)
	return widget.NewSimpleRenderer(c)
}
