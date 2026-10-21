package ui

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	meterMaxDB = 0
	meterMinDB = -60
	// Fall-back rate of IEC 60268-10 Type I peak meters, in dB per second.
	meterDecayDBPerSecond = 11.8
	// The peak readout follows the bars again after this long.
	meterPeakHold = 4 * time.Second
)

var meterScale = []int{0, -3, -6, -9, -12, -15, -18, -21, -24, -30, -35, -40, -45, -50, -55, -60}

// Meter is the stereo level meter: bars coloured by level, a peak readout and
// a dB scale. It decays on its own through a Fyne animation, so it is only
// ever touched on the main thread.
type Meter struct {
	widget.BaseWidget

	left, right float64
	peak        float64
	peakAt      time.Time
	lastTick    time.Time
	enabled     bool
	anim        *fyne.Animation
}

func NewMeter() *Meter {
	m := &Meter{left: meterMinDB, right: meterMinDB, peak: meterMinDB, enabled: true}
	m.ExtendBaseWidget(m)
	return m
}

// Start begins the decay animation. Call it once the app is running.
func (m *Meter) Start() {
	m.lastTick, m.peakAt = time.Now(), time.Now()
	m.anim = fyne.NewAnimation(time.Second, func(float32) { m.tick() })
	m.anim.RepeatCount = fyne.AnimationRepeatForever
	m.anim.Curve = fyne.AnimationLinear
	m.anim.Start()
}

// SetLevels shows new levels. Higher levels show at once, lower ones are
// reached by decay.
func (m *Meter) SetLevels(left, right float64) {
	left, right = clampMeter(left), clampMeter(right)
	update := false
	if left > m.left {
		m.left, update = left, true
	}
	if right > m.right {
		m.right, update = right, true
	}
	if p := max(left, right); p > m.peak {
		m.peak, update = p, true
	}
	if update {
		m.Refresh()
	}
}

func (m *Meter) SetDisabled(disabled bool) {
	if m.enabled == disabled {
		m.enabled = !disabled
		m.Refresh()
	}
}

func (m *Meter) tick() {
	now := time.Now()
	fall := meterDecayDBPerSecond * now.Sub(m.lastTick).Seconds()
	m.lastTick = now
	update := false
	if m.left > meterMinDB {
		m.left, update = max(m.left-fall, meterMinDB), true
	}
	if m.right > meterMinDB {
		m.right, update = max(m.right-fall, meterMinDB), true
	}
	if now.Sub(m.peakAt) >= meterPeakHold {
		m.peakAt = now
		if p := max(m.left, m.right); p != m.peak {
			m.peak, update = p, true
		}
	}
	if update {
		m.Refresh()
	}
}

func clampMeter(v float64) float64 { return min(max(v, meterMinDB), meterMaxDB) }

// meterScaleFraction maps a level to 0..1 of the bar height, linear in dB.
func meterScaleFraction(value float64) float64 {
	if value <= meterMinDB {
		return 0
	}
	if value >= meterMaxDB {
		return 1
	}
	minLin, maxLin := math.Pow(10, meterMinDB/20.0), math.Pow(10, meterMaxDB/20.0)
	valLin := math.Pow(10, value/20)
	return (math.Log10(valLin) - math.Log10(minLin)) / (math.Log10(maxLin) - math.Log10(minLin))
}

func (m *Meter) MinSize() fyne.Size { return fyne.NewSize(90, 120) }

func (m *Meter) CreateRenderer() fyne.WidgetRenderer {
	left := canvas.NewRectangle(grey)
	left.CornerRadius = 2
	right := canvas.NewRectangle(grey)
	right.CornerRadius = 2
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	bg.CornerRadius = 5
	border := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	border.CornerRadius = 5
	shade := canvas.NewRectangle(color.Transparent)
	shade.CornerRadius = 5
	label := canvas.NewText("--- dB", theme.Color(theme.ColorNameForeground))
	label.Alignment = fyne.TextAlignCenter

	scale := map[int]*canvas.Text{}
	for _, v := range meterScale {
		t := canvas.NewText(fmt.Sprintf("%d dB", v), color.White)
		t.TextSize -= 5
		scale[v] = t
	}
	return &meterRenderer{m: m, left: left, right: right, bg: bg, border: border, shade: shade, label: label, scale: scale}
}

