package protocol

import "github.com/ECUST-XX/xml"

type MixInput struct {
	XMLName xml.Name     `xml:"input"`
	Gain    ElementFloat `xml:"gain"`
	Pan     ElementFloat `xml:"pan"`
	Mute    ElementBool  `xml:"mute"`
	Solo    ElementBool  `xml:"solo"`
}
