package focusritexml

import "github.com/ECUST-XX/xml"

type MixerInputs struct {
	XMLName              xml.Name      `xml:"inputs"`
	AddInput             ElementString `xml:"add-input"`
	AddInputWithoutReset ElementString `xml:"add-input-without-reset"`
	AddStereoInput       ElementString `xml:"add-stereo-input"`
	RemoveInput          ElementString `xml:"remove-input"`
	FreeInputs           ElementString `xml:"free-inputs"`
	InputList            []Input       `xml:"input"`
}
