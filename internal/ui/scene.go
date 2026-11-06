package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/sebastianrau/sweetspot/internal/domain"
)

const (
	// storeHold is how long the inactive scene's button must be held to
	// store the live state in that scene.
	storeHold = 5 * time.Second
	// tapMax is the longest press that still switches the scene. A longer
	// one that is released early does nothing.
	tapMax = 500 * time.Millisecond
)

// sceneButton recalls a scene and shows what it holds: the volume, Mute and
// Dim, and below them the speakers, each letter at a fixed place and green
// when the speaker is on. Holding the inactive scene's button stores the
// live state in it; a bar shows the progress and the button flashes when it
// is done.
type sceneButton struct {
	widget.BaseWidget
	tapped  func()
	stored  func()
	enabled bool
	current bool // the active scene: holding it does nothing

	hold    *fyne.Animation
	pressed time.Time // start of the press, zero without one
	held    bool      // the press stored the scene, so its release is no tap
	bar     *canvas.Rectangle
	flash   *canvas.Rectangle

	bg      *canvas.Rectangle
	key     *canvas.Text
	volume  *canvas.Text
	mute    *canvas.Text
	dim     *canvas.Text
	letters [domain.NumSpeakers]*canvas.Text
}

func newSceneButton(id domain.SceneID, tapped, stored func()) *sceneButton {
	small := func(s string, c color.Color) *canvas.Text {
		t := canvas.NewText(s, c)
		t.TextSize = 9
		t.TextStyle.Bold = true
		t.Alignment = fyne.TextAlignCenter
		return t
	}
	b := &sceneButton{
		tapped:  tapped,
		stored:  stored,
		enabled: true,
		bar:     canvas.NewRectangle(colGreen),
		flash:   canvas.NewRectangle(color.Transparent),
		bg:      canvas.NewRectangle(colPanel),
		key:     canvas.NewText(string(rune('A'+id)), colTextDim),
		volume:  canvas.NewText("", colTextIdle),
		mute:    small("M", colRedText),
		dim:     small("D", colYellow),
	}
	b.bg.CornerRadius = 10
	b.flash.CornerRadius = 10
	b.bar.CornerRadius = 1
	b.bar.Hide()
	b.key.TextSize = 12
	b.volume.TextSize = 14
	for i := range b.letters {
		b.letters[i] = small(sceneLetter(domain.SpeakerID(i)), colTextDisabled)
	}
	b.ExtendBaseWidget(b)
	return b
}

func sceneLetter(id domain.SpeakerID) string {
	if id.IsSub() {
		return "S"
	}
	return string(rune('A' + id))
}

// set shows sc. It redraws only what changed: render runs for every meter
// update.
func (b *sceneButton) set(sc domain.Scene, speakers [domain.NumSpeakers]domain.Speaker, current, active bool) {
	b.enabled, b.current = active, current
	changed := false
	fill, stroke, width := colPanel, color.Color(nil), float32(0)
	if current && active {
		fill, stroke, width = colGreenBg, colGreen, 1.5
	}
	if b.bg.FillColor != fill || b.bg.StrokeColor != stroke {
		b.bg.FillColor, b.bg.StrokeColor, b.bg.StrokeWidth = fill, stroke, width
		b.bg.Refresh()
	}
	vol := colTextIdle
	switch {
	case !active:
		vol = colTextDisabled
	case current:
		vol = colText
	}
	if b.volume.TextStyle.Bold != current {
		b.volume.TextStyle.Bold = current
		changed = true
	}
	changed = setText(b.volume, fmt.Sprint(sc.VolumeDB), vol) || changed
	changed = show(b.mute, sc.Mute) || changed
	changed = show(b.dim, sc.Dim) || changed
	for i, l := range b.letters {
		c := colTextDisabled
		if sc.Selected[i] && active {
			c = colGreen
		}
		setText(l, l.Text, c)
		changed = show(l, !speakers[i].Disabled) || changed
	}
	if changed {
		b.Refresh() // the volume's width may have changed
	}
}

// show shows or hides o and reports whether that changed.
func show(o fyne.CanvasObject, on bool) bool {
	if o.Visible() == on {
		return false
	}
	if on {
		o.Show()
	} else {
		o.Hide()
	}
	return true
}

