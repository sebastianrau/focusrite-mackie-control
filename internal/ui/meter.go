package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	meterMinDB = -60.0
	meterMaxDB = 0.0
	// Fall-back rate of IEC 60268-10 Type I peak meters.
	meterDecayDBPerSecond = 11.8
	meterPeakHold         = 2 * time.Second
)

var (
	meterTicks    = []float64{0, -6, -12, -18, -24, -30, -40, -50, -60}
	meterGradient = gradient{
		{-60, color.NRGBA{0, 102, 0, 255}},
		{-15, color.NRGBA{51, 255, 51, 255}},
		{-6, color.NRGBA{255, 255, 51, 255}},
		{0, color.NRGBA{255, 153, 51, 255}},
	}
)

// Meter is a stereo peak meter with decay and peak hold.
type Meter struct {
	widget.BaseWidget

	left, right float64 // displayed levels in dBFS
	peak        float64
	peakAt      time.Time
	lastTick    time.Time
	disabled    bool
	anim        *fyne.Animation
}

func NewMeter() *Meter {
	m := &Meter{left: meterMinDB, right: meterMinDB, peak: meterMinDB}
	m.ExtendBaseWidget(m)
	return m
}

// Start begins the decay animation. Call it once the app is running.
func (m *Meter) Start() {
	m.lastTick = time.Now()
	m.anim = fyne.NewAnimation(time.Second, func(float32) { m.tick() })
	m.anim.RepeatCount = fyne.AnimationRepeatForever
	m.anim.Curve = fyne.AnimationLinear
	m.anim.Start()
}

// SetLevels shows new levels. Higher levels show at once; lower ones are
// reached by decay.
func (m *Meter) SetLevels(left, right float64) {
	left, right = clampMeter(left), clampMeter(right)
	changed := false
	if left > m.left {
		m.left, changed = left, true
	}
	if right > m.right {
		m.right, changed = right, true
	}
	if p := max(left, right); p >= m.peak {
		m.peak, m.peakAt, changed = p, time.Now(), true
	}
	if changed {
		m.Refresh()
	}
}

func (m *Meter) SetDisabled(disabled bool) {
	if m.disabled != disabled {
		m.disabled = disabled
		m.Refresh()
	}
}

func (m *Meter) tick() {
	now := time.Now()
	fall := meterDecayDBPerSecond * now.Sub(m.lastTick).Seconds()
	m.lastTick = now
	changed := false
	if m.left > meterMinDB {
		m.left, changed = max(m.left-fall, meterMinDB), true
	}
	if m.right > meterMinDB {
		m.right, changed = max(m.right-fall, meterMinDB), true
	}
	if m.peak > meterMinDB && now.Sub(m.peakAt) > meterPeakHold {
		m.peak, m.peakAt, changed = max(m.left, m.right), now, true
	}
	if changed {
		m.Refresh()
	}
}

func clampMeter(db float64) float64 { return min(max(db, meterMinDB), meterMaxDB) }

// meterFraction maps a level to 0..1 of the meter height.
func meterFraction(db float64) float32 {
	return float32((clampMeter(db) - meterMinDB) / (meterMaxDB - meterMinDB))
}

func (m *Meter) MinSize() fyne.Size { return fyne.NewSize(90, 160) }

func (m *Meter) CreateRenderer() fyne.WidgetRenderer {
	r := &meterRenderer{
		m:     m,
		bg:    canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground)),
		track: canvas.NewRectangle(theme.Color(theme.ColorNameBackground)),
		left:  canvas.NewRectangle(meterGradient.at(meterMinDB)),
		right: canvas.NewRectangle(meterGradient.at(meterMinDB)),
		shade: canvas.NewRectangle(color.Transparent),
		peak:  canvas.NewText("", theme.Color(theme.ColorNameForeground)),
	}
	r.bg.CornerRadius, r.track.CornerRadius, r.shade.CornerRadius = 5, 4, 5
	r.left.CornerRadius, r.right.CornerRadius = 2, 2
	r.peak.Alignment = fyne.TextAlignCenter
	r.peak.TextStyle.Monospace = true
	for _, db := range meterTicks {
		t := canvas.NewText(fmt.Sprintf("%.0f", db), theme.Color(theme.ColorNamePlaceHolder))
		t.TextSize = theme.CaptionTextSize()
		r.ticks = append(r.ticks, t)
	}
	r.Refresh()
	return r
}

