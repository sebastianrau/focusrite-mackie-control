package protocol

import "github.com/ECUST-XX/xml"

type KeepAlive struct {
	XMLName xml.Name `xml:"keep-alive"`
}
