package monitorcontroller

import "github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"

type SpeakerEnabledMessage struct {
	SpeakerID      int
	SpeakerEnabled []bool
}

type SpeakerLevelMessage struct {
	SpeakerID    int
	SpeakerLevel []uint16
}

type MuteMessage struct {
	Mute bool
}

type TransportMessage struct {
	Key gomcu.Switch
}
