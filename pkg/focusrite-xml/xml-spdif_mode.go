package focusritexml

type SpdifMode struct {
	Name string        `xml:"name,attr"`
	Mode ElementString `xml:"mode"`
}
