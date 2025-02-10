package focusritexml

import "github.com/ECUST-XX/xml"

type Mixer struct {
	XMLName xml.Name `xml:"mixer"`

	Available ElementBool `xml:"available"`

	Inputs MixerInputs `xml:"inputs"`
	Mixes  []Mix       `xml:"mixes>mix"`
}
