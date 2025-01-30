package monitorcontroller

import "github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"

type SpeakerEnabledMessage struct {
	SpeakerID      int
	SpeakerEnabled []bool
}

type MasterLevelMessage struct {
	SpeakerLevel   uint16
	SpeakerLevelDB float64
}

// TODO Rename
func (c *Controller) NewSpeakerLevelMessage() *MasterLevelMessage {
	return &MasterLevelMessage{SpeakerLevel: c.masterLevel, SpeakerLevelDB: c.masterLevelDB}
}

type MuteMessage struct {
	Mute bool
}

type TransportMessage struct {
	Key gomcu.Switch
}
