package focusritexml

import "github.com/ECUST-XX/xml"

// ClientDetails repräsentiert die Informationen zu einem verbundenen Client.
// <client-details id="10875864856933266113" />
type ClientDetails struct {
	XMLName xml.Name `xml:"client-details"`

	Hostname  string `xml:"hostname,attr,omitempty"`
	ClientKey string `xml:"client-key,attr,omitempty"`
	Id        string `xml:"id,attr,omitempty"`
}
