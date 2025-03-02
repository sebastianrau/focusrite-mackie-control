package focusritexml

import "github.com/ECUST-XX/xml"

type Mix struct {
	XMLName    xml.Name     `xml:"mix"`
	ID         string       `xml:"id,attr"`
	Name       string       `xml:"name,attr"`
	StereoName string       `xml:"stereo-name,attr"`
	Talkback   ElementBool  `xml:"talkback"`
	Meter      ElementFloat `xml:"meter"`
	Inputs     []MixInput   `xml:"input"`
}
