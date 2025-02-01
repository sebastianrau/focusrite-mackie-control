package focusritexml

import "github.com/ECUST-XX/xml"

type HaloSettings struct {
	XMLName          xml.Name        `xml:"halo-settings"`
	AvailableColours []ElementString `xml:"available-colours>enum"`
	GoodMeterColour  ElementString   `xml:"good-meter-colour"`
	PreClipColour    ElementString   `xml:"pre-clip-meter-colour"`
	ClippingColour   ElementString   `xml:"clipping-meter-colour"`
	EnablePreview    ElementBool     `xml:"enable-preview-mode"`
	Halos            ElementString   `xml:"halos"`
}
