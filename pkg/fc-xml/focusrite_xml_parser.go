package focusritexml

import (
	"fmt"
	"strings"

	"github.com/ECUST-XX/xml"
)

type Wrapper struct {
	XMLName xml.Name
	Content []byte `xml:",innerxml"`
}

func ParseToXML(data interface{}) (string, error) {
	buffer, err := xml.MarshalIndentShortForm(data, "", "")
	if err != nil {
		return "", fmt.Errorf("error encoding data: %v", err)
	}
	xmlString := strings.ReplaceAll(string(buffer), " />", "/>")
	xmlString = fmt.Sprintf("Length=%06X %s", len(xmlString), xmlString)
	return xmlString, nil
}

func SplitLenXML(data string) (string, error) {
	data, found := strings.CutPrefix(data, "Length=")
	if !found {
		return "", fmt.Errorf("no Length found")
	}

	splitted := strings.SplitN(data, " ", 2)
	if len(splitted) != 2 {
		return "", fmt.Errorf("no xml tag found")
	}
	return splitted[1], nil
}

func ParseFromXML(in string) (interface{}, error) {
	lenSplit := strings.SplitN(in, " ", 2) //split len and xml
	if len(lenSplit) != 2 {
		return nil, fmt.Errorf("no length could be found")
	}
	xmlData := strings.TrimSpace(lenSplit[1])

	var wrapper Wrapper
	if err := xml.Unmarshal([]byte(xmlData), &wrapper); err != nil {
		return nil, err
	}

	switch wrapper.XMLName.Local {

	case "client-details":
		var v ClientDetails
		if err := xml.Unmarshal([]byte(xmlData), &v); err != nil {
			return nil, err
		}
		return v, nil

	case "set":
		var v Set
		if err := xml.Unmarshal([]byte(xmlData), &v); err != nil {
			return nil, err
		}
		return v, nil

	case "device-arrival":
		var v DeviceArrival
		if err := xml.Unmarshal([]byte(xmlData), &v); err != nil {
			return nil, err
		}
		return v, nil

	case "device-removal":
		var v DeviceRemoval
		if err := xml.Unmarshal([]byte(xmlData), &v); err != nil {
			return nil, err
		}
		return v, nil

	case "keep-alive":
		var v KeepAlive
		if err := xml.Unmarshal([]byte(xmlData), &v); err != nil {
			return nil, err
		}
		return v, nil

	case "approval":
		var v Approval
		if err := xml.Unmarshal([]byte(xmlData), &v); err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unknown XML type: %s\n%s", wrapper.XMLName.Local, xmlData)
	}
}
