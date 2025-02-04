package monitorcontroller

import (
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

type SpeakerEnabledMessage struct {
	SpeakerID      int
	SpeakerEnabled bool
}

type MasterLevelMessage struct {
	SpeakerLevel   uint16
	SpeakerLevelDB float64
}

type MuteMessage bool
type DimMessage bool
type TransportMessage gomcu.Switch
