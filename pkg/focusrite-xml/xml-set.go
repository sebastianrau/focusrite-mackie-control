package focusritexml

import "encoding/xml"

type Set struct {
	XMLName xml.Name `xml:"set"`
	DevID   int      `xml:"devid,attr"`
	Items   []Item   `xml:"item"`
}

type Item struct {
	ID    int    `xml:"id,attr"`
	Value string `xml:"value,attr"`
}
