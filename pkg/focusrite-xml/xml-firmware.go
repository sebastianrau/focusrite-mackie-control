package focusritexml

import "github.com/ECUST-XX/xml"

type Firmware struct {
	XMLName xml.Name `xml:"firmware"`

	Version          ElementString `xml:"version"`
	NeedsUpdate      ElementBool   `xml:"needs-update"`
	FirmwareProgress ElementString `xml:"firmware-progress"`
	UpdateFirmware   ElementString `xml:"update-firmware"`
	RestoreFactory   ElementString `xml:"restore-factory"`
}
