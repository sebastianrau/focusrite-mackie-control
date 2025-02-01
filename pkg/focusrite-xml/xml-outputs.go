package focusritexml

import "github.com/ECUST-XX/xml"

type Outputs struct {
	XMLName xml.Name `xml:"outputs"`

	Analogues []Analogue `xml:"analogue"`
	Loopbacks []Loopback `xml:"loopback"`
	SpdifRca  []SpdifRca `xml:"spdif-rca"`
	Adat      []Adat     `xml:"adat"`
}
