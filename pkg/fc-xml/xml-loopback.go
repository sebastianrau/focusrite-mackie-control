package focusritexml

import "github.com/ECUST-XX/xml"

type Loopback struct {
	XMLName xml.Name `xml:"loopback"`

	Name       string        `xml:"name,attr"`
	StereoName string        `xml:"stereo-name,attr"`
	Available  ElementString `xml:"available"`
	Meter      ElementFloat  `xml:"meter"`
	AssignMix  ElementString `xml:"assign-mix"`
	AssignTBM  ElementString `xml:"assign-talkback-mix"`
	Mute       ElementBool   `xml:"mute"`
	Source     ElementString `xml:"source"`
	Stereo     ElementBool   `xml:"stereo"`
	Nickname   ElementString `xml:"nickname"`
}
