package gui

import (
	"fmt"
	"image/color" // Importiere das color-Paket
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const DROP_RATE_DB_PER_SECOND float64 = 11.8
const DECAY_UPDATE_RATE time.Duration = 50 * time.Millisecond
const MAX_HOLD_TIME = 4 * time.Second

type AudioLevelChanged struct {
	LevelMeter *AudioLevelMeter
	Value      float64
}

// AudioLevelMeter ist unser benutzerdefiniertes Widget.
type AudioLevelMeter struct {
	widget.BaseWidget

	faderValue float64

	level    float64
	minLevel float64
	maxLevel float64

	levelMax           float64
	levelMaxUpdateTime time.Time

	mu               sync.Mutex
	decayRefreshTime time.Duration
	decayRate        float64

	title string
	unit  string

	colorGradient *Gradient

	//titleLabel *canvas.Text
	fader      *widget.Slider
	faderLabel *canvas.Text
	levelMeter *VerticalBar

	ValueChanged chan AudioLevelChanged
}

func NewAudioFaderMeter(min, max float64, faderValue float64, level float64, title string, unit string, valueChanged chan AudioLevelChanged) *AudioLevelMeter {
	alm := &AudioLevelMeter{
		faderValue: faderValue,

		level:    level,
		minLevel: min,
		maxLevel: max,

		mu:               sync.Mutex{},
		decayRefreshTime: DECAY_UPDATE_RATE,
		decayRate:        DROP_RATE_DB_PER_SECOND / float64(time.Millisecond) * float64(DECAY_UPDATE_RATE) / float64(time.Microsecond),

		title: title,
		unit:  unit,

		ValueChanged: valueChanged,
	}
	alm.ExtendBaseWidget(alm)

	//alm.titleLabel = canvas.NewText(alm.title, color.White)
	//alm.titleLabel.Alignment = fyne.TextAlignCenter

	alm.faderLabel = canvas.NewText("--- "+alm.unit, color.White)
	alm.faderLabel.Alignment = fyne.TextAlignCenter

	alm.fader = widget.NewSlider(min, max)
	alm.fader.Value = alm.faderValue
	alm.fader.Orientation = widget.Vertical

	alm.levelMeter = NewVerticalBar(10, alm.minLevel, alm.maxLevel)

	alm.fader.OnChanged = func(alm *AudioLevelMeter) func(float64) {
		return func(value float64) {
			alm.faderValue = level
			alm.ValueChanged <- AudioLevelChanged{LevelMeter: alm, Value: value}
		}
	}(alm)

	go alm.runAutoDecay()

	return alm
}
func (alm *AudioLevelMeter) updateMaxHold(level float64) {
	if alm.maxLevel < level || time.Since(alm.levelMaxUpdateTime) > MAX_HOLD_TIME {
		alm.maxLevel = level
		alm.faderLabel.Text = fmt.Sprintf("%.1f %s", alm.maxLevel, alm.unit)
		if alm.colorGradient != nil {
			alm.faderLabel.Color = alm.colorGradient.GetColor(level)
		}
		alm.faderLabel.Refresh()
		alm.levelMaxUpdateTime = time.Now()
	}
}

func (alm *AudioLevelMeter) SetLevel(level float64) {
	alm.levelMeter.SetValue(level)
	alm.updateMaxHold(level)
}

func (alm *AudioLevelMeter) SetGradient(grad *Gradient) {
	alm.colorGradient = grad
	alm.levelMeter.SetGradient(grad)
}

func (alm *AudioLevelMeter) CreateRenderer() fyne.WidgetRenderer {
	border := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))

	c := container.NewBorder(nil, alm.faderLabel, container.NewStack(border, alm.levelMeter), nil, alm.fader)
	//c := container.NewBorder(alm.titleLabel, alm.faderLabel, alm.levelMeter, alm.fader, nil)
	return widget.NewSimpleRenderer(c)
}

func (b *AudioLevelMeter) runAutoDecay() {
	ticker := time.NewTicker(b.decayRefreshTime)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		if b.level > b.minLevel {
			b.level -= b.decayRate
			if b.level < b.minLevel {
				b.level = b.minLevel
			}
			b.Refresh()
		}
		b.mu.Unlock()
	}
}
