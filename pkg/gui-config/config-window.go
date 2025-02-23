package guiconfig

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"gopkg.in/yaml.v2"
)

type ConfigApp struct {
	newConfig *config.Config
	Content   *fyne.Container
}

func NewConfigApp(window fyne.Window, cfg *config.Config, restartFunc func(), exitFunc func()) *ConfigApp {
	c := &ConfigApp{}

	var err error

	c.newConfig, err = cfg.DeepCopy()
	if err != nil {
		log.Errorf("Cant Deep Copy Config %s", err.Error())
	}

	// Config Gui Parts
	controllerConfig := NewControllerConfig(c.newConfig.MonitorController)
	midiConfig := NewMidiConfigGui(c.newConfig.Midi)
	focusriteConfig := NewFocusriteConfigGui(c.newConfig.FocusriteDevice)

	// Save
	saveButton := widget.NewButton("Save & Restart", func() {
		dialog.ShowConfirm(
			"Save Config",
			"Save the config and restart the app?",
			func(response bool) {
				if response {
					out, _ := yaml.Marshal(c.newConfig)
					fmt.Print(string(out))
					c.newConfig.Save()
					fmt.Printf("Saved Config: %+v\n", c.newConfig)
					if restartFunc != nil {
						restartFunc()
					}
				}
			},
			window)
	})

	exitButton := widget.NewButton("Close", func() {
		if exitFunc != nil {
			dialog.ShowConfirm(
				"Close Config",
				"Your Settings will not be saved. Will you Close?",
				func(response bool) {
					if response && exitFunc != nil {
						exitFunc()
					}
				},
				window)
		}
	})

	configAccordion := widget.NewAccordion(
		controllerConfig.Container,
		midiConfig.Container,
		focusriteConfig.Container,
	)
	configAccordion.Open(0)

	//layout
	c.Content = container.NewBorder(
		nil,
		container.NewGridWithColumns(3, saveButton, layout.NewSpacer(), exitButton),
		nil,
		nil,
		container.NewVScroll(configAccordion),
	)

	window.SetContent(c.Content)
	return c
}
