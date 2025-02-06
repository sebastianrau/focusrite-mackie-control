package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// VerticalBar ist ein Widget, das eine vertikale grüne Leiste rendert.
type VerticalBar struct {
	widget.BaseWidget
	value   float64 // Wert zwischen 0 und 1
	padding float32 // Padding in Pixeln
}

// NewVerticalBar erstellt eine neue VerticalBar
func NewVerticalBar(padding float32) *VerticalBar {
	b := &VerticalBar{padding: padding}
	b.ExtendBaseWidget(b)
	return b
}

// SetValue setzt den Wert und aktualisiert die Darstellung
func (b *VerticalBar) SetValue(v float64) {
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	b.value = v
	b.Refresh()
}

// CreateRenderer erstellt den Renderer für das Widget
func (b *VerticalBar) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.RGBA{0, 255, 0, 255})
	border := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	return &verticalBarRenderer{bar: b, rect: rect, border: border}
}

type verticalBarRenderer struct {
	bar    *VerticalBar
	rect   *canvas.Rectangle
	border *canvas.Rectangle
}

func (r *verticalBarRenderer) Layout(size fyne.Size) {
	padding := r.bar.padding
	height := (size.Height - 2*padding) * float32(r.bar.value)
	r.rect.Move(fyne.NewPos(5+padding, size.Height-height-5-padding))
	r.rect.Resize(fyne.NewSize(size.Width-10-2*padding, height))
	r.border.Move(fyne.NewPos(padding, padding))
	r.border.Resize(fyne.NewSize(size.Width-2*padding, size.Height-2*padding))
}

func (r *verticalBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(30+2*r.bar.padding, 60+2*r.bar.padding)
}

func (r *verticalBarRenderer) Refresh() {
	r.Layout(r.bar.Size())
	r.rect.FillColor = color.RGBA{0, 255, 0, 255}
	canvas.Refresh(r.rect)
	canvas.Refresh(r.border)
}

func (r *verticalBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.border, r.rect}
}

func (r *verticalBarRenderer) Destroy() {}
