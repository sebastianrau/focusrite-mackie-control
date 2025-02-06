package gui

import (
	"image/color"
	"sort"
)

// ColorValuePair represents a pair of a value and an associated color
type ColorValuePair struct {
	Value float64
	Color color.Color
}

// Gradient struct to hold the color-value pairs
type Gradient struct {
	ColorValuePairs []ColorValuePair
}

// NewGradient creates a new Gradient with a list of color-value pairs
func NewGradient(pairs []ColorValuePair) *Gradient {

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value < pairs[j].Value
	})

	return &Gradient{
		ColorValuePairs: pairs,
	}
}

/* Version 1
// GetColor calculates the interpolated color based on the value
func (g *Gradient) GetColor(value float64) color.Color {
	// If the value is below the first entry, return the first color
	if value <= g.ColorValuePairs[0].Value {
		return g.ColorValuePairs[0].Color
	}
	// If the value is above the last entry, return the last color
	if value >= g.ColorValuePairs[len(g.ColorValuePairs)-1].Value {
		return g.ColorValuePairs[len(g.ColorValuePairs)-1].Color
	}

	// Loop through the pairs and find the appropriate range
	for i := 0; i < len(g.ColorValuePairs)-1; i++ {
		// Find two adjacent values for interpolation
		low := g.ColorValuePairs[i]
		high := g.ColorValuePairs[i+1]
		if value >= low.Value && value <= high.Value {
			// Interpolate between low and high color
			normalized := (value - low.Value) / (high.Value - low.Value)
			lowR, lowG, lowB, _ := low.Color.RGBA()
			highR, highG, highB, _ := high.Color.RGBA()

			// Calculate the interpolated RGB values

			r := uint8(float64(lowR) + (normalized * float64(highR-lowR)))
			gVal := uint8(float64(lowG) + (normalized * float64(highG-lowG)))
			b := uint8(float64(lowB) + (normalized * float64(highB-lowB)))

			return color.RGBA{r, gVal, b, 255}
		}
	}
	// Default return black if something goes wrong (shouldn't happen)
	return color.Black
}
*/

// GetColor calculates the interpolated color based on the value
func (g *Gradient) GetColor(value float64) color.Color {
	// If the value is below the first entry, return the first color
	if value <= g.ColorValuePairs[0].Value {
		return g.ColorValuePairs[0].Color
	}
	// If the value is above the last entry, return the last color
	if value >= g.ColorValuePairs[len(g.ColorValuePairs)-1].Value {
		return g.ColorValuePairs[len(g.ColorValuePairs)-1].Color
	}

	// Loop through the pairs and find the appropriate range
	for i := 0; i < len(g.ColorValuePairs)-1; i++ {
		low := g.ColorValuePairs[i]
		high := g.ColorValuePairs[i+1]

		if value >= low.Value && value <= high.Value {
			// Normalized interpolation factor
			normalized := (value - low.Value) / (high.Value - low.Value)

			// Convert to 0-255 range before interpolation
			lowR, lowG, lowB, _ := low.Color.RGBA()
			highR, highG, highB, _ := high.Color.RGBA()

			// Scale down to 0-255
			lowR8 := float64(lowR) / 257.0
			lowG8 := float64(lowG) / 257.0
			lowB8 := float64(lowB) / 257.0
			highR8 := float64(highR) / 257.0
			highG8 := float64(highG) / 257.0
			highB8 := float64(highB) / 257.0

			// Interpolate correctly in the 0-255 range
			r := uint8(lowR8 + normalized*(highR8-lowR8))
			gVal := uint8(lowG8 + normalized*(highG8-lowG8))
			b := uint8(lowB8 + normalized*(highB8-lowB8))

			return color.RGBA{r, gVal, b, 255}
		}
	}
	return color.Black // Fallback (sollte nie passieren)
}
