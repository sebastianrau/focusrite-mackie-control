package focusritexml

import "github.com/ECUST-XX/xml"

// <approval hostname="Monitor Controller" id="10875864856933266113" type="response" authorised="true"/>

type Approval struct {
	XMLName    xml.Name `xml:"approval"`
	Hostname   string   `xml:"hostname,attr,omitempty"`
	Id         string   `xml:"id,attr,omitempty"`
	Type       string   `xml:"type,attr,omitempty"`
	Authorised bool     `xml:"authorised,attr,omitempty"`
}
