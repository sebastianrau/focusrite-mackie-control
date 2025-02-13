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

type DecayRate float64

const (
	Slow      DecayRate = 4.0
	EBU_Slow  DecayRate = 6.3
	IEC_TypII DecayRate = 8.6
	IEC_TypI  DecayRate = 11.8
	Fast      DecayRate = 20.0

	Default DecayRate = IEC_TypI
)

const DECAY_UPDATE_RATE time.Duration = 50 * time.Millisecond
const MAX_HOLD_TIME = 4 * time.Second
const MAX_HOLD_UPDATE_TIME = 100 * time.Millisecond

const MIN_LEVEL = -60

var SCALE_VALUES []int = []int{0, -3, -6, -9, -12, -15, -18, -21, -24, -30, -35, -40, -45, -50, -55, -60}

// AudioMeter ist ein Widget, das eine vertikale grüne Leiste rendert.
type AudioMeter struct {
	widget.BaseWidget
	colorGradient *Gradient

	valueL   float64
	valueR   float64
	unit     string
	minValue float64
	maxValue float64

	stereo bool

	valueMaxHold       float64
	levelMaxUpdateTime time.Time

	mu               sync.Mutex
	decayRefreshTime time.Duration
	decayRate        float64
}

func NewAudioMeterBar(maxValue float64, stereo bool) *AudioMeter {
	b := &AudioMeter{
		unit:     "dB",
		minValue: MIN_LEVEL,
		maxValue: maxValue,

		valueL: math.MinInt,
		valueR: math.MinInt,

		stereo: stereo,

		valueMaxHold:       MIN_LEVEL,
		levelMaxUpdateTime: time.Now(),

		mu:               sync.Mutex{},
		decayRefreshTime: DECAY_UPDATE_RATE,
	}
	b.SetDecayRate(Default)
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
	if b.valueL < v {
		b.valueL = v
	}
	b.mu.Unlock()
	b.Refresh()
}

func (b *AudioMeter) SetValueStereo(l, r float64) {
	if l < b.minValue {
		l = b.minValue
	} else if l > b.maxValue {
		l = b.maxValue
	}

	if r < b.minValue {
		r = b.minValue
	} else if r > b.maxValue {
		r = b.maxValue
	}

	b.mu.Lock()
	if b.valueL < l {
		b.valueL = l
	}
	if b.valueR < r {
		b.valueR = r
	}
	b.mu.Unlock()
	b.Refresh()
}

func (b *AudioMeter) runAutoDecay() {
	ticker := time.NewTicker(b.decayRefreshTime)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		if b.valueL > MIN_LEVEL {
			b.valueL -= b.decayRate
			if b.valueL < MIN_LEVEL {
				b.valueL = MIN_LEVEL
			}
		}
		if b.valueR > MIN_LEVEL {
			b.valueR -= b.decayRate
			if b.valueR < MIN_LEVEL {
				b.valueR = MIN_LEVEL
			}

		}
		b.Refresh()
		b.mu.Unlock()
	}
}

func (b *AudioMeter) runMaxHold() {

	ticker := time.NewTicker(MAX_HOLD_UPDATE_TIME)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		level := max(b.valueL, b.valueR)
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
	bar           *AudioMeter
	levelBarLeft  *canvas.Rectangle
	levelBarRight *canvas.Rectangle
	levelBarBg    *canvas.Rectangle
	border        *canvas.Rectangle
	faderLabel    *canvas.Text
	scale         map[int]*canvas.Text
}

func (b *AudioMeter) SetGradient(grad *Gradient) {
	b.colorGradient = grad
}

func (b *AudioMeter) SetDecayRate(rate DecayRate) {
	b.decayRate = float64(rate) / float64(time.Millisecond) * float64(DECAY_UPDATE_RATE) / float64(time.Microsecond)
}

// CreateRenderer erstellt den Renderer für das Widget
func (b *AudioMeter) CreateRenderer() fyne.WidgetRenderer {
	levelBarLeft := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	levelBarLeft.CornerRadius = 2

	LevelBarRight := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	LevelBarRight.CornerRadius = 2

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

	return &verticalBarRenderer{bar: b, levelBarLeft: levelBarLeft, levelBarRight: LevelBarRight, border: border, faderLabel: faderLabel, scale: scale, levelBarBg: levelBarBg}
}

func (r *verticalBarRenderer) Layout(size fyne.Size) {
	scaledValueL := float32(r.logScale(r.bar.valueL, r.bar.minValue, r.bar.maxValue))
	scaledValueR := float32(r.logScale(r.bar.valueR, r.bar.minValue, r.bar.maxValue))

	padding := float32(10)
	labelHeight := float32(30.0)
	labelWidth := size.Width

	r.border.Move(fyne.NewPos(0, 0))
	r.border.Resize(fyne.NewSize(size.Width, size.Height))

	r.faderLabel.Resize(fyne.NewSize(labelWidth, labelHeight))
	r.faderLabel.Move(fyne.NewPos(0, 3))

	rectWith := float32(10)
	rectHightL := (size.Height - 3*padding - labelHeight) * scaledValueL
	rectHightR := (size.Height - 3*padding - labelHeight) * scaledValueR

	barLeftX := (size.Width)/2 - rectWith - padding
	barRightX := (size.Width)/2 - rectWith - padding

	barLeftY := size.Height - rectHightL - 2*padding
	barRightY := size.Height - rectHightR - 2*padding

	bgBorder := float32(3)
	bgWidth := rectWith + 2*bgBorder
	bgHeight := (size.Height - 3*padding - labelHeight) + 2*bgBorder

	bgX := barLeftX - bgBorder
	bgY := size.Height - (size.Height - 3*padding - labelHeight) - 2*padding - bgBorder

	r.levelBarRight.Hidden = !r.bar.stereo

	if r.bar.stereo {
		//make BG Size bigger
		bgWidth = bgWidth + bgBorder + rectWith
		//	move BG to left
		bgX = bgX - bgBorder - rectWith

		//move left bar to left
		barLeftX = barLeftX - rectWith - bgBorder
	}

	r.levelBarBg.Move(fyne.NewPos(bgX, bgY))
	r.levelBarBg.Resize(fyne.NewSize(bgWidth, bgHeight))

	r.levelBarLeft.Move(fyne.NewPos(barLeftX, barLeftY))
	r.levelBarLeft.Resize(fyne.NewSize(rectWith, rectHightL))

	r.levelBarRight.Move(fyne.NewPos(barRightX, barRightY))
	r.levelBarRight.Resize(fyne.NewSize(rectWith, rectHightR))

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
		leftColor := r.bar.colorGradient.GetColor(r.bar.valueL)
		rightColor := r.bar.colorGradient.GetColor(r.bar.valueL)

		r.levelBarLeft.FillColor = leftColor
		r.levelBarRight.FillColor = rightColor
	}

	r.faderLabel.Text = fmt.Sprintf("%.1f %s", r.bar.valueMaxHold, r.bar.unit)
	if r.bar.colorGradient != nil {
		r.faderLabel.Color = r.bar.colorGradient.GetColor(r.bar.valueMaxHold)
	}

	canvas.Refresh(r.faderLabel)
	canvas.Refresh(r.levelBarLeft)
	canvas.Refresh(r.levelBarRight)
	canvas.Refresh(r.border)
}

// Objects gibt die UI-Elemente zurück
func (r *verticalBarRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{
		r.border,
		r.levelBarBg,
		r.levelBarLeft,
		r.levelBarRight,
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
