package monitorcontroller

import "github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"

type SpeakerEnabledMessage struct {
	SpeakerID      int
	SpeakerEnabled []bool
}

type SpeakerLevelMessage struct {
	SpeakerID      int
	SpeakerLevel   []uint16
	SpeakerLevelDB []float64
}

func (c *Controller) NewSpeakerLevelMessage(id int) *SpeakerLevelMessage {
	return &SpeakerLevelMessage{SpeakerID: id, SpeakerLevel: c.speakerLevel, SpeakerLevelDB: c.speakerLevelDB}
}

type MuteMessage struct {
	Mute bool
}

type TransportMessage struct {
	Key gomcu.Switch
}
