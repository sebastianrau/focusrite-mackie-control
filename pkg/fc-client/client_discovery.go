package focusriteclient

import (
	"encoding/xml"
	"fmt"
	"net"
	"time"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-xml"
)

// port 30096 - 30098
// Length=000042 <client-discovery app="SAFFIRE-CONTROL" version="4" device="iOS"/>

type DiscoveryRequest struct {
	XMLName xml.Name `xml:"client-discovery"`
	App     string   `xml:"app,attr,omitempty"`
	Version string   `xml:"version,attr,omitempty"`
	Device  string   `xml:"device,attr,omitempty"`
}

// <server-announcement app='SAFFIRE-CONTROL' version='4' hostname='MacBook-Pro-von-Sebastian.local' port='55145'/>
type ServerAnnouncement struct {
	XMLName  xml.Name `xml:"server-announcement"`
	App      string   `xml:"app,attr,omitempty"`
	Version  string   `xml:"version,attr,omitempty"`
	Hostname string   `xml:"hostname,attr,omitempty"`
	Port     int      `xml:"port,attr,omitempty"`
}

var (
	ports []int = []int{
		30096,
		30097,
		30098,
	}

	dc DiscoveryRequest = DiscoveryRequest{
		App:     "SAFFIRE-CONTROL",
		Version: "4",
		Device:  "iOS",
	}
)

func DiscoverServer() (int, error) {
	for _, p := range ports {
		port, err := discoverClient(p)
		if err == nil {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no server found")
}

func discoverClient(port int) (int, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", "127.0.0.1", port))
	if err != nil {
		return 0, err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	msg, err := focusritexml.ParseToXML(dc)
	if err != nil {
		return 0, err
	}
	_, err = conn.Write([]byte(msg))
	if err != nil {
		return 0, err
	}

	buffer := make([]byte, 1024)
	deadline := time.Now().Add(5 * time.Second)
	err = conn.SetReadDeadline(deadline)
	if err != nil {
		return 0, err
	}

	_, err = conn.Read(buffer)
	if err != nil {
		return 0, err
	}

	xmlData, err := focusritexml.SplitLenXML(string(buffer))
	if err != nil {
		return 0, err
	}

	var announcement ServerAnnouncement
	err = xml.Unmarshal([]byte(xmlData), &announcement)
	if err != nil {
		return 0, err
	}

	return announcement.Port, nil
}