func (b *sceneButton) Tapped(*fyne.PointEvent) {
	long := !b.pressed.IsZero() && time.Since(b.pressed) > tapMax
	b.pressed = time.Time{}
	if b.held || long {
		b.held = false
		return
	}
	if b.enabled && b.tapped != nil {
		b.tapped()
	}
}

func (b *sceneButton) MouseDown(*desktop.MouseEvent) {
	if !b.enabled {
		return
	}
	b.held = false
	b.pressed = time.Now()
	b.stopHold()
	if b.current {
		return
	}
	b.hold = fyne.NewAnimation(storeHold, func(p float32) {
		if time.Duration(float32(storeHold)*p) < tapMax {
			p = 0 // still a tap: no bar yet
		}
		b.showProgress(p)
		if p >= 1 {
			b.store()
		}
	})
	b.hold.Curve = fyne.AnimationLinear
	b.hold.Start()
}

func (b *sceneButton) MouseUp(*desktop.MouseEvent) { b.stopHold() }

func (b *sceneButton) stopHold() {
	if b.hold != nil {
		b.hold.Stop()
		b.hold = nil
	}
	b.showProgress(0)
}

// store runs when the button was held long enough.
func (b *sceneButton) store() {
	b.held = true
	b.hold = nil
	b.showProgress(0)
	if b.stored != nil {
		b.stored()
	}
	flash := canvas.NewColorRGBAAnimation(color.NRGBA{R: 0x3d, G: 0xdc, B: 0x84, A: 0x90}, color.Transparent,
		400*time.Millisecond, func(c color.Color) {
			b.flash.FillColor = c
			b.flash.Refresh()
		})
	flash.Start()
}

// showProgress draws the hold bar at fraction p of the width; 0 hides it.
func (b *sceneButton) showProgress(p float32) {
	if p <= 0 {
		if b.bar.Visible() {
			b.bar.Hide()
		}
		return
	}
	size := b.Size()
	b.bar.Move(fyne.NewPos(8, size.Height-5))
	b.bar.Resize(fyne.NewSize((size.Width-16)*p, 2))
	b.bar.Show()
}

func (b *sceneButton) Cursor() desktop.Cursor {
	if b.enabled {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (b *sceneButton) MinSize() fyne.Size { return fyne.NewSize(60, 44) }

func (b *sceneButton) CreateRenderer() fyne.WidgetRenderer { return &sceneRenderer{b: b} }

type sceneRenderer struct{ b *sceneButton }

const (
	sceneLetterW   = 7
	sceneLetterGap = 3
)

func (r *sceneRenderer) Layout(size fyne.Size) {
	b := r.b
	b.bg.Resize(size)
	b.flash.Resize(size)
	place := func(o fyne.CanvasObject, x, y float32) {
		s := o.MinSize()
		o.Move(fyne.NewPos(x, y))
		o.Resize(s)
	}
	ks := b.key.MinSize()
	place(b.key, 8, (size.Height-ks.Height)/2)

	// Two lines, right-aligned: volume with M and D, then the speakers.
	vs, ls := b.volume.MinSize(), b.letters[0].MinSize()
	top := (size.Height - vs.Height - ls.Height) / 2
	right := size.Width - 8
	place(b.volume, right-vs.Width, top)
	x := right - vs.Width - 3
	for _, flag := range []*canvas.Text{b.dim, b.mute} {
		x -= sceneLetterW
		flag.Move(fyne.NewPos(x, top+vs.Height-ls.Height-1))
		flag.Resize(fyne.NewSize(sceneLetterW, ls.Height))
		x -= sceneLetterGap
	}
	x = right - float32(len(b.letters))*(sceneLetterW+sceneLetterGap) + sceneLetterGap
	for _, l := range b.letters {
		l.Move(fyne.NewPos(x, top+vs.Height))
		l.Resize(fyne.NewSize(sceneLetterW, ls.Height))
		x += sceneLetterW + sceneLetterGap
	}
}

func (r *sceneRenderer) MinSize() fyne.Size { return r.b.MinSize() }

func (r *sceneRenderer) Refresh() {
	r.Layout(r.b.Size())
	for _, o := range r.Objects() {
		o.Refresh()
	}
}

func (r *sceneRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.b.bg, r.b.flash, r.b.bar, r.b.key, r.b.volume, r.b.mute, r.b.dim}
	for _, l := range r.b.letters {
		objs = append(objs, l)
	}
	return objs
}

func (r *sceneRenderer) Destroy() {}
