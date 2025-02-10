package focusritexml

import "github.com/ECUST-XX/xml"

type Inputs struct {
	XMLName xml.Name `xml:"inputs"`

	Analogues []Analogue `xml:"analogue"`
	Playbacks []Playback `xml:"playback"`
	SpdifRca  []SpdifRca `xml:"spdif-rca"`
	Adat      []Adat     `xml:"adat"`
}
