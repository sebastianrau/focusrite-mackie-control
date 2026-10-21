// Package ui is the Fyne desktop app: the main window, the tray menu and the
// settings window. Widgets are only touched on the Fyne main thread; input
// from other goroutines is handed over with fyne.Do.
package ui

import (
	_ "embed"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed logo.png
var logoPNG []byte

var logo = fyne.NewStaticResource("logo.png", logoPNG)

// darkTheme is the dark variant with the channel strip palette.
type darkTheme struct{ fyne.Theme }

func (t darkTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return colBackground
	case theme.ColorNameForeground:
		return colText
	case theme.ColorNameDisabled:
		return colTextDisabled
	case theme.ColorNamePrimary: // slider fill and focus
		return colText
	case theme.ColorNameInputBackground:
		return colRaised
	}
	return t.Theme.Color(n, theme.VariantDark)
}
