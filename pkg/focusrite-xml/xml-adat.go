package focusritexml

import "github.com/ECUST-XX/xml"

type Adat struct {
	XMLName xml.Name `xml:"adat"`

	ID               int    `xml:"id,attr"`
	SupportsTalkback bool   `xml:"supports-talkback,attr"`
	Hidden           bool   `xml:"hidden,attr"`
	Name             string `xml:"name,attr"`
	StereoName       string `xml:"stereo-name,attr"`
	Port             int    `xml:"port,attr"`

	Available ElementBool   `xml:"available"`
	Meter     ElementInt    `xml:"meter"`
	Nickname  ElementString `xml:"nickname"`
	Mute      ElementBool   `xml:"mute"`
	Stereo    ElementBool   `xml:"stereo"`
	Source    ElementInt    `xml:"source"`
}
