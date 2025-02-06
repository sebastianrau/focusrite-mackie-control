package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// decay per 50ms should be 0,59db/50ms

// VerticalBar ist ein Widget, das eine vertikale grüne Leiste rendert.
type VerticalBar struct {
	widget.BaseWidget
	padding       float32 // Padding in Pixeln
	colorGradient *Gradient
	value         float64
	minValue      float64
	maxValue      float64

	scaledValue float64
}

//second := time.Second
//fmt.Print(int64(second/time.Millisecond)) // prints 1000

// NewVerticalBar erstellt eine neue VerticalBar
func NewVerticalBar(padding float32, minValue float64, maxValue float64) *VerticalBar {
	b := &VerticalBar{padding: padding,
		minValue: minValue,
		maxValue: maxValue,
		value:    maxValue, //TOOD set to min
	}

	b.ExtendBaseWidget(b)
	return b
}

func (b *VerticalBar) SetValue(v float64) {
	if v < b.minValue {
		v = b.minValue
	} else if v > b.maxValue {
		v = b.maxValue
	}
	b.value = v
	b.Refresh()
}

// Rendere

type verticalBarRenderer struct {
	bar    *VerticalBar
	rect   *canvas.Rectangle
	border *canvas.Rectangle
}

func (b *VerticalBar) SetGradient(grad *Gradient) {
	b.colorGradient = grad
}

// CreateRenderer erstellt den Renderer für das Widget
func (b *VerticalBar) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.RGBA{0, 50, 0, 255})
	border := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	return &verticalBarRenderer{bar: b, rect: rect, border: border}
}

func (r *verticalBarRenderer) Layout(size fyne.Size) {
	scaledValue := (r.bar.value - r.bar.minValue) / (r.bar.maxValue - r.bar.minValue)
	padding := r.bar.padding
	height := (size.Height - 2*padding) * float32(scaledValue)
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

	if r.bar.colorGradient != nil {
		c := r.bar.colorGradient.GetColor(r.bar.value)
		r.rect.FillColor = c
	}
	canvas.Refresh(r.rect)
	canvas.Refresh(r.border)
}

func (r *verticalBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.border, r.rect}
}

func (r *verticalBarRenderer) Destroy() {}
