package gui

import "math"

func linearToLog(value, min, max float64) float64 {
	minLin := math.Pow(10, min/20) // dB in linearen Wert umwandeln
	maxLin := math.Pow(10, max/20)
	valLin := minLin + value*(maxLin-minLin)

	return 20 * math.Log10(valLin) // Umgekehrt in dB-Wert umrechnen
}

// Wandelt logarithmischen dB-Wert in linearen Slider-Wert um
func logToLinear(value, min, max float64) float64 {
	minLin := math.Pow(10, min/20)
	maxLin := math.Pow(10, max/20)
	valLin := math.Pow(10, value/20)

	return (valLin - minLin) / (maxLin - minLin) // Normalisiert zwischen 0 und 1
}

// lineare Skalierungsfunktion
func normalise(value, min, max float64) float64 {
	// Verhindere, dass der Wert außerhalb des gültigen Bereichs liegt
	if value < min {
		return 0
	}
	if value > max {
		return 1
	}

	// Normalisierung des Werts zwischen 0 und 1
	return (value - min) / (max - min)
}

func deNormalise(normalizedValue, min, max float64) float64 {
	// Berechne den originalen Wert basierend auf dem normalisierten Wert
	return min + normalizedValue*(max-min)
}
