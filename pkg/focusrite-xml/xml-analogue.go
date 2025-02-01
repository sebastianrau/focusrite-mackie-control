package focusritexml

import "github.com/ECUST-XX/xml"

type Analogue struct {
	XMLName xml.Name `xml:"analogue"`

	ID               int    `xml:"id,attr"`
	SupportsTalkback string `xml:"supports-talkback,attr"`
	Hidden           string `xml:"hidden,attr"`
	Name             string `xml:"name,attr"`
	StereoName       string `xml:"stereo-name,attr"`

	Available       ElementBool   `xml:"available"`
	Meter           ElementInt    `xml:"meter"`
	Nickname        ElementString `xml:"nickname"`
	Stereo          ElementBool   `xml:"stereo"`
	SourceID        ElementInt    `xml:"source"`
	Mode            ElementString `xml:"mode"`
	Air             ElementString `xml:"air"`
	Pad             ElementString `xml:"pad"`
	Mute            ElementBool   `xml:"mute"`
	Gain            ElementInt    `xml:"gain"`
	HardwareControl ElementString `xml:"hardware-control"`
}
