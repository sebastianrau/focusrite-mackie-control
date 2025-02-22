package guiconfig

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"gopkg.in/yaml.v2"
)

type ConfigApp struct {
	oldConfig config.Config
	newConfig config.Config

	Content *fyne.Container
}

func NewConfigApp(app fyne.App, cfg *config.Config, exit func()) *ConfigApp {
	c := &ConfigApp{
		oldConfig: *cfg,
		newConfig: *cfg,
	}

	// Config Gui Parts
	controllerConfig := NewControllerConfig(c.newConfig.MonitorController)
	midiConfig := NewMidiConfigGui(c.newConfig.Midi)
	focusriteConfig := NewFocusriteConfigGui(c.newConfig.FocusriteDevice)

	// Save
	saveButton := widget.NewButton("Save Config", func() {
		out, _ := yaml.Marshal(c.newConfig)
		fmt.Print(string(out))
		c.newConfig.Save()
		fmt.Printf("Saved Config: %+v\n", c.newConfig)
	})

	exitButton := widget.NewButton("Exit", func() {
		if exit != nil {
			exit()
		}
	})

	//layout
	c.Content = container.NewBorder(
		nil,
		container.NewGridWithColumns(3, saveButton, layout.NewSpacer(), exitButton),
		nil,
		nil,
		container.NewVScroll(
			widget.NewAccordion(
				controllerConfig.Container,
				midiConfig.Container,
				focusriteConfig.Container,
			),
		),
	)

	return c
}
