package mcu

import "github.com/sebastianrau/gomcu"

// ----------------------- TO MCU -----------------------

type LedCommand struct {
	Led   gomcu.Switch
	State gomcu.State
}

type FaderCommand struct {
	Fader gomcu.Channel
	Value uint16
}

type TimeDisplayCommand struct {
	Text string
}

type ChannelTextCommand struct {
	Fader      gomcu.Channel
	Text       string
	BottomLine bool
}

type VPotLedCommand struct {
	Channel gomcu.Channel
	Mode    gomcu.VPotMode
	Led     gomcu.VPotLED
}

type MeterCommand struct {
	Channel gomcu.Channel
	Value   gomcu.MeterLevel
}

type FaderSelectCommand struct {
	Channel     gomcu.Channel
	ChnnalValue uint16
}

// ----------------------- FROM MCU -----------------------

type KeyMessage struct {
	KeyNumber  gomcu.Switch
	Pressed    bool
	HotkeyName string
}

type BankMessage struct {
	Offset int
}

type RawFaderTouchMessage struct {
	Channel byte
	Pressed bool
}

type VPotButtonMessage struct {
	FaderNumber byte
}

type MuteMessage struct {
	FaderNumber byte
	Value       bool
}

type RecMessage struct {
	FaderNumber byte
}

type SoloMessage struct {
	FaderNumber byte
}

type AssignMessage struct {
	Mode byte
}

type SelectMessage struct {
	FaderNumber gomcu.Channel
}

type VPotChangeMessage struct {
	FaderNumber  byte
	ChangeAmount int
}

type RawFaderMessage struct {
	FaderNumber gomcu.Channel
	FaderValue  uint16
}

type ConnectionMessage struct {
	Connection bool
}
