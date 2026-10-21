package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ToggleButton is a button whose label lights up in its colour while on.
type ToggleButton struct {
	widget.BaseWidget
	on     bool
	button *widget.Button
	label  *canvas.Text

	onColor color.Color
}

func NewToggleButton(label string, onColor color.Color, tapped func()) *ToggleButton {
	t := &ToggleButton{
		button:  widget.NewButton("", tapped),
		label:   canvas.NewText(label, theme.Color(theme.ColorNameForeground)),
		onColor: onColor,
	}
	t.label.Alignment = fyne.TextAlignCenter
	t.ExtendBaseWidget(t)
	return t
}

func (t *ToggleButton) Text() string { return t.label.Text }
func (t *ToggleButton) On() bool     { return t.on }

func (t *ToggleButton) Set(on bool) {
	t.on = on
	t.update()
}

func (t *ToggleButton) SetText(text string) {
	if t.label.Text != text {
		t.label.Text = text
		t.label.Refresh()
	}
}

func (t *ToggleButton) SetEnabled(enabled bool) {
	if enabled == !t.button.Disabled() {
		return
	}
	if enabled {
		t.button.Enable()
	} else {
		t.button.Disable()
	}
	t.update()
}

func (t *ToggleButton) Disabled() bool { return t.button.Disabled() }

func (t *ToggleButton) update() {
	switch {
	case t.button.Disabled():
		t.label.TextStyle = fyne.TextStyle{Bold: true}
		t.label.Color = theme.Color(theme.ColorNameDisabled)
	case t.on:
		t.label.TextStyle = fyne.TextStyle{Bold: true}
		t.label.Color = t.onColor
	default:
		t.label.TextStyle = fyne.TextStyle{}
		t.label.Color = theme.Color(theme.ColorNameForeground)
	}
	t.label.Refresh()
}

func (t *ToggleButton) CreateRenderer() fyne.WidgetRenderer {
	t.update()
	return widget.NewSimpleRenderer(container.NewStack(t.button, t.label))
}
