package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/sebastianrau/sweetspot/internal/domain"
)

const (
	meterMaxDB = 0
	// Fall-back rate of IEC 60268-10 Type I peak meters, in dB per second.
	meterDecayDBPerSecond = 11.8
	meterSegments         = 30
)

// meterScales are the dB labels for each meter range. They keep enough
// distance for the labels not to overlap at the window's meter height.
var meterScales = map[int][]float64{
	-40:  {0, -6, -15, -20, -30, -40},
	-60:  {0, -6, -15, -30, -45, -60},
	-80:  {0, -10, -20, -40, -60, -80},
	-96:  {0, -15, -30, -48, -72, -96},
	-127: {0, -15, -30, -60, -90, -127},
}

func meterScale(floor float64) []float64 {
	if sc, ok := meterScales[int(floor)]; ok {
		return sc
	}
	return meterScales[domain.DefaultMeterFloorDB]
}

// Meter is a stereo LED-style meter: segments lit up to the level, coloured
// by zone. It decays on its own through a Fyne animation, so it is only ever
// touched on the main thread.
type Meter struct {
	widget.BaseWidget

	left, right float64 // levels in dBFS, decaying
	floor       float64 // lowest level shown, in dBFS
	litL, litR  int     // segments currently drawn lit
	lastTick    time.Time
	enabled     bool
	paused      bool
	redraws     int // for tests
	anim        *fyne.Animation
}

