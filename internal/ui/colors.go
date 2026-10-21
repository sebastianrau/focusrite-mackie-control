package ui

import "image/color"

// Palette of the channel strip design.
var (
	colBackground   = hex(0x121417)
	colPanel        = hex(0x1b1e22) // channel strip, idle speaker rows
	colRaised       = hex(0x262a30) // idle Mute/Dim, unlit meter segments
	colLine         = hex(0x1f2327)
	colText         = hex(0xe8eaed)
	colTextIdle     = hex(0xc4c8cd)
	colTextDim      = hex(0x8b9198) // headings, scale, keys
	colTextDisabled = hex(0x5c636b)
	colGreen        = hex(0x3ddc84)
	colGreenBg      = hex(0x1d3326)
	colRed          = hex(0xe5484d)
	colRedText      = hex(0xff8a8d)
	colYellow       = hex(0xffe14d)
	colOrange       = hex(0xff9f43)
	colDark         = hex(0x1a1a1a) // text on yellow
)

func hex(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

// zoneColor is the meter colour for a level: green below -15 dB, yellow
// up to -6 dB, orange above.
func zoneColor(db float64) color.Color {
	switch {
	case db > -6:
		return colOrange
	case db > -15:
		return colYellow
	default:
		return colGreen
	}
}
