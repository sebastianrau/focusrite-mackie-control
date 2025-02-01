package focusritexml

import "github.com/ECUST-XX/xml"

type MixInput struct {
	XMLName xml.Name    `xml:"input"`
	Gain    ElementInt  `xml:"gain"`
	Pan     ElementInt  `xml:"pan"`
	Mute    ElementBool `xml:"mute"`
	Solo    ElementBool `xml:"solo"`
}
