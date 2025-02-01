package focusritexml

import "github.com/ECUST-XX/xml"

type Input struct {
	XMLName xml.Name `xml:"input"`

	Source ElementString `xml:"source"`
	Stereo ElementString `xml:"stereo"`
}
