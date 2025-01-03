package monitorcontroller

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
