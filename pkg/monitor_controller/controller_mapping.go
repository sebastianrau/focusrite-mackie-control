package monitorcontroller

import (
	"fmt"

	"github.com/normen/obs-mcu/gomcu"
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
	Name         string
	Fader        gomcu.Channel
	SelectButton gomcu.Switch
	EnableButton []gomcu.Switch
}

type Master struct {
	Name         string
	Fader        gomcu.Channel
	SelectButton gomcu.Switch
}

type ControllerMapping struct {
	Speaker map[int]Speaker
	Master  Master
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

func DefaultMapping() *ControllerMapping {
	m := ControllerMapping{
		Speaker: map[int]Speaker{
			SpeakerA: {Name: SpeakerNames[SpeakerA], Fader: gomcu.Channel1, SelectButton: gomcu.Select1, EnableButton: []gomcu.Switch{gomcu.AssignTrack}},
			SpeakerB: {Name: SpeakerNames[SpeakerB], Fader: gomcu.Channel2, SelectButton: gomcu.Select2, EnableButton: []gomcu.Switch{gomcu.AssignSend}},
			SpeakerC: {Name: SpeakerNames[SpeakerC], Fader: gomcu.Channel3, SelectButton: gomcu.Select3, EnableButton: []gomcu.Switch{gomcu.AssignPan}},
			SpeakerD: {Name: SpeakerNames[SpeakerD], Fader: gomcu.Channel4, SelectButton: gomcu.Select4, EnableButton: []gomcu.Switch{gomcu.AssignPlugin}},
			SubA:     {Name: SpeakerNames[SubA], Fader: gomcu.Channel5, SelectButton: gomcu.Select5, EnableButton: []gomcu.Switch{gomcu.AssignEQ}},
			SubB:     {Name: SpeakerNames[SubB], Fader: gomcu.Channel6, SelectButton: gomcu.Select6, EnableButton: []gomcu.Switch{gomcu.AssignInstrument}},
		},

		Master: Master{Name: SpeakerNames[MasterFader], Fader: gomcu.Master, SelectButton: gomcu.FaderMaster},
	}
	return &m
}

// Getter by controller mapping id

func (m *ControllerMapping) GetMcuFader(id int) (gomcu.Channel, error) {
	if f, ok := m.Speaker[id]; ok {
		return f.Fader, nil
	}

	if id == MasterFader {
		return m.Master.Fader, nil
	}

	return 0, fmt.Errorf("name not found: %d", id)
}

func (m *ControllerMapping) GetMcuEnabledSwitch(id int) ([]gomcu.Switch, error) {
	if f, ok := m.Speaker[id]; ok {
		return f.EnableButton, nil
	} else {
		return nil, fmt.Errorf("name not found: %d", id)
	}
}

func (m *ControllerMapping) GetMcuName(id int) (string, error) {
	if f, ok := m.Speaker[id]; ok {
		return f.Name, nil
	} else {
		return "", fmt.Errorf("name not found: %d", id)
	}
}

func (m *ControllerMapping) GetIdByFader(c gomcu.Channel) (int, bool) {
	for k, v := range m.Speaker {
		if v.Fader == c {
			return k, true
		}
	}
	if m.Master.Fader == c {
		return MasterFader, true
	}

	return -1, false
}

func (m *ControllerMapping) GetIdBySwitch(c gomcu.Switch) (int, bool) {
	for k, v := range m.Speaker {
		for _, b := range v.EnableButton {
			if b == c {
				return k, true
			}
		}
	}
	return -1, false
}
