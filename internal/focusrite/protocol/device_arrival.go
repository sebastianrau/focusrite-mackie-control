package protocol

import "github.com/ECUST-XX/xml"

// DeviceArrival announces an interface with its complete element tree.
type DeviceArrival struct {
	XMLName xml.Name `xml:"device-arrival"`

	Device Device `xml:"device"`
}
