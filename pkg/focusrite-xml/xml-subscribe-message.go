package focusritexml

import "github.com/ECUST-XX/xml"

type SubscribeMessage struct {
	XMLName   xml.Name `xml:"device-subscribe"`
	DeviceId  int      `xml:"devid,attr,omitempty"`
	Subscribe bool     `xml:"subscribe,attr,omitempty"`
}
