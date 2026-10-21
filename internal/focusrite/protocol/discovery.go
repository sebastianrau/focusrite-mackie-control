package protocol

import "github.com/ECUST-XX/xml"

// DiscoveryRequest is sent by UDP to find the Focusrite Control TCP port.
//
//	<client-discovery app="SAFFIRE-CONTROL" version="4" device="iOS"/>
type DiscoveryRequest struct {
	XMLName xml.Name `xml:"client-discovery"`
	App     string   `xml:"app,attr,omitempty"`
	Version string   `xml:"version,attr,omitempty"`
	Device  string   `xml:"device,attr,omitempty"`
}

// ServerAnnouncement answers a DiscoveryRequest.
//
//	<server-announcement app='SAFFIRE-CONTROL' version='4' hostname='host.local' port='55145'/>
type ServerAnnouncement struct {
	XMLName  xml.Name `xml:"server-announcement"`
	App      string   `xml:"app,attr,omitempty"`
	Version  string   `xml:"version,attr,omitempty"`
	Hostname string   `xml:"hostname,attr,omitempty"`
	Port     int      `xml:"port,attr,omitempty"`
}
