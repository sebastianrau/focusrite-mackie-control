package focusritexml

import "encoding/xml"

// Set repräsentiert die oberste Ebene des XML mit dem Attribut "devid".
type Set struct {
	XMLName xml.Name `xml:"set"`
	DevID   int      `xml:"devid,attr"`
	Items   []Item   `xml:"item"`
}

// Item repräsentiert ein Element innerhalb des Sets mit den Attributen "id" und "value".
type Item struct {
	ID    int    `xml:"id,attr"`
	Value string `xml:"value,attr"`
}
