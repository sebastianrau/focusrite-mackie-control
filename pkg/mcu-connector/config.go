package mcuconnector

import (
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
	"github.com/sebastianrau/gomcu"
)

var (
	ALL_GOMCU_MUTES         []gomcu.Switch  = []gomcu.Switch{gomcu.Mute1, gomcu.Mute2, gomcu.Mute3, gomcu.Mute4, gomcu.Mute5, gomcu.Mute6, gomcu.Mute7, gomcu.Mute8}
	ALL_GOMUC_SOLO          []gomcu.Switch  = []gomcu.Switch{gomcu.Solo1, gomcu.Solo2, gomcu.Solo3, gomcu.Solo4, gomcu.Solo5, gomcu.Solo6, gomcu.Solo7, gomcu.Solo8}
	ALL_GOMCU_FADER_CHANNEL []gomcu.Channel = []gomcu.Channel{gomcu.Channel1, gomcu.Channel2, gomcu.Channel3, gomcu.Channel4, gomcu.Channel5, gomcu.Channel6, gomcu.Channel7, gomcu.Channel8, gomcu.Master}
)

type McuConnectorConfig struct {
	MidiInputPort  string `yaml:"MidiInputPort"`
	MidiOutputPort string `yaml:"MidiOnputPort"`

	SpeakerSelect map[monitorcontroller.SpeakerID]gomcu.Switch

	MasterMuteSwitch    gomcu.Switch
	MasterDimSwitch     gomcu.Switch
	MasterVolumeChannel gomcu.Channel

	FaderScaleLog bool
}

func DefaultConfiguration() *McuConnectorConfig {
	/*
		faderport := &McuConnectorConfig{
			MidiInputPort:  "PreSonus FP2",
			MidiOutputPort: "PreSonus FP2",
			SpeakerSelect: map[monitorcontroller.SpeakerID][]gomcu.Switch{
				monitorcontroller.SpeakerA: gomcu.Trim,
				monitorcontroller.SpeakerB: gomcu.Touch,
				monitorcontroller.SpeakerC: gomcu.Write,
				monitorcontroller.SpeakerD: gomcu.F4,
				monitorcontroller.Sub:      gomcu.Read,
			},
			MasterMuteSwitch:    gomcu.Mute1,
			MasterDimSwitch:     gomcu.Solo1,
			MasterVolumeChannel: gomcu.Channel1,
			FaderScaleLog:       false,
		}
	*/
	streamDeck := &McuConnectorConfig{
		MidiInputPort:  "IAC StreamDeckToController",
		MidiOutputPort: "IAC ControllerToStreamDeck",
		SpeakerSelect: map[monitorcontroller.SpeakerID]gomcu.Switch{
			monitorcontroller.SpeakerA: gomcu.Trim,
			monitorcontroller.SpeakerB: gomcu.Touch,
			monitorcontroller.SpeakerC: gomcu.Write,
			monitorcontroller.SpeakerD: gomcu.F4,
			monitorcontroller.Sub:      gomcu.Read,
		},
		MasterMuteSwitch:    gomcu.Mute1,
		MasterDimSwitch:     gomcu.Solo1,
		MasterVolumeChannel: gomcu.Channel1,
		FaderScaleLog:       false,
	}

	return streamDeck
}