type meterRenderer struct {
	m                       *Meter
	left, right, bg, border *canvas.Rectangle
	shade                   *canvas.Rectangle
	label                   *canvas.Text
	scale                   map[int]*canvas.Text
	size                    fyne.Size
}

const (
	meterPadding  = float32(10)
	meterLabelH   = float32(30)
	meterBgBorder = float32(3)
	meterBarW     = float32(10)
	meterScaleH   = float32(20)
)

func (r *meterRenderer) Layout(size fyne.Size) {
	if size != r.size {
		r.size = size
		r.layoutBackground(size)
	}
	r.layoutBars(size)
}

func (r *meterRenderer) layoutBackground(size fyne.Size) {
	r.shade.Move(fyne.NewPos(0, 0))
	r.shade.Resize(size)
	r.border.Move(fyne.NewPos(0, 0))
	r.border.Resize(size)
	r.label.Resize(fyne.NewSize(size.Width, meterLabelH))
	r.label.Move(fyne.NewPos(0, 3))

	scaleH := size.Height - 2*meterPadding - meterLabelH - meterScaleH/2
	for v, t := range r.scale {
		y := size.Height - meterPadding - meterBgBorder - meterScaleH/2 - scaleH*float32(meterScaleFraction(float64(v)))
		t.Move(fyne.NewPos(size.Width/2+meterBgBorder, y))
		t.Resize(fyne.NewSize(30, meterScaleH))
	}

	// Two bars side by side, left of the centre line.
	bgX := size.Width/2 - 2*meterBarW - 3*meterBgBorder
	bgY := meterPadding + meterLabelH + meterBgBorder
	r.bg.Move(fyne.NewPos(bgX, bgY))
	r.bg.Resize(fyne.NewSize(2*meterBarW+3*meterBgBorder, size.Height-bgY-meterPadding+meterBgBorder))
}

func (r *meterRenderer) layoutBars(size fyne.Size) {
	bgY := meterPadding + meterLabelH + meterBgBorder
	bgH := size.Height - bgY - meterPadding + meterBgBorder
	rightX := size.Width/2 - meterBgBorder - meterBarW
	leftX := rightX - meterBarW - meterBgBorder
	for _, b := range []struct {
		rect *canvas.Rectangle
		x    float32
		v    float64
	}{{r.left, leftX, r.m.left}, {r.right, rightX, r.m.right}} {
		h := (bgH - 2*meterBgBorder) * float32(meterScaleFraction(b.v))
		b.rect.Move(fyne.NewPos(b.x, size.Height-h-3*meterBgBorder))
		b.rect.Resize(fyne.NewSize(meterBarW, h))
	}
}

func (r *meterRenderer) MinSize() fyne.Size { return r.m.MinSize() }

func (r *meterRenderer) Refresh() {
	if r.m.enabled {
		r.shade.FillColor = color.Transparent
		r.left.FillColor = levelGradient.at(r.m.left)
		r.right.FillColor = levelGradient.at(r.m.right)
		r.label.Text = fmt.Sprintf("%.1f dB", r.m.peak)
		r.label.Color = levelGradient.at(r.m.peak)
	} else {
		cr, cg, cb, _ := theme.Color(theme.ColorNameInputBackground).RGBA()
		r.shade.FillColor = color.RGBA{uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), 200}
	}
	r.layoutBars(r.m.Size())
	for _, o := range r.Objects() {
		o.Refresh()
	}
}

func (r *meterRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.border, r.bg, r.left, r.right, r.label}
	for _, v := range meterScale {
		objs = append(objs, r.scale[v])
	}
	return append(objs, r.shade)
}

func (r *meterRenderer) Destroy() {
	if r.m.anim != nil {
		r.m.anim.Stop()
	}
}
