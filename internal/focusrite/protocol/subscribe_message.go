package protocol

import "github.com/ECUST-XX/xml"

type SubscribeMessage struct {
	XMLName xml.Name `xml:"device-subscribe"`

	DeviceID  int  `xml:"devid,attr,omitempty"`
	Subscribe bool `xml:"subscribe,attr,omitempty"`
}
