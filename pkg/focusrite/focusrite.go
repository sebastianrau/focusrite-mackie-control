package focusrite

import (
	"encoding/xml"
	"fmt"
)

// FocusriteControl repräsentiert die Hauptstruktur des XML-Dokuments
type FocusriteControl struct {
	XMLName       xml.Name      `xml:"FocusriteControl"`
	ClientDetails ClientDetails `xml:"client-details,omitempty"`
	Device        Device        `xml:"Device,omitempty"`
	Mix           Mix           `xml:"Mix,omitempty"`
	Routing       Routing       `xml:"Routing,omitempty"`
	MessageType   string        `xml:"message-type,attr,omitempty"`
}

// ClientDetails repräsentiert die Informationen zu einem verbundenen Client.
type ClientDetails struct {
	Hostname  string `xml:"hostname,attr"`
	ClientKey string `xml:"client-key,attr"`
}

// Device repräsentiert die Geräteinformationen
type Device struct {
	Name            string `xml:"Name"`
	FirmwareVersion string `xml:"FirmwareVersion"`
}

// Mix repräsentiert die Mixing-Einstellungen
type Mix struct {
	Inputs  []Input  `xml:"Input"`
	Outputs []Output `xml:"Output"`
}

// Input repräsentiert einen Eingabekanal
type Input struct {
	Channel int     `xml:"Channel"`
	Gain    float64 `xml:"Gain"`
	Mute    bool    `xml:"Mute"`
	Solo    bool    `xml:"Solo"`
	Pan     float64 `xml:"Pan"`
}

// Output repräsentiert einen Ausgabekanal
type Output struct {
	Channel int     `xml:"Channel"`
	Volume  float64 `xml:"Volume"`
	Mute    bool    `xml:"Mute"`
}

// Routing repräsentiert das Routing zwischen Kanälen
type Routing struct {
	InputToOutputs []InputToOutput `xml:"InputToOutput"`
}

// InputToOutput repräsentiert eine Verknüpfung von Eingangs- zu Ausgangskanälen
type InputToOutput struct {
	InputChannel  int `xml:"InputChannel"`
	OutputChannel int `xml:"OutputChannel"`
}

// NewFocusriteControl erstellt eine neue Standardkonfiguration
func NewFocusriteControl(deviceName, firmwareVersion string) *FocusriteControl {
	return &FocusriteControl{
		Device: Device{
			Name:            deviceName,
			FirmwareVersion: firmwareVersion,
		},
		ClientDetails: ClientDetails{
			Hostname:  "Monitor Controller",
			ClientKey: "12345678",
		},
		Mix: Mix{
			Inputs:  []Input{},
			Outputs: []Output{},
		},
		Routing: Routing{
			InputToOutputs: []InputToOutput{},
		},
	}
}

// AddInput fügt einen neuen Eingangskanal hinzu
func (fc *FocusriteControl) AddInput(channel int, gain float64, mute, solo bool, pan float64) {
	fc.Mix.Inputs = append(fc.Mix.Inputs, Input{
		Channel: channel,
		Gain:    gain,
		Mute:    mute,
		Solo:    solo,
		Pan:     pan,
	})
}

// AddOutput fügt einen neuen Ausgangskanal hinzu
func (fc *FocusriteControl) AddOutput(channel int, volume float64, mute bool) {
	fc.Mix.Outputs = append(fc.Mix.Outputs, Output{
		Channel: channel,
		Volume:  volume,
		Mute:    mute,
	})
}

// AddRouting fügt ein neues Routing hinzu
func (fc *FocusriteControl) AddRouting(inputChannel, outputChannel int) {
	fc.Routing.InputToOutputs = append(fc.Routing.InputToOutputs, InputToOutput{
		InputChannel:  inputChannel,
		OutputChannel: outputChannel,
	})
}

// SetClientDetails fügt die Client-Details hinzu.
func (fc *FocusriteControl) SetClientDetails(hostname, clientKey string) {
	fc.ClientDetails = ClientDetails{
		Hostname:  hostname,
		ClientKey: clientKey,
	}
}

// ToXML serialisiert die Konfiguration in XML
func (fc *FocusriteControl) ToXML() (string, error) {
	output, err := xml.MarshalIndent(fc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("Fehler beim Erstellen der XML: %v", err)
	}
	return string(output), nil
}
