package protocol

import "github.com/ECUST-XX/xml"

// <device-removal id="4"/>
type DeviceRemoval struct {
	XMLName xml.Name `xml:"device-removal"`

	ID int `xml:"id,attr,omitempty"`
}
