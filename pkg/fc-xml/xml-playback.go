package focusritexml

import "github.com/ECUST-XX/xml"

type Playback struct {
	XMLName          xml.Name      `xml:"playback"`
	ID               string        `xml:"id,attr"`
	SupportsTalkback string        `xml:"supports-talkback,attr"`
	Hidden           string        `xml:"hidden,attr"`
	Name             string        `xml:"name,attr"`
	StereoName       string        `xml:"stereo-name,attr"`
	Available        ElementBool   `xml:"available"`
	Meter            ElementFloat  `xml:"meter"`
	Nickname         ElementString `xml:"nickname"`
}
