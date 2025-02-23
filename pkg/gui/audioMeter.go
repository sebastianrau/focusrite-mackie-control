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

const MAX_LEVEL = 0
const MIN_LEVEL = -60

var SCALE_VALUES []int = []int{0, -3, -6, -9, -12, -15, -18, -21, -24, -30, -35, -40, -45, -50, -55, -60}

// AudioMeter ist ein Widget, das eine vertikale grüne Leiste rendert.
type AudioMeter struct {
	widget.BaseWidget
	colorGradient *Gradient

	mu           sync.Mutex
	valueL       float64
	valueR       float64
	valueMaxHold float64
	unit         string

	stereo bool

	decayRefreshTime time.Duration
	decayRate        float64

	oldSize fyne.Size

	//debug only ups int
}

func NewAudioMeterBar(stereo bool) *AudioMeter {
	b := &AudioMeter{
		colorGradient: nil,

		mu:           sync.Mutex{},
		valueL:       MIN_LEVEL,
		valueR:       MIN_LEVEL,
		valueMaxHold: MIN_LEVEL,
		unit:         "dB",

		stereo: stereo,

		decayRefreshTime: DECAY_UPDATE_RATE,

		oldSize: fyne.NewSize(0, 0),
	}
	b.SetDecayRate(Default)

	b.ExtendBaseWidget(b)

	go b.runAutoDecay()
	go b.runMaxHold()
	// DEBUG go b.runUPS()
	return b
}

func (b *AudioMeter) SetValue(v float64) {
	if v < MIN_LEVEL {
		v = MIN_LEVEL
	} else if v > MAX_LEVEL {
		v = MAX_LEVEL
	}
	b.mu.Lock()
	if b.valueL < v {
		b.valueL = v
	}
	b.valueMaxHold = max(b.valueL, b.valueMaxHold)

	b.mu.Unlock()

	if b.Visible() {
		b.Refresh()
	}

}

func (b *AudioMeter) SetValueStereo(l, r float64) {
	if l < MIN_LEVEL {
		l = MIN_LEVEL
	} else if l > MAX_LEVEL {
		l = MAX_LEVEL
	}

	if r < MIN_LEVEL {
		r = MIN_LEVEL
	} else if r > MAX_LEVEL {
		r = MAX_LEVEL
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
	b.valueMaxHold = max(l, r, b.valueMaxHold)
	b.mu.Unlock()

	if update && b.Visible() {
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
		b.Refresh()
	}
}

func (b *AudioMeter) runMaxHold() {
	ticker := time.NewTicker(MAX_HOLD_TIME)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		b.valueMaxHold = max(b.valueL, b.valueR)
		b.mu.Unlock()
	}

}

//Debug only
/*
func (b *AudioMeter) runUPS() {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for range t.C {
		log.Debugf("Updates: %d/s", b.ups)
		b.ups = 0
	}
}
*/

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
	levelBarLeft := canvas.NewRectangle(GREY)
	levelBarLeft.CornerRadius = 2

	LevelBarRight := canvas.NewRectangle(GREY)
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
	// DEBUG r.bar.ups = r.bar.ups + 1

	padding := float32(10)
	labelHeight := float32(30.0)

	bgBorder := float32(3)
	barWith := float32(10)

	labelWidth := size.Width

	bgX := (size.Width / 2) - barWith - 2*bgBorder
	bgY := padding + labelHeight + bgBorder //from top padding, label, spacer

	bgWidth := barWith + 2*bgBorder                    // Bar + borders left and right
	bgHeight := size.Height - bgY - padding + bgBorder //from Y start down to max size, minus on padding

	if size.Height != r.bar.oldSize.Height || size.Width != r.bar.oldSize.Width {
		r.bar.oldSize = size
		r.LayoutBackground(size, labelWidth, labelHeight, barWith, padding, bgBorder)

		// scale background if stereo
		if r.bar.stereo {
			bgX = bgX - barWith - bgBorder         // one bar and spacing more to the left
			bgWidth = bgWidth + barWith + bgBorder // one bar and spacing more in width
		}

		r.levelBarBg.Move(fyne.NewPos(bgX, bgY))
		r.levelBarBg.Resize(fyne.NewSize(bgWidth, bgHeight))
	}

	scaledValueL := float32(r.logScale(r.bar.valueL, MIN_LEVEL, MAX_LEVEL))
	barLeftHeight := (bgHeight - 2*bgBorder) * scaledValueL
	barLeftX := (size.Width / 2) - bgBorder - barWith
	barLeftY := size.Height - barLeftHeight - 3*bgBorder

	r.levelBarRight.Hidden = !r.bar.stereo
	if r.bar.stereo {
		scaledValueR := float32(r.logScale(r.bar.valueR, MIN_LEVEL, MAX_LEVEL))
		barRightHeight := (bgHeight - 2*bgBorder) * scaledValueR
		barRightX := barLeftX //right bar to place of left bar in mono
		barRightY := size.Height - barRightHeight - 3*bgBorder

		barLeftX = barLeftX - barWith - bgBorder //move left bar to left

		r.levelBarRight.Move(fyne.NewPos(barRightX, barRightY))
		r.levelBarRight.Resize(fyne.NewSize(barWith, barRightHeight))
	}

	r.levelBarLeft.Move(fyne.NewPos(barLeftX, barLeftY))
	r.levelBarLeft.Resize(fyne.NewSize(barWith, barLeftHeight))

	// Draw Label
	r.faderLabel.Text = fmt.Sprintf("%.1f %s", r.bar.valueMaxHold, r.bar.unit)

	if r.bar.colorGradient != nil {
		r.levelBarLeft.FillColor = r.bar.colorGradient.GetColor(r.bar.valueL)
		r.levelBarRight.FillColor = r.bar.colorGradient.GetColor(r.bar.valueR)
		r.faderLabel.Color = r.bar.colorGradient.GetColor(r.bar.valueMaxHold)
	}

}

func (r *verticalBarRenderer) LayoutBackground(size fyne.Size, labelWidth, labelHeight, barWith, padding, bgBorder float32) {

	dbLabelHeight := float32(20.0)

	r.border.Move(fyne.NewPos(0, 0))
	r.border.Resize(fyne.NewSize(size.Width, size.Height))

	r.faderLabel.Resize(fyne.NewSize(labelWidth, labelHeight))
	r.faderLabel.Move(fyne.NewPos(0, 3))

	bgHeight := size.Height - 2*padding - labelHeight - (dbLabelHeight / 2)

	for dbValue, label := range r.scale {
		scaledLabelPos := r.logScale(float64(dbValue), MIN_LEVEL, MAX_LEVEL)
		labelX := (size.Width)/2 + bgBorder
		labelY := (size.Height - padding - bgBorder - (dbLabelHeight / 2)) - (bgHeight * float32(scaledLabelPos))
		label.Move(fyne.NewPos(labelX, labelY))
		label.Resize(fyne.NewSize(30, dbLabelHeight))
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