type meterRenderer struct {
	m                       *Meter
	bg, track, left, right  *canvas.Rectangle
	shade                   *canvas.Rectangle
	peak                    *canvas.Text
	ticks                   []*canvas.Text
	barTop, barHeight, barX float32
}

const (
	meterPad    = 8
	meterLabelH = 22
	meterBarW   = 10
	meterGap    = 3
)

func (r *meterRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.shade.Resize(size)
	r.peak.Move(fyne.NewPos(0, meterPad/2))
	r.peak.Resize(fyne.NewSize(size.Width, meterLabelH))

	trackW := float32(2*meterBarW + 3*meterGap)
	top := float32(meterPad + meterLabelH)
	h := size.Height - top - meterPad
	x := size.Width/2 - trackW + meterGap
	r.track.Move(fyne.NewPos(x, top))
	r.track.Resize(fyne.NewSize(trackW, h))
	r.barX, r.barTop, r.barHeight = x+meterGap, top+meterGap, h-2*meterGap

	for i, db := range meterTicks {
		t := r.ticks[i]
		ts := t.MinSize()
		y := r.barTop + r.barHeight*(1-meterFraction(db)) - ts.Height/2
		t.Move(fyne.NewPos(x+trackW+4, y))
		t.Resize(ts)
	}
	r.placeBars()
}

func (r *meterRenderer) placeBars() {
	for i, bar := range []*canvas.Rectangle{r.left, r.right} {
		db := r.m.left
		if i == 1 {
			db = r.m.right
		}
		h := r.barHeight * meterFraction(db)
		bar.Move(fyne.NewPos(r.barX+float32(i)*(meterBarW+meterGap), r.barTop+r.barHeight-h))
		bar.Resize(fyne.NewSize(meterBarW, h))
		bar.FillColor = meterGradient.at(db)
	}
}

func (r *meterRenderer) MinSize() fyne.Size { return r.m.MinSize() }

func (r *meterRenderer) Refresh() {
	r.placeBars()
	if r.m.disabled {
		r.peak.Text = "--"
		r.peak.Color = theme.Color(theme.ColorNameDisabled)
		bg := theme.Color(theme.ColorNameInputBackground)
		cr, cg, cb, _ := bg.RGBA()
		r.shade.FillColor = color.NRGBA{uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), 200}
	} else {
		r.peak.Text = fmt.Sprintf("%.1f dB", r.m.peak)
		r.peak.Color = meterGradient.at(r.m.peak)
		r.shade.FillColor = color.Transparent
	}
	for _, o := range r.Objects() {
		o.Refresh()
	}
}

func (r *meterRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.bg, r.track, r.left, r.right, r.peak}
	for _, t := range r.ticks {
		objs = append(objs, t)
	}
	return append(objs, r.shade)
}

func (r *meterRenderer) Destroy() {
	if r.m.anim != nil {
		r.m.anim.Stop()
	}
}

// gradient interpolates colours between stops sorted by value.
type gradient []struct {
	db float64
	c  color.NRGBA
}

func (g gradient) at(db float64) color.Color {
	if db <= g[0].db {
		return g[0].c
	}
	for i := 1; i < len(g); i++ {
		lo, hi := g[i-1], g[i]
		if db <= hi.db {
			f := (db - lo.db) / (hi.db - lo.db)
			mix := func(a, b uint8) uint8 { return uint8(float64(a) + f*(float64(b)-float64(a))) }
			return color.NRGBA{mix(lo.c.R, hi.c.R), mix(lo.c.G, hi.c.G), mix(lo.c.B, hi.c.B), 255}
		}
	}
	return g[len(g)-1].c
}
