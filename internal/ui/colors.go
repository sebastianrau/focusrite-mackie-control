package ui

import (
	"image/color"
	"sort"
)

// Colours of the original UI.
var (
	darkGreen = color.RGBA{0, 102, 0, 225}
	green     = color.RGBA{51, 255, 51, 255}
	yellow    = color.RGBA{255, 255, 51, 255}
	orange    = color.RGBA{255, 153, 51, 255}
	red       = color.RGBA{255, 51, 0, 255}
	grey      = color.RGBA{128, 128, 128, 255}
)

// levelGradient colours the meter bars and the peak label by level.
var levelGradient = newGradient([]colorStop{
	{Value: -127, Color: darkGreen},
	{Value: -15, Color: green},
	{Value: -6, Color: yellow},
	{Value: 0, Color: orange},
})

type colorStop struct {
	Value float64
	Color color.Color
}

// gradient interpolates colours between stops.
type gradient struct{ stops []colorStop }

func newGradient(stops []colorStop) *gradient {
	sort.Slice(stops, func(i, j int) bool { return stops[i].Value < stops[j].Value })
	return &gradient{stops: stops}
}

func (g *gradient) at(value float64) color.Color {
	if value <= g.stops[0].Value {
		return g.stops[0].Color
	}
	last := g.stops[len(g.stops)-1]
	if value >= last.Value {
		return last.Color
	}
	for i := 0; i < len(g.stops)-1; i++ {
		lo, hi := g.stops[i], g.stops[i+1]
		if value >= lo.Value && value <= hi.Value {
			f := (value - lo.Value) / (hi.Value - lo.Value)
			lr, lg, lb, _ := lo.Color.RGBA()
			hr, hg, hb, _ := hi.Color.RGBA()
			mix := func(a, b uint32) uint8 {
				a8, b8 := float64(a)/257, float64(b)/257
				return uint8(a8 + f*(b8-a8))
			}
			return color.RGBA{mix(lr, hr), mix(lg, hg), mix(lb, hb), 255}
		}
	}
	return color.Black
}
