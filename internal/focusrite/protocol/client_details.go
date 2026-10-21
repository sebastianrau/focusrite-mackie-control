package protocol

import "github.com/ECUST-XX/xml"

// ClientDetails identifies this app to Focusrite Control; the server answers with the assigned id.
// <client-details id="10875864856933266113" />
type ClientDetails struct {
	XMLName xml.Name `xml:"client-details"`

	Hostname  string `xml:"hostname,attr,omitempty"`
	ClientKey string `xml:"client-key,attr,omitempty"`
	ID        string `xml:"id,attr,omitempty"`
}
