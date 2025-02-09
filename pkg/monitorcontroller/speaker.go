package monitorcontroller

type SpeakerID int

const (
	SpeakerA SpeakerID = iota
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

var SpeakerName map[SpeakerID]string = map[SpeakerID]string{
	SpeakerA: "Speaker A",
	SpeakerB: "Speaker B",
	SpeakerC: "Speaker C",
	SpeakerD: "Speaker D",
	Sub:      "Sub",
}
