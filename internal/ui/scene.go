package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// sceneButton recalls a scene and shows what it holds: the volume, Mute and
// Dim, and below them the speakers, each letter at a fixed place and green
// when the speaker is on.
type sceneButton struct {
	widget.BaseWidget
	tapped  func()
	enabled bool

	bg      *canvas.Rectangle
	key     *canvas.Text
	volume  *canvas.Text
	mute    *canvas.Text
	dim     *canvas.Text
	letters [domain.NumSpeakers]*canvas.Text
}

func newSceneButton(id domain.SceneID, tapped func()) *sceneButton {
	small := func(s string, c color.Color) *canvas.Text {
		t := canvas.NewText(s, c)
		t.TextSize = 9
		t.TextStyle.Bold = true
		t.Alignment = fyne.TextAlignCenter
		return t
	}
	b := &sceneButton{
		tapped:  tapped,
		enabled: true,
		bg:      canvas.NewRectangle(colPanel),
		key:     canvas.NewText(string(rune('A'+id)), colTextDim),
		volume:  canvas.NewText("", colTextIdle),
		mute:    small("M", colRedText),
		dim:     small("D", colYellow),
	}
	b.bg.CornerRadius = 10
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
	b.enabled = active
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
	if b.enabled && b.tapped != nil {
		b.tapped()
	}
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
	objs := []fyne.CanvasObject{r.b.bg, r.b.key, r.b.volume, r.b.mute, r.b.dim}
	for _, l := range r.b.letters {
		objs = append(objs, l)
	}
	return objs
}

func (r *sceneRenderer) Destroy() {}
