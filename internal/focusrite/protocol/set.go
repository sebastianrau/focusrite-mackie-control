package protocol

import (
	"strconv"

	"github.com/ECUST-XX/xml"
)

// Set carries element values: from the server when something changed, and to
// the server to change them.
type Set struct {
	XMLName xml.Name `xml:"set"`
	DevID   int      `xml:"devid,attr"`
	Items   []Item   `xml:"item"`
}

type Item struct {
	ID    int    `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

func BoolItem(id int, v bool) Item     { return Item{ID: id, Value: strconv.FormatBool(v)} }
func IntItem(id, v int) Item           { return Item{ID: id, Value: strconv.Itoa(v)} }
func StringItem(id int, v string) Item { return Item{ID: id, Value: v} }
