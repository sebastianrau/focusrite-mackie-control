package monitorcontroller

import (
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
}
