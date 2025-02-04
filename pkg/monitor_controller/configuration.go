package monitorcontroller

import (
	"fmt"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

const (
	SpeakerA = iota
	SpeakerB
	SpeakerC
	SpeakerD
	SubA
	SubB

	SPEAKER_LEN
)

type SpeakerType int

const (
	Speaker SpeakerType = iota
	Subwoofer
)

type SpeakerConfig struct {
	Name      MappingString
	Mute      MappingBool
	Type      SpeakerType
	Exclusive bool
}

type MasterConfig struct {
	MuteSwitch MappingBool
	DimSwitch  MappingBool
	Meter      MappingInt

	VolumeMcuChannel []gomcu.Channel
	VolumeMcuRaw     uint16
	VolumeDB         float64
	DimVolumeOffset  float64
}

type Configuration struct {
	Speaker map[int]SpeakerConfig
	Master  MasterConfig

	FocusriteSerialNumber string
	FocusriteDeviceId     int

	fcElementMap map[int]*Mapping
}

func (c *Configuration) UpdateMaps() {
	c.fcElementMap = make(map[int]*Mapping)
}

// Getter by controller mapping id
func (m *Configuration) GetMasterFaderMcuChannel() []gomcu.Channel {
	return m.Master.VolumeMcuChannel

}

func (m *Configuration) GetSpeakerEnabledMcuSwitch(id int) ([]gomcu.Switch, error) {
	if f, ok := m.Speaker[id]; ok {
		return f.Mute.McuButtons(), nil
	} else {
		return nil, fmt.Errorf("no speaker with id:%d found", id)
	}
}

func (m *Configuration) GetSpeakerName(id int) (string, error) {
	if f, ok := m.Speaker[id]; ok {
		return f.Name.Value, nil
	} else {
		return "", fmt.Errorf("no speaker with id:%d found", id)
	}
}
