package focusritexml

import "github.com/ECUST-XX/xml"

type QuickStart struct {
	XMLName xml.Name      `xml:"quick-start"`
	URL     string        `xml:"url,attr"`
	MsdMode ElementString `xml:"msd-mode"`
}
