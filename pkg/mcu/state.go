package mcu

import (
	"fmt"
	"time"

	"github.com/normen/obs-mcu/config"
	"github.com/normen/obs-mcu/gomcu"
	"gitlab.com/gomidi/midi/v2"
)

// McuState stores the current state of the MCU
// and syncs it with the hardware
type McuState struct {
	// TODO: combine
	faderLevels       []uint16
	faderTouch        []bool
	meterLevels       []byte
	faderTouchTimeout []time.Time
	ledStates         map[byte]bool
	vPotLedStates     map[byte]byte
	text              string
	display           string
	assign            []rune

	sendMidi func(m []midi.Message)
}

// NewMcuState creates a new McuState
func NewMcuState(sendMidi func(m []midi.Message)) *McuState {
	now := time.Now()

	state := McuState{
		text:              "",
		assign:            []rune{' ', ' '},
		faderLevels:       []uint16{0, 0, 0, 0, 0, 0, 0, 0, 0},
		meterLevels:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0},
		faderTouch:        []bool{false, false, false, false, false, false, false, false, false},
		faderTouchTimeout: []time.Time{now, now, now, now, now, now, now, now, now},
		ledStates:         make(map[byte]bool),
		vPotLedStates:     make(map[byte]byte),
		sendMidi:          sendMidi,
	}

	return &state
}

// UpdateTouch checks if a simulated fader touch has timed out
// and sends the buffered value
func (m *McuState) UpdateTouch() {
	for i, level := range m.faderLevels {
		if m.faderTouch[i] {
			now := time.Now()
			timeout := m.faderTouchTimeout[i]
			since := now.Sub(timeout)
			if since.Milliseconds() > 300 {
				m.faderTouch[i] = false
				// sends if not already same
				m.SetFaderLevel(byte(i), level)
			}
		}
	}
}

// SetFaderTouched sets the fader touch state and sends the buffered value
// if the touch has ended
func (m *McuState) SetFaderTouched(fader byte, touched bool) {
	m.faderTouch[fader] = touched
	if !touched {
		m.SetFaderLevel(fader, m.faderLevels[fader])
	} else if config.Config.McuFaders.SimulateTouch {
		m.faderTouchTimeout[fader] = time.Now()
	}
}

// SetFaderLevel sets the fader level and sends the value to the hardware
// fader if it has changed
func (m *McuState) SetFaderLevel(fader byte, level uint16) {

	if !m.faderTouch[fader] {
		if m.faderLevels[fader] != level {
			m.faderLevels[fader] = level
			channel := gomcu.Channel(fader)
			x := []midi.Message{gomcu.SetFaderPos(channel, level)}
			m.sendMidi(x)

		}
	}
}

// SetMonitorState sets the monitor state for a fader (rec+solo button)
func (m *McuState) SetMonitorState(fader byte, state string) {
	// OBS_MONITORING_TYPE_NONE
	// OBS_MONITORING_TYPE_MONITOR_AND_OUTPUT
	// OBS_MONITORING_TYPE_MONITOR_ONLY
	num := byte(gomcu.Rec1) + fader
	num2 := byte(gomcu.Solo1) + fader
	switch state {
	case "OBS_MONITORING_TYPE_NONE":
		m.SendLed(num, false)
		m.SendLed(num2, false)
	case "OBS_MONITORING_TYPE_MONITOR_AND_OUTPUT":
		m.SendLed(num, false)
		m.SendLed(num2, true)
	case "OBS_MONITORING_TYPE_MONITOR_ONLY":
		m.SendLed(num, true)
		m.SendLed(num2, false)
	}
}

// SetMuteState sets the mute state for a fader
func (m *McuState) SetMuteState(fader byte, state bool) {
	num := byte(gomcu.Mute1) + fader
	m.SendLed(num, state)
}

// SetSelectState sets the selected fader and lights up
// the select buttons accordingly
func (m *McuState) SetSelectState(fader byte, state bool) {
	for i := 0; i < 8; i++ {
		lit := (byte(i) == fader) && state
		num := byte(gomcu.Select1) + byte(i)
		m.SendLed(num, lit)
	}
}

