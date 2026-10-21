package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// blockButton is a filled, rounded button whose fill shows its state:
// Mute, Dim and the speaker rows.
type blockButton struct {
	widget.BaseWidget
	tapped  func()
	enabled bool

	bg    *canvas.Rectangle
	key   *canvas.Text // optional leading letter
	label *canvas.Text
	align fyne.TextAlign
}

func newBlockButton(text string, align fyne.TextAlign, tapped func()) *blockButton {
	b := &blockButton{
		tapped:  tapped,
		enabled: true,
		bg:      canvas.NewRectangle(colRaised),
		label:   canvas.NewText(text, colTextIdle),
		align:   align,
	}
	b.bg.CornerRadius = 10
	b.label.TextSize = 14
	b.label.Alignment = align
	b.ExtendBaseWidget(b)
	return b
}

// withKey adds a small leading letter, as on the speaker rows.
func (b *blockButton) withKey(k string) *blockButton {
	b.key = canvas.NewText(k, colTextDim)
	b.key.TextSize = 12
	return b
}

// style sets fill, outline and text in one go. It redraws only on a
// change: render runs for every meter update.
func (b *blockButton) style(fill, stroke, text color.Color, bold bool) {
	var width float32
	if stroke != nil {
		width = 1.5
	}
	if b.bg.FillColor == fill && b.bg.StrokeColor == stroke && b.bg.StrokeWidth == width &&
		b.label.Color == text && b.label.TextStyle.Bold == bold {
		return
	}
	b.bg.FillColor, b.bg.StrokeColor, b.bg.StrokeWidth = fill, stroke, width
	b.label.Color = text
	b.label.TextStyle.Bold = bold
	b.Refresh()
}

func (b *blockButton) setText(s string) {
	if b.label.Text != s {
		b.label.Text = s
		b.Refresh()
	}
}

func (b *blockButton) Text() string { return b.label.Text }

func (b *blockButton) setEnabled(on bool) { b.enabled = on }

func (b *blockButton) Tapped(*fyne.PointEvent) {
	if b.enabled && b.tapped != nil {
		b.tapped()
	}
}

func (b *blockButton) Cursor() desktop.Cursor {
	if b.enabled {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (b *blockButton) MinSize() fyne.Size { return fyne.NewSize(60, 44) }

func (b *blockButton) CreateRenderer() fyne.WidgetRenderer {
	return &blockRenderer{b: b}
}

type blockRenderer struct{ b *blockButton }

func (r *blockRenderer) Layout(size fyne.Size) {
	b := r.b
	b.bg.Resize(size)
	x := float32(10)
	if b.key != nil {
		ks := b.key.MinSize()
		b.key.Move(fyne.NewPos(x, (size.Height-ks.Height)/2))
		b.key.Resize(ks)
		x += 20
	}
	ls := b.label.MinSize()
	if b.align == fyne.TextAlignCenter {
		b.label.Move(fyne.NewPos(0, (size.Height-ls.Height)/2))
		b.label.Resize(fyne.NewSize(size.Width, ls.Height))
		return
	}
	b.label.Move(fyne.NewPos(x, (size.Height-ls.Height)/2))
	b.label.Resize(fyne.NewSize(size.Width-x-10, ls.Height))
}

func (r *blockRenderer) MinSize() fyne.Size { return r.b.MinSize() }

func (r *blockRenderer) Refresh() {
	for _, o := range r.Objects() {
		o.Refresh()
	}
}

func (r *blockRenderer) Objects() []fyne.CanvasObject {
	if r.b.key != nil {
		return []fyne.CanvasObject{r.b.bg, r.b.key, r.b.label}
	}
	return []fyne.CanvasObject{r.b.bg, r.b.label}
}

func (r *blockRenderer) Destroy() {}
