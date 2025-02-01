package monitorcontroller

import (
	"fmt"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

const (
	SpeakerA = iota
	SpeakerB
	SpeakerC
	SpeakerD
	SubA
	SubB
	MasterFader

	faderLength
)

type SpeakerType int

const (
	SpeakerNormal SpeakerType = iota
	SubwooferNormal
)

const (
	SpeakerAEnabled = iota
	SpeakerBEnabled
	SpeakerCEnabled
	SpeakerDEnabled
	SubAEnabled
	SubBEnabled
	Mute

	buttonLength
)

type Speaker struct {
	Name string
	Mcu  McuConfiguration
	//Focusrite FocusriteConfiguration
	Type      SpeakerType
	Exclusive bool
}

/*
type FocusriteConfiguration struct {
	SerialNumber focusritexml.ElementString
	Nickname     focusritexml.ElementString
	Available    focusritexml.ElementBool
	Volume       focusritexml.ElementInt
	Mute         focusritexml.ElementBool
	Meter        focusritexml.ElementInt
}
*/

type McuConfiguration struct {
	EnableButton []gomcu.Switch
}

type Master struct {
	Name      string
	Mcu       McuMaster
	Focusrite FocusriteMaster
}

type McuMaster struct {
	Fader        gomcu.Channel
	SelectButton gomcu.Switch
}

type FocusriteMaster struct {
	Mute focusritexml.ElementBool
}

type Configuration struct {
	Speaker      map[int]Speaker
	Master       Master
	SerialNumber focusritexml.ElementString
}

var (
	SpeakerNames = map[int]string{
		SpeakerA:    "Speaker A",
		SpeakerB:    "Speaker B",
		SpeakerC:    "Speaker C",
		SpeakerD:    "Speaker D",
		SubA:        "Sub A",
		SubB:        "Sub B",
		MasterFader: "Master",
	}

	MuteButtons = []gomcu.Switch{
		gomcu.Mute1,
		gomcu.Mute2,
		gomcu.Mute3,
		gomcu.Mute4,
		gomcu.Mute5,
		gomcu.Mute6,
		gomcu.Mute7,
		gomcu.Mute8,
	}
)

// Getter by controller mapping id

func (m *Configuration) GetMasterFader() (gomcu.Channel, error) {
	return m.Master.Mcu.Fader, nil

}

func (m *Configuration) GetMcuEnabledSwitch(id int) ([]gomcu.Switch, error) {
	if f, ok := m.Speaker[id]; ok {
		return f.Mcu.EnableButton, nil
	} else {
		return nil, fmt.Errorf("name not found: %d", id)
	}
}

func (m *Configuration) GetMcuName(id int) (string, error) {
	if f, ok := m.Speaker[id]; ok {
		return f.Name, nil
	} else {
		return "", fmt.Errorf("name not found: %d", id)
	}
}

func (m *Configuration) GetIdBySwitch(c gomcu.Switch) (int, bool) {
	for k, v := range m.Speaker {
		for _, b := range v.Mcu.EnableButton {
			if b == c {
				return k, true
			}
		}
	}
	return -1, false
}
