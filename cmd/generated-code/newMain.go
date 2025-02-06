package main

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gui"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Gradient Lookup Table Example")

	// Create a Gradient with specific color-value pairs
	gradient := gui.NewGradient([]gui.ColorValuePair{
		{Value: 0, Color: color.RGBA{255, 0, 0, 255}},   // Red at value 0
		{Value: 50, Color: color.RGBA{0, 255, 0, 255}},  // Green at value 50
		{Value: 100, Color: color.RGBA{0, 0, 255, 255}}, // Blue at value 100
	})

	// Create a widget to display the interpolated color for a value
	value := 75.0

	colorDisplay := canvas.NewText(fmt.Sprintf("Color for value %.2f", value), color.White)
	colorDisplay.Resize(fyne.NewSize(200, 50))

	// Set the color display background to the interpolated color

	colorDisplay.Color = gradient.GetColor(value)
	//	colorDisplay.Color()

	// Add the label to the window
	myWindow.SetContent(container.NewVBox(colorDisplay))

	// Show and run the app
	myWindow.ShowAndRun()
}
