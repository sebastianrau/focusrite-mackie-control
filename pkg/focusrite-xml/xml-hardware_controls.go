package focusritexml

import "github.com/ECUST-XX/xml"

type HardwareControls struct {
	XMLName xml.Name `xml:"hardware-controls"`

	Exclusive bool `xml:"exclusive,attr"`
	MinGain   int  `xml:"min-gain,attr"`
	MaxGain   int  `xml:"max-gain,attr"`

	Talkback ElementBool `xml:"talkback"`

	Controls Controls `xml:"hardware-controls"`
}

type Controls struct {
	Gain      ElementInt  `xml:"gain"`
	Dim       ElementInt  `xml:"dim"`
	Mute      ElementBool `xml:"mute"`
	AltEnable ElementBool `xml:"alt-enable"`
	Alt       ElementInt  `xml:"alt"`
}
