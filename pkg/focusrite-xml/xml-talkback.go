package focusritexml

import "github.com/ECUST-XX/xml"

type Talkback struct {
	XMLName           xml.Name    `xml:"talkback"`
	InputSource       ElementInt  `xml:"talkback-input-source"`
	SourceAttenuation ElementInt  `xml:"source-attenuation"`
	Available         ElementBool `xml:"talkback-available"`
}
