package monitorcontroller

import (
	"github.com/sebastianrau/gomcu"
)

type SpeakerEnabledMessage struct {
	SpeakerID      SpeakerID
	SpeakerEnabled bool
}

type MasterLevelMessage struct {
	SpeakerLevel   uint16
	SpeakerLevelDB float64
}

type MuteMessage bool
type DimMessage bool
type TransportMessage gomcu.Switch
