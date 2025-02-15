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

const DECAY_UPDATE_RATE time.Duration = 150 * time.Millisecond
const MAX_HOLD_TIME = 4 * time.Second
const MAX_HOLD_UPDATE_TIME = 500 * time.Millisecond

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

	levelMono          float64
	valueMaxHold       float64
	levelMaxUpdateTime time.Time

	mu               sync.Mutex
	decayRefreshTime time.Duration
	decayRate        float64

	oldSize   fyne.Size
	oldLevelL int
	oldLevelR int

	ups int
}

func NewAudioMeterBar(maxValue float64, stereo bool) *AudioMeter {
	b := &AudioMeter{
		unit:     "dB",
		minValue: MIN_LEVEL,
		maxValue: maxValue,

		valueL: math.MinInt,
		valueR: math.MinInt,

		stereo: stereo,

		levelMono:          MIN_LEVEL,
		valueMaxHold:       MIN_LEVEL,
		levelMaxUpdateTime: time.Now(),

		mu:               sync.Mutex{},
		decayRefreshTime: DECAY_UPDATE_RATE,

		oldSize: fyne.NewSize(0.0, 0),
	}
	b.SetDecayRate(Default)
	b.ExtendBaseWidget(b)

	go b.runAutoDecay()
	go b.runMaxHold()
	go b.runUPS()
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
	b.levelMono = b.valueL
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

	update := false
	b.mu.Lock()
	if b.valueL < l {
		b.valueL = l
		update = true
	}
	if b.valueR < r {
		b.valueR = r
		update = true
	}

	b.levelMono = max(l, r)
	b.mu.Unlock()

	if update {
		b.Refresh()
	}
}

func (b *AudioMeter) runAutoDecay() {
	t := time.NewTicker(b.decayRefreshTime)
	defer t.Stop()

	for range t.C {
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
		b.mu.Unlock()
	}
}

func (b *AudioMeter) runMaxHold() {
	ticker := time.NewTicker(MAX_HOLD_UPDATE_TIME)
	defer ticker.Stop()

	for range ticker.C {
		if b.valueMaxHold < b.levelMono || time.Since(b.levelMaxUpdateTime) > MAX_HOLD_TIME {
			b.valueMaxHold = b.levelMono
			b.levelMaxUpdateTime = time.Now()
		}
	}
}

func (b *AudioMeter) runUPS() {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for range t.C {
		log.Debugf("Updates: %d/s", b.ups)
		b.ups = 0
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

	r.bar.ups = r.bar.ups + 1

	padding := float32(10)
	labelHeight := float32(30.0)
	bgBorder := float32(3)

	labelWidth := size.Width

	if size.Height != r.bar.oldSize.Height || size.Width != r.bar.oldSize.Width {
		r.bar.oldSize = size
		log.Debugf("Update Background")
		r.LayoutBackground(size, labelWidth, labelHeight, padding)
	}

	scaledValueL := float32(r.logScale(r.bar.valueL, r.bar.minValue, r.bar.maxValue))
	scaledValueR := float32(r.logScale(r.bar.valueR, r.bar.minValue, r.bar.maxValue))

	barWith := float32(10)
	barHightL := (size.Height - 3*padding - labelHeight) * scaledValueL
	barHightR := (size.Height - 3*padding - labelHeight) * scaledValueR

	barLeftX := (size.Width)/2 - barWith - padding
	barRightX := (size.Width)/2 - barWith - padding

	barLeftY := size.Height - barHightL - 2*padding
	barRightY := size.Height - barHightR - 2*padding

	bgWidth := barWith + 2*bgBorder
	bgHeight := (size.Height - 3*padding - labelHeight) + 2*bgBorder

	bgX := barLeftX - bgBorder
	bgY := size.Height - (size.Height - 3*padding - labelHeight) - 2*padding - bgBorder

	r.levelBarRight.Hidden = !r.bar.stereo

	if r.bar.stereo {
		//make BG Size bigger
		bgWidth = bgWidth + bgBorder + barWith
		//	move BG to left
		bgX = bgX - bgBorder - barWith
		//move left bar to left
		barLeftX = barLeftX - barWith - bgBorder
	}

	r.levelBarBg.Move(fyne.NewPos(bgX, bgY))
	r.levelBarBg.Resize(fyne.NewSize(bgWidth, bgHeight))

	r.levelBarLeft.Move(fyne.NewPos(barLeftX, barLeftY))
	r.levelBarLeft.Resize(fyne.NewSize(barWith, barHightL))

	r.levelBarRight.Move(fyne.NewPos(barRightX, barRightY))
	r.levelBarRight.Resize(fyne.NewSize(barWith, barHightR))

	r.faderLabel.Text = fmt.Sprintf("%.1f %s", r.bar.valueMaxHold, r.bar.unit)

	if r.bar.colorGradient != nil {
		r.levelBarLeft.FillColor = r.bar.colorGradient.GetColor(r.bar.valueL)
		r.levelBarRight.FillColor = r.bar.colorGradient.GetColor(r.bar.valueR)
		r.faderLabel.Color = r.bar.colorGradient.GetColor(r.bar.valueMaxHold)
	}
}

func (r *verticalBarRenderer) LayoutBackground(size fyne.Size, labelWidth, labelHeight, padding float32) {
	r.border.Move(fyne.NewPos(0, 0))
	r.border.Resize(fyne.NewSize(size.Width, size.Height))

	r.faderLabel.Resize(fyne.NewSize(labelWidth, labelHeight))
	r.faderLabel.Move(fyne.NewPos(0, 3))

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