func NewMeter() *Meter {
	const floor = domain.DefaultMeterFloorDB
	m := &Meter{left: floor, right: floor, floor: floor, enabled: true}
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

// SetLevels shows new levels. Higher levels show at once, lower ones are
// reached by decay.
func (m *Meter) SetLevels(left, right float64) {
	left, right = m.clamp(left), m.clamp(right)
	m.left, m.right = max(m.left, left), max(m.right, right)
	m.redrawIfChanged()
}

// Pause stops the decay animation and all drawing, e.g. while the window
// is hidden. Resume starts both again.
func (m *Meter) Pause() {
	m.paused = true
	if m.anim != nil {
		m.anim.Stop()
	}
}

func (m *Meter) Resume() {
	if !m.paused {
		return
	}
	m.paused = false
	m.lastTick = time.Now()
	if m.anim != nil {
		m.anim.Start()
	}
	m.Refresh()
}

// redrawIfChanged redraws only when the number of lit segments changes.
// Redrawing on every animation frame kept the GPU busy for nothing.
func (m *Meter) redrawIfChanged() {
	l, r := m.litSegments(m.left), m.litSegments(m.right)
	if m.paused || (l == m.litL && r == m.litR) {
		return
	}
	m.redraws++
	m.Refresh()
}

// SetFloor sets the lowest level shown, in dBFS.
func (m *Meter) SetFloor(db float64) {
	if db == m.floor {
		return
	}
	// A silent meter stays silent; others are clamped into the new range.
	old := m.floor
	m.floor = db
	if m.left <= old {
		m.left = db
	}
	if m.right <= old {
		m.right = db
	}
	m.left, m.right = m.clamp(m.left), m.clamp(m.right)
	m.litL, m.litR = -1, -1 // the same count of lit segments now means other levels
	m.redrawIfChanged()
}

// litSegments is the number of segments lit for a level.
func (m *Meter) litSegments(db float64) int {
	n := 0
	for i := range meterSegments {
		if db >= m.segmentDB(i) {
			n = i + 1
		}
	}
	return n
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
	if m.left <= m.floor && m.right <= m.floor {
		return
	}
	m.left = max(m.left-fall, m.floor)
	m.right = max(m.right-fall, m.floor)
	m.redrawIfChanged()
}

func (m *Meter) clamp(v float64) float64 { return min(max(v, m.floor), meterMaxDB) }

// segmentDB is the level at the middle of segment i, counted from the bottom.
func (m *Meter) segmentDB(i int) float64 {
	return m.floor + (float64(i)+0.5)*(meterMaxDB-m.floor)/meterSegments
}

func (m *Meter) MinSize() fyne.Size { return fyne.NewSize(23, 120) }

func (m *Meter) CreateRenderer() fyne.WidgetRenderer {
	r := &meterRenderer{m: m}
	for range 2 * meterSegments {
		seg := canvas.NewRectangle(colRaised)
		seg.CornerRadius = 1.5
		r.segs = append(r.segs, seg)
	}
	r.Refresh()
	return r
}

type meterRenderer struct {
	m    *Meter
	segs []*canvas.Rectangle // left channel bottom to top, then right
}

const (
	meterBarW = 10
	meterGap  = 3
	segGap    = 2
)

func (r *meterRenderer) Layout(size fyne.Size) {
	segH := (size.Height - segGap*(meterSegments-1)) / meterSegments
	for ch := range 2 {
		x := float32(ch) * (meterBarW + meterGap)
		for i := range meterSegments {
			y := size.Height - float32(i+1)*segH - float32(i)*segGap
			s := r.segs[ch*meterSegments+i]
			s.Move(fyne.NewPos(x, y))
			s.Resize(fyne.NewSize(meterBarW, segH))
		}
	}
}

func (r *meterRenderer) MinSize() fyne.Size { return r.m.MinSize() }

func (r *meterRenderer) Refresh() {
	r.m.litL, r.m.litR = r.m.litSegments(r.m.left), r.m.litSegments(r.m.right)
	for ch, level := range []float64{r.m.left, r.m.right} {
		for i := range meterSegments {
			s := r.segs[ch*meterSegments+i]
			db := r.m.segmentDB(i)
			var want color.Color = colRaised
			if r.m.enabled && level >= db {
				want = zoneColor(db)
			}
			if s.FillColor != want {
				s.FillColor = want
				s.Refresh()
			}
		}
	}
}

func (r *meterRenderer) Objects() []fyne.CanvasObject {
	objs := make([]fyne.CanvasObject, len(r.segs))
	for i, s := range r.segs {
		objs[i] = s
	}
	return objs
}

func (r *meterRenderer) Destroy() {
	if r.m.anim != nil {
		r.m.anim.Stop()
	}
}

// meterScaleWidget labels the meter in dB, aligned with its segments.
type meterScaleWidget struct {
	widget.BaseWidget
	floor float64
}

func newMeterScale() *meterScaleWidget {
	s := &meterScaleWidget{floor: domain.DefaultMeterFloorDB}
	s.ExtendBaseWidget(s)
	return s
}

// SetFloor relabels the scale for a meter range.
func (s *meterScaleWidget) SetFloor(db float64) {
	if db != s.floor {
		s.floor = db
		s.Refresh()
	}
}

func (s *meterScaleWidget) CreateRenderer() fyne.WidgetRenderer {
	r := &scaleRenderer{s: s}
	r.Refresh()
	return r
}

type scaleRenderer struct {
	s      *meterScaleWidget
	floor  float64 // range the labels were made for
	labels []*canvas.Text
	size   fyne.Size
}

func (r *scaleRenderer) Layout(size fyne.Size) {
	r.size = size
	for i, db := range meterScale(r.floor) {
		t := r.labels[i]
		ts := t.MinSize()
		frac := float32((db - r.floor) / (meterMaxDB - r.floor))
		y := size.Height*(1-frac) - ts.Height/2
		y = min(max(y, 0), size.Height-ts.Height)
		t.Move(fyne.NewPos(0, y))
		t.Resize(fyne.NewSize(size.Width, ts.Height))
	}
}

func (r *scaleRenderer) MinSize() fyne.Size { return fyne.NewSize(18, 120) }

func (r *scaleRenderer) Refresh() {
	if r.labels != nil && r.floor == r.s.floor {
		return
	}
	r.floor = r.s.floor
	r.labels = nil
	for _, db := range meterScale(r.floor) {
		t := canvas.NewText(fmt.Sprintf("%.0f", db), colTextDim)
		t.TextSize = 9
		t.Alignment = fyne.TextAlignTrailing
		r.labels = append(r.labels, t)
	}
	r.Layout(r.size)
	canvas.Refresh(r.s)
}
func (r *scaleRenderer) Destroy() {}

func (r *scaleRenderer) Objects() []fyne.CanvasObject {
	objs := make([]fyne.CanvasObject, len(r.labels))
	for i, t := range r.labels {
		objs[i] = t
	}
	return objs
}