// SetAssignMode sets the assign mode and lights up
// the assign buttons accordingly
func (m *McuState) SetAssignMode(number byte) {
	for i := 0; i < 6; i++ {
		lit := (byte(i) == number)
		num := byte(gomcu.AssignTrack) + byte(i)
		m.SendLed(num, lit)
	}
}

// SetTrackEnabledState sets the enabled state for a single track
func (m *McuState) SetTrackEnabledState(track byte, state bool) {
	num := byte(gomcu.Read) + track
	m.SendLed(num, state)
}

// SendLed checks if the led state has changed and sends the
// message to the hardware if it has changed
func (m *McuState) SendLed(num byte, state bool) {
	if m.ledStates[num] != state {
		m.ledStates[num] = state
		var mstate gomcu.State
		if state {
			mstate = gomcu.StateOn
		} else {
			mstate = gomcu.StateOff
		}
		x := []midi.Message{gomcu.SetLED(gomcu.Switch(num), mstate)}
		m.sendMidi(x)
	}

	fmt.Println(m.ledStates)
}

// SetAssignText sets the two letters above the assign buttons
func (m *McuState) SetAssignText(text []rune) {
	if m.assign[0] != text[0] || m.assign[1] != text[1] {
		x := []midi.Message{gomcu.SetDigit(gomcu.AssignLeft, gomcu.Char(text[0])), gomcu.SetDigit(gomcu.AssignRight, gomcu.Char(text[1]))}
		m.sendMidi(x)
		m.assign = text
	}
}

// SetDisplayText sets the text on the display (LED)
func (m *McuState) SetDisplayText(text string) {
	if len(text) > 10 {
		text = text[:10]
	} else {
		text = fmt.Sprintf("%-10s", text)
	}
	if m.display != text {
		m.display = text
		x := []midi.Message{}
		x = append(x, gomcu.SetTimeDisplay(text)...)
		m.sendMidi(x)

	}
}

// SetChannelText sets the text above the fader channel strip (LCD)
// the text is automatically shortened to 6 characters
func (m *McuState) SetChannelText(fader byte, text string, lower bool) {
	idx := int(fader * 7)
	if lower {
		idx += 56
	}
	text = ShortenText(text)
	if m.text[idx:idx+6] != text {
		m.text = fmt.Sprintf("%s%s%s", m.text[0:idx], text, m.text[idx+6:])
		x := []midi.Message{gomcu.SetLCD(idx, text)}
		m.sendMidi(x)
	}
}

// SetMeter sets the meter level for a fader, it is sent directly
func (m *McuState) SetMeter(fader byte, value float64) {
	var outByte byte
	if value >= 0 {
		outByte = byte(gomcu.MoreThan0)
	} else if value > -2 {
		outByte = byte(gomcu.MoreThan2)
	} else if value > -4 {
		outByte = byte(gomcu.MoreThan4)
	} else if value > -6 {
		outByte = byte(gomcu.MoreThan6)
	} else if value > -8 {
		outByte = byte(gomcu.MoreThan8)
	} else if value > -10 {
		outByte = byte(gomcu.MoreThan10)
	} else if value > -14 {
		outByte = byte(gomcu.MoreThan14)
	} else if value > -20 {
		outByte = byte(gomcu.MoreThan20)
	} else if value > -30 {
		outByte = byte(gomcu.MoreThan30)
	} else if value > -40 {
		outByte = byte(gomcu.MoreThan40)
	} else if value > -50 {
		outByte = byte(gomcu.MoreThan50)
	} else if value > -60 {
		outByte = byte(gomcu.MoreThan60)
	} else {
		outByte = byte(gomcu.LessThan60)
	}
	if m.meterLevels[fader] != outByte {
		m.meterLevels[fader] = outByte
		x := []midi.Message{gomcu.SetMeter(gomcu.Channel(fader), gomcu.MeterLevel(outByte))}
		m.sendMidi(x)

	}
}

// SetVPotLed sets the LED state for a VPot
// value 0 is off, values 1-12 are full left
func (m *McuState) SetVPotLed(fader byte, value byte) {
	if m.vPotLedStates[fader] != value {
		m.vPotLedStates[fader] = value
		x := []midi.Message{gomcu.SetVPot(gomcu.Channel(fader), gomcu.VPotMode0, gomcu.VPotLED(value))}
		m.sendMidi(x)
	}
}