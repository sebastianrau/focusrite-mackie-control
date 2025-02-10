package gui

import (
	"fmt"
	"image/color"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// decay per 50ms should be 0,59db/50ms
const DROP_RATE_DB_PER_SECOND float64 = 11.8
const DECAY_UPDATE_RATE time.Duration = 50 * time.Millisecond
const MAX_HOLD_TIME = 4 * time.Second
const MAX_HOLD_UPDATE_TIME = 100 * time.Millisecond

const MIN_LEVEL = -60

var SCALE_VALUES []int = []int{0, -3, -6, -9, -12, -15, -18, -21, -24, -30, -35, -40, -45, -50, -55, -60}

// AudioMeter ist ein Widget, das eine vertikale grüne Leiste rendert.
type AudioMeter struct {
	widget.BaseWidget
	colorGradient *Gradient

	value    float64
	unit     string
	minValue float64
	maxValue float64

	valueMaxHold       float64
	levelMaxUpdateTime time.Time

	mu               sync.Mutex
	decayRefreshTime time.Duration
	decayRate        float64
}

func NewAudioMeterBar(maxValue float64) *AudioMeter {
	b := &AudioMeter{
		unit:     "dB",
		minValue: MIN_LEVEL,
		maxValue: maxValue,
		value:    math.MinInt,

		valueMaxHold:       MIN_LEVEL,
		levelMaxUpdateTime: time.Now(),

		mu:               sync.Mutex{},
		decayRefreshTime: DECAY_UPDATE_RATE,
		decayRate:        DROP_RATE_DB_PER_SECOND / float64(time.Millisecond) * float64(DECAY_UPDATE_RATE) / float64(time.Microsecond),
	}
	b.ExtendBaseWidget(b)

	go b.runAutoDecay()
	go b.runMaxHold()
	return b
}

func (b *AudioMeter) SetValue(v float64) {
	if v < b.minValue {
		v = b.minValue
	} else if v > b.maxValue {
		v = b.maxValue
	}
	b.mu.Lock()
	if b.value < v {
		b.value = v
	}
	b.mu.Unlock()
	b.Refresh()
}

func (b *AudioMeter) runAutoDecay() {
	ticker := time.NewTicker(b.decayRefreshTime)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		if b.value > MIN_LEVEL {
			b.value -= b.decayRate
			if b.value < MIN_LEVEL {
				b.value = MIN_LEVEL
			}
			b.Refresh()
		}
		b.mu.Unlock()

	}
}

func (b *AudioMeter) runMaxHold() {

	ticker := time.NewTicker(MAX_HOLD_UPDATE_TIME)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		level := b.value
		b.mu.Unlock()

		if b.valueMaxHold < level || time.Since(b.levelMaxUpdateTime) > MAX_HOLD_TIME {
			b.valueMaxHold = level
			b.levelMaxUpdateTime = time.Now()
			b.Refresh()
		}
	}

}

// Renderer
type verticalBarRenderer struct {
	bar        *AudioMeter
	levelBar   *canvas.Rectangle
	levelBarBg *canvas.Rectangle
	border     *canvas.Rectangle
	faderLabel *canvas.Text
	scale      map[int]*canvas.Text
}

func (b *AudioMeter) SetGradient(grad *Gradient) {
	b.colorGradient = grad
}

// CreateRenderer erstellt den Renderer für das Widget
func (b *AudioMeter) CreateRenderer() fyne.WidgetRenderer {
	levelBar := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	levelBar.CornerRadius = 2

	levelBarBg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	levelBarBg.CornerRadius = 5

	border := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	border.CornerRadius = 5

	faderLabel := canvas.NewText("--- "+b.unit, theme.Color(theme.ColorNameForeground))
	faderLabel.Alignment = fyne.TextAlignCenter

	scale := map[int]*canvas.Text{}
	for _, val := range SCALE_VALUES {
		label := canvas.NewText(fmt.Sprintf("%d dB", val), color.White)
		label.TextSize = label.TextSize - 5
		scale[val] = label
	}

	return &verticalBarRenderer{bar: b, levelBar: levelBar, border: border, faderLabel: faderLabel, scale: scale, levelBarBg: levelBarBg}
}

func (r *verticalBarRenderer) Layout(size fyne.Size) {
	scaledValue := float32(r.logScale(r.bar.value, r.bar.minValue, r.bar.maxValue))

	padding := float32(10)
	labelHeight := float32(30.0)
	labelWidth := size.Width

	r.border.Move(fyne.NewPos(0, 0))
	r.border.Resize(fyne.NewSize(size.Width, size.Height))

	r.faderLabel.Resize(fyne.NewSize(labelWidth, labelHeight))
	r.faderLabel.Move(fyne.NewPos(0, 3))

	rectWith := float32(10)
	rectHight := (size.Height - 3*padding - labelHeight) * scaledValue
	rectX := (size.Width)/2 - rectWith - padding
	rectY := size.Height - rectHight - 2*padding

	bgBorder := float32(3)
	bgWidth := rectWith + 2*bgBorder
	bgHeight := (size.Height - 3*padding - labelHeight) + 2*bgBorder
	bgX := rectX - bgBorder
	bgY := size.Height - (size.Height - 3*padding - labelHeight) - 2*padding - bgBorder

	r.levelBarBg.Move(fyne.NewPos(bgX, bgY))
	r.levelBarBg.Resize(fyne.NewSize(bgWidth, bgHeight))

	r.levelBar.Move(fyne.NewPos(rectX, rectY))
	r.levelBar.Resize(fyne.NewSize(rectWith, rectHight))

	for dbValue, label := range r.scale {

		scaledLabelPos := r.logScale(float64(dbValue), r.bar.minValue, r.bar.maxValue)

		labelX := (size.Width) / 2
		labelY := size.Height - (size.Height-3*padding-labelHeight)*float32(scaledLabelPos) - labelHeight

		label.Move(fyne.NewPos(labelX, labelY))
		label.Resize(fyne.NewSize(30, 20))
	}

}

func (r *verticalBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(90, 120)
}

func (r *verticalBarRenderer) Refresh() {
	r.Layout(r.bar.Size())

	if r.bar.colorGradient != nil {
		c := r.bar.colorGradient.GetColor(r.bar.value)
		r.levelBar.FillColor = c
	}

	r.faderLabel.Text = fmt.Sprintf("%.1f %s", r.bar.valueMaxHold, r.bar.unit)
	if r.bar.colorGradient != nil {
		r.faderLabel.Color = r.bar.colorGradient.GetColor(r.bar.valueMaxHold)
	}

	canvas.Refresh(r.faderLabel)
	canvas.Refresh(r.levelBar)
	canvas.Refresh(r.border)
}

// Objects gibt die UI-Elemente zurück
func (r *verticalBarRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{
		r.border,
		r.levelBarBg,
		r.levelBar,
		r.faderLabel,
	}
	for _, label := range r.scale {
		objects = append(objects, label)
	}
	return objects
}

func (r *verticalBarRenderer) Destroy() {}

func (r *verticalBarRenderer) logScale(value, min, max float64) float64 {
	if value <= min {
		return 0
	}
	if value >= max {
		return 1
	}

	// Exponentielle Skalierung für dB-Werte (Vermeidung von Problemen mit negativen dB)
	minLin := math.Pow(10, min/20) // dB in linearen Wert umwandeln
	maxLin := math.Pow(10, max/20)
	valLin := math.Pow(10, value/20)

	return (math.Log10(valLin) - math.Log10(minLin)) / (math.Log10(maxLin) - math.Log10(minLin))
}
