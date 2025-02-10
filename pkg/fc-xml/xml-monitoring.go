package focusritexml

import "github.com/ECUST-XX/xml"

type Monitoring struct {
	XMLName xml.Name `xml:"monitoring"`

	MonitorGroupPairs string `xml:",chardata"`

	HardwareControls HardwareControls `xml:"hardware-controls"`
	Preset           ElementString    `xml:"preset"`
}
