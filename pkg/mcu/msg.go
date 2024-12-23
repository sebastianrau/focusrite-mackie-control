package mcu

/*
	Rec1 Switch = iota
	Rec2
	Rec3
	Rec4
	Rec5
	Rec6
	Rec7
	Rec8
	Solo1
	Solo2
	Solo3
	Solo4
	Solo5
	Solo6
	Solo7
	Solo8
	Mute1
	Mute2
	Mute3
	Mute4
	Mute5
	Mute6
	Mute7
	Mute8
	Select1
	Select2
	Select3
	Select4
	Select5
	Select6
	Select7
	Select8
	V1
	V2
	V3
	V4
	V5
	V6
	V7
	V8
	AssignTrack
	AssignSend
	AssignPan
	AssignPlugin
	AssignEQ
	AssignInstrument
	BankL
	BankR
	ChannelL
	ChannelR
	Flip
	GlobalView
	NameValue
	SMPTEBeats
	F1
	F2
	F3
	F4
	F5
	F6
	F7
	F8
	MIDITracks
	Inputs
	AudioTracks
	AudioInstrument
	Aux
	Busses
	Outputs
	User
	Shift
	Option
	Control
	CMDAlt
	Read
	Write
	Trim
	Touch
	Latch
	Group
	Save
	Undo
	Cancel
	Enter
	Marker
	Nudge
	Cycle
	Drop
	Replace
	Click
	Solo
	Rewind
	FastFwd
	Stop
	Play
	Record
	Up
	Down
	Left
	Right
	Zoom
	Scrub
	UserA
	UserB
	Fader1
	Fader2
	Fader3
	Fader4
	Fader5
	Fader6
	Fader7
	Fader8
	FaderMaster
	STMPELED
	BeatsLED
	RudeSoloLED
	RelayClickLED
*/

type KeyMessage struct {
	HotkeyName string
}

// obs <- mackie
type BankMessage struct {
	ChangeAmount int
}

// obs <- mackie
type VPotChangeMessage struct {
	FaderNumber  byte
	ChangeAmount int
}

// obs <- mackie
type VPotButtonMessage struct {
	FaderNumber byte
}

// obs <-> mackie
type MuteMessage struct {
	FaderNumber byte
	Value       bool
}

// obs <-> mackie
type SelectMessage struct {
	FaderNumber byte
	Value       bool
}

// obs <-> mackie
type AssignMessage struct {
	Mode byte
}

// obs <-> mackie
type TrackEnableMessage struct {
	TrackNumber byte
	Value       bool
}

// obs <-> mackie
type FaderMessage struct {
	FaderNumber byte
	FaderValue  uint16
}

// obs <-> mackie
type MonitorTypeMessage struct {
	FaderNumber byte
	MonitorType string
}

// obs -> mackie
type LedMessage struct {
	LedName  string
	LedState bool
}

// obs -> mackie
type ChannelTextMessage struct {
	FaderNumber byte
	Text        string
	Lower       bool
}

// obs -> mackie
type DisplayTextMessage struct {
	Text string
}

// obs -> mackie
type VPotLedMessage struct {
	FaderNumber byte
	LedState    byte
}

// obs -> mackie
type AssignLEDMessage struct {
	Characters []rune
}

// obs -> mackie
type MeterMessage struct {
	FaderNumber byte
	Value       float64
}

// internal mcu message
type RawFaderMessage struct {
	FaderNumber byte
	FaderValue  uint16
}

// internal mcu message
type RawFaderTouchMessage struct {
	FaderNumber byte
	Pressed     bool
}
