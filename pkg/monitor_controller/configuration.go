package monitorcontroller

import (
	faderdb "github.com/sebastianrau/focusrite-mackie-control/pkg/faderDB"
	"github.com/sebastianrau/gomcu"
)

const (
	SpeakerA = iota
	SpeakerB
	SpeakerC
	SpeakerD
	Sub

	SPEAKER_LEN
)

type SpeakerType int

const (
	Speaker SpeakerType = iota
	Subwoofer
)

type SpeakerConfig struct {
	Name       MappingString
	Mute       MappingBool
	OutputGain MappingInt
	Meter      MappingInt
	Type       SpeakerType
	Exclusive  bool
}

type MasterConfig struct {
	MuteSwitch MappingBool
	DimSwitch  MappingBool
	//Meter      MappingInt

	VolumeMcuChannel []gomcu.Channel
	VolumeMcuRaw     uint16
	VolumeDB         int
	DimVolumeOffset  int
}

type Configuration struct {
	Speaker map[int]*SpeakerConfig
	Master  MasterConfig

	FocusriteSerialNumber string
	FocusriteDeviceId     int
}

func (c *Configuration) DefaultValues() {
	c.Master.DimSwitch.Value = false
	c.Master.DimSwitch.Value = false
	c.Master.VolumeMcuRaw = 0
	c.Master.VolumeDB = int(faderdb.FaderToDB(0))

	for _, spk := range c.Speaker {
		spk.Mute.Value = true
		spk.OutputGain.Value = -127
	}
}
