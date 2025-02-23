package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	guiconfig "github.com/sebastianrau/focusrite-mackie-control/pkg/gui-config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"

	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

var log *logger.CustomLogger = logger.WithPackage("main-config")

func main() {
	myApp := app.New()

	cfg, err := config.Load()
	if err != nil {
		log.Debugf("can't open config file: %s", err.Error())
		cfg = config.Default()
	}

	myWindow := myApp.NewWindow("Config GUI")
	myWindow.Resize(fyne.NewSize(450, 600))

	myConfigApp := guiconfig.NewConfigApp(myWindow, cfg,
		func() {},
		func() {
			myApp.Quit()
		})

	myWindow.SetContent(myConfigApp.Content)

	myWindow.ShowAndRun()
}
