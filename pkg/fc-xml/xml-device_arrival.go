package focusritexml

import "encoding/xml"

// DeviceArrival represents the root element.
type DeviceArrival struct {
	XMLName xml.Name `xml:"device-arrival"`

	Device Device `xml:"device"`
}
