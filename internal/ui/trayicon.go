package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Menu bar icons: a console fader whose cap shows the state. They are
// monochrome; a themed resource becomes a template image on macOS, which the
// system tints for light and dark menu bars.
// All shapes are filled: theme recolouring changes fills only, so strokes
// would keep their colour on Windows and Linux.
const (
	faderSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">` +
		`<rect x="11.1" y="2.5" width="1.8" height="19" rx="0.9" fill="#000000"/>` + // track
		`<rect x="5" y="5.1" width="2" height="1.8" rx="0.9" fill="#000000"/>` + // scale marks
		`<rect x="5" y="11.1" width="2" height="1.8" rx="0.9" fill="#000000"/>` +
		`<rect x="5" y="17.1" width="2" height="1.8" rx="0.9" fill="#000000"/>`
	capNormal = `<rect x="8" y="6" width="8" height="4.5" rx="1" fill="#000000"/>`
	capDim    = `<rect x="8" y="13" width="8" height="4.5" rx="1" fill="#000000"/>`
	capMute   = `<rect x="8" y="16.5" width="8" height="4.5" rx="1" fill="#000000"/>` +
		`<path d="M3.58 2.31L21.58 17.31L20.42 18.69L2.42 3.69Z" fill="#000000"/>` // slash
)

var (
	trayNormal = trayIcon("fader.svg", capNormal)
	trayDim    = trayIcon("fader-dim.svg", capDim)
	trayMute   = trayIcon("fader-mute.svg", capMute)
)

func trayIcon(name, cap string) fyne.Resource {
	return theme.NewThemedResource(fyne.NewStaticResource(name, []byte(faderSVG+cap+`</svg>`)))
}

// trayIconFor picks the icon for the master state; mute wins over dim.
func trayIconFor(mute, dim bool) fyne.Resource {
	switch {
	case mute:
		return trayMute
	case dim:
		return trayDim
	}
	return trayNormal
}
