package main

import (
	"fmt"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite"
)

func main() {
	// Neue Konfiguration erstellen
	fc := focusrite.NewFocusriteControl("Scarlett 18i20", "1234")

	// Eingänge und Ausgänge hinzufügen
	fc.AddInput(1, 30.0, false, false, 0.5)
	fc.AddOutput(1, -6.0, false)

	// Routing hinzufügen
	fc.AddRouting(1, 1)

	// XML generieren
	xmlOutput, err := fc.ToXML()
	if err != nil {
		fmt.Printf("Fehler: %v\n", err)
		return
	}

	fmt.Println(xmlOutput)
}
