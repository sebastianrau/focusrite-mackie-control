package focusrite

import (
	"fmt"

	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
)

func ValidateXMLWithXSD(xmlData string, xsdPath string) error {
	// Initialisiere einen Validator
	xsdvalidate.Init()
	defer xsdvalidate.Cleanup()
	xsdhandler, err := xsdvalidate.NewXsdHandlerUrl("examples/test1_split.xsd", xsdvalidate.ParsErrDefault)

	if err != nil {
		return fmt.Errorf("Fehler beim Laden der XSD: %v", err)
	}
	err = xsdhandler.ValidateMem([]byte(xmlData), xsdvalidate.ValidErrDefault)
	if err != nil {
		return err
	}

	return nil
}

func ValidateXML(xmlData string) {

	xsdPath := "focusrite.xsd" // Pfad zur XSD-Datei

	if err := ValidateXMLWithXSD(xmlData, xsdPath); err != nil {
		fmt.Println("Fehler:", err)
	} else {
		fmt.Println("XML entspricht dem Schema!")
	}
}
