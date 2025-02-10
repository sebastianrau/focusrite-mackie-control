package focusritexml

import "github.com/ECUST-XX/xml"

type Clocking struct {
	XMLName xml.Name `xml:"clocking"`

	Locked      ElementBool   `xml:"locked"`
	ClockSource ElementString `xml:"clock-source"`
	SampleRate  ElementString `xml:"sample-rate"`
	ClockMaster ElementString `xml:"clock-master"`
}
