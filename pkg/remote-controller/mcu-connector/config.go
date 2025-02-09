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

	SpeakerSelect map[monitorcontroller.SpeakerID][]gomcu.Switch

	MasterMuteSwitch    []gomcu.Switch
	MasterDimSwitch     []gomcu.Switch
	MasterVolumeChannel []gomcu.Channel
}

func DefaultConfiguration() *McuConnectorConfig {

	c := &McuConnectorConfig{
		MidiInputPort:  "PreSonus FP2",
		MidiOutputPort: "PreSonus FP2",
		SpeakerSelect: map[monitorcontroller.SpeakerID][]gomcu.Switch{
			monitorcontroller.SpeakerA: []gomcu.Switch{gomcu.Trim},
			monitorcontroller.SpeakerB: []gomcu.Switch{gomcu.Touch},
			monitorcontroller.SpeakerC: []gomcu.Switch{gomcu.Write},
			monitorcontroller.SpeakerD: []gomcu.Switch{},
			monitorcontroller.Sub:      []gomcu.Switch{gomcu.Read},
		},
		MasterMuteSwitch:    ALL_GOMCU_MUTES,
		MasterDimSwitch:     ALL_GOMUC_SOLO,
		MasterVolumeChannel: ALL_GOMCU_FADER_CHANNEL,
	}

	return c

}
