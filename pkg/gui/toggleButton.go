package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ButtonID int

type ButtonEvent struct {
	Label  string
	Button *ToggleButton
}

type ToggleButton struct {
	widget.BaseWidget
	ID          ButtonID
	state       bool
	Button      *widget.Button
	LabelWidget *canvas.Text

	OnColor       color.Color
	OffColor      color.Color
	DisabledColor color.Color

	ButtonPressed chan ButtonEvent
}

func NewToggleButton(id ButtonID, label string, onColor color.Color, eventChan chan ButtonEvent) *ToggleButton {

	toggleBtn := &ToggleButton{
		ID:            id,
		state:         false,
		OnColor:       onColor,
		OffColor:      theme.Color(theme.ColorNameForeground),
		DisabledColor: theme.Color(theme.ColorNameDisabled),
		//label:         label,
		ButtonPressed: eventChan,
	}

	btn := widget.NewButton("", nil)

	//btn.Importance = widget.HighImportance
	btn.OnTapped = func() {
		eventChan <- ButtonEvent{Label: label, Button: toggleBtn}
	}

	toggleBtn.LabelWidget = canvas.NewText(label, toggleBtn.OffColor)
	toggleBtn.LabelWidget.Alignment = fyne.TextAlignCenter
	toggleBtn.Button = btn

	toggleBtn.ExtendBaseWidget(toggleBtn)
	return toggleBtn
}

func (tb *ToggleButton) Set(state bool) {
	tb.state = state

	if tb.Button.Disabled() {
		tb.LabelWidget.TextStyle = fyne.TextStyle{Bold: true}
		tb.LabelWidget.Color = tb.DisabledColor
	} else {
		if tb.state {
			tb.LabelWidget.TextStyle = fyne.TextStyle{Bold: true}
			tb.LabelWidget.Color = tb.OnColor
		} else {
			tb.LabelWidget.TextStyle = fyne.TextStyle{Bold: false}
			tb.LabelWidget.Color = tb.OffColor
		}
	}

	tb.LabelWidget.Refresh()
}

func (tb *ToggleButton) SetLabel(label string) {
	tb.LabelWidget.Text = label
}

func (tb *ToggleButton) SetHidden(disable bool) {
	if disable {
		//tb.Button.Disable()
		tb.Button.Hide()
		tb.LabelWidget.Hide()

	} else {
		//tb.Button.Enable()
		tb.Button.Show()
		tb.LabelWidget.Show()
	}
}

func (tb *ToggleButton) Enable() {
	tb.Button.Enable()
	tb.Set(tb.state)
}
func (tb *ToggleButton) Disable() {
	tb.Button.Disable()
	tb.Set(tb.state)
}

func (tb *ToggleButton) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(tb.Button, tb.LabelWidget)
	return widget.NewSimpleRenderer(c)
}
