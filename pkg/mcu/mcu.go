package mcu

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"

	"github.com/normen/obs-mcu/gomcu"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

type Mcu struct {
	config       *config.Config
	waitGroup    *sync.WaitGroup
	state        *McuState
	midiInput    drivers.In
	midiOutput   drivers.Out
	midiStop     func()
	connectRetry *time.Timer

	interrupt  chan os.Signal
	connection chan int

	toMcu       chan interface{}
	fromMcu     chan interface{}
	internalMcu chan interface{}
}

// Initialize the MCU runloop
func InitMcu(fromMcu chan interface{}, toMcu chan interface{}, interrupt chan os.Signal, wg *sync.WaitGroup, cfg config.Config) *Mcu {
	m := Mcu{
		config:    &cfg,
		waitGroup: wg,
		fromMcu:   fromMcu,
		toMcu:     toMcu,
		interrupt: interrupt,

		connection:  make(chan int, 1),
		internalMcu: make(chan interface{}),
	}
	m.state = NewMcuState(m.sendMidi)

	m.connection <- 0
	wg.Add(1)
	go m.run()
	return &m
}

// connects to the MCU, called from runloop
func (m *Mcu) connect() {
	var err error
	m.disconnect()

	m.midiInput, err = midi.FindInPort(m.config.MidiInputPort)
	if err != nil {
		log.Printf("Could not find MIDI Input '%s'", m.config.MidiInputPort)
		m.retryConnect()
		return
	}

	m.midiOutput, err = midi.FindOutPort(m.config.MidiOutputPort)
	if err != nil {
		log.Printf("Could not find MIDI Output '%s'", m.config.MidiOutputPort)
		m.retryConnect()
		return
	}

	err = m.midiInput.Open()
	if err != nil {
		log.Printf("Could not open MIDI Input '%s'", m.config.MidiInputPort)
		m.retryConnect()
		return
	}
	err = m.midiOutput.Open()
	if err != nil {
		log.Printf("Could not open MIDI Output '%s'", m.config.MidiOutputPort)
		m.retryConnect()
		return
	}

	gomcu.Reset(m.midiOutput)

	m.midiStop, err = midi.ListenTo(m.midiInput, m.receiveMidi)
	if err != nil {
		log.Print(err)
		m.retryConnect()
		return
	}

	send, err := midi.SendTo(m.midiOutput)
	if err != nil {
		log.Print(err)
		m.retryConnect()
		return
	}

	msg := []midi.Message{}
	msg = append(msg, gomcu.SetTimeDisplay("Monitor Control")...)
	for _, ms := range msg {
		send(ms)
	}

	//TODO m.fromMcu <- ms.UpdateRequest{}
	log.Print("MIDI Connected")
}

// disconnects from the MCU, called from runloop
func (m *Mcu) disconnect() {
	//debug.PrintStack()
	if m.midiStop != nil {
		m.midiStop()
		m.midiStop = nil
	}
	if m.midiInput != nil {
		err := m.midiInput.Close()
		if err != nil {
			log.Print(err)
		}
		m.midiInput = nil
	}
	if m.midiOutput != nil {
		err := m.midiOutput.Close()
		if err != nil {
			log.Print(err)
		}
		m.midiOutput = nil
	}
}

// retry connection after 3 seconds
func (m *Mcu) retryConnect() {
	log.Print("Retry MIDI connection..")
	m.disconnect()
	if m.connectRetry != nil {
		m.connectRetry.Stop()
	}
	m.connectRetry = time.AfterFunc(3*time.Second, func() { m.connection <- 0 })
}

// check if midi connection is still open,
// call reconnect if not
func (m *Mcu) checkMidiConnection() bool {
	if m.midiInput != nil {
		if !m.midiInput.IsOpen() {
			m.retryConnect()
			return false
		}
	} else {
		return false
	}
	return true
}

// send a list of midi messages
func (mcu *Mcu) sendMidi(m []midi.Message) {
	send, err := midi.SendTo(mcu.midiOutput)
	if err != nil {
		log.Print(err)
		return
	}
	for _, msg := range m {
		send(msg)
	}
}

// receives midi messages from the MCU, called from midi runloop!
func (m *Mcu) receiveMidi(message midi.Message, timestamps int32) {
	var c, k, v uint8
	var val int16
	var uval uint16

	if message.GetNoteOn(&c, &k, &v) {

		// fader touch - handle locally
		if gomcu.Switch(k) >= gomcu.Fader1 && gomcu.Switch(k) <= gomcu.Fader8 {
			m.internalMcu <- RawFaderTouchMessage{
				FaderNumber: k - byte(gomcu.Fader1),
				Pressed:     v == 127,
			}
		}

		// avoid noteoffs for the other commands
		if v == 0 {
			return
		}

		if gomcu.Switch(k) >= gomcu.BankL && gomcu.Switch(k) <= gomcu.ChannelR {
			var amount int
			switch gomcu.Switch(k) {
			case gomcu.BankL:
				amount = -8
			case gomcu.BankR:
				amount = 8
			case gomcu.ChannelL:
				amount = -1
			case gomcu.ChannelR:
				amount = 1
			}
			m.fromMcu <- BankMessage{
				ChangeAmount: amount,
			}
		} else if gomcu.Switch(k) >= gomcu.V1 && gomcu.Switch(k) <= gomcu.V8 {
			m.fromMcu <- VPotButtonMessage{
				FaderNumber: k - byte(gomcu.V1),
			}
		} else if gomcu.Switch(k) >= gomcu.Mute1 && gomcu.Switch(k) <= gomcu.Mute8 {
			m.fromMcu <- MuteMessage{
				FaderNumber: k - byte(gomcu.Mute1),
			}
		} else if gomcu.Switch(k) >= gomcu.Rec1 && gomcu.Switch(k) <= gomcu.Rec8 {
			m.fromMcu <- MonitorTypeMessage{
				FaderNumber: k,
				MonitorType: "REC",
			}
		} else if gomcu.Switch(k) >= gomcu.Solo1 && gomcu.Switch(k) <= gomcu.Solo8 {
			m.fromMcu <- MonitorTypeMessage{
				FaderNumber: k - byte(gomcu.Solo1),
				MonitorType: "SOLO",
			}
		} else if gomcu.Switch(k) >= gomcu.Select1 && gomcu.Switch(k) <= gomcu.Select8 {
			m.fromMcu <- SelectMessage{
				FaderNumber: k - byte(gomcu.Select1),
			}
		} else if gomcu.Switch(k) >= gomcu.AssignTrack && gomcu.Switch(k) <= gomcu.AssignInstrument {
			m.fromMcu <- AssignMessage{
				Mode: k - byte(gomcu.AssignTrack),
			}
		} else if len(gomcu.Names) > int(k) {
			fieldName := gomcu.Names[k]
			m.fromMcu <- KeyMessage{
				HotkeyName: fieldName,
			}
		} else {
			log.Printf("Unknown Button with key: %x", k)
		}

	} else if message.GetControlChange(&c, &k, &v) {
		if gomcu.Switch(k) >= 0x10 && gomcu.Switch(k) <= 0x17 {
			amount := 0
			if v < 65 {
				amount = int(v)
			} else {
				amount = -1 * (int(v) - 64)
			}
			m.fromMcu <- VPotChangeMessage{
				FaderNumber:  k - 0x10,
				ChangeAmount: amount,
			}
		}

	} else if message.GetPitchBend(&c, &val, &uval) {
		m.fromMcu <- RawFaderMessage{
			FaderNumber: c,
			FaderValue:  uval,
		}
	}

}

// run the MCU
func (m *Mcu) run() {

	for {
		select {

		case state := <-m.connection:
			if state == 0 {
				m.connect()
			}

		case <-m.interrupt:
			m.disconnect()
			m.waitGroup.Done()
			return

		case message := <-m.toMcu:
			if !m.checkMidiConnection() {
				continue
			}

			switch e := message.(type) {
			case FaderMessage:
				m.state.SetFaderLevel(e.FaderNumber, e.FaderValue)
			case TrackEnableMessage:
				m.state.SetTrackEnabledState(e.TrackNumber, e.Value)
			case MuteMessage:
				m.state.SetMuteState(e.FaderNumber, e.Value)
			case ChannelTextMessage:
				m.state.SetChannelText(e.FaderNumber, e.Text, e.Lower)
			case DisplayTextMessage:
				m.state.SetDisplayText(e.Text)
			case AssignLEDMessage:
				m.state.SetAssignText(e.Characters)
			case MonitorTypeMessage:
				m.state.SetMonitorState(e.FaderNumber, e.MonitorType)
			case SelectMessage:
				m.state.SetSelectState(e.FaderNumber, e.Value)
			case AssignMessage:
				m.state.SetAssignMode(e.Mode)
			case VPotLedMessage:
				m.state.SetVPotLed(e.FaderNumber, e.LedState)
			case MeterMessage:
				m.state.SetMeter(e.FaderNumber, e.Value)
			case LedMessage:
				if num, ok := gomcu.IDs[e.LedName]; ok {
					m.state.SendLed(byte(num), e.LedState)
				} else {
					log.Printf("Could not find led with id %v", e.LedName)
				}
			}
		case message := <-m.internalMcu:
			switch e := message.(type) {
			case RawFaderTouchMessage:
				m.state.SetFaderTouched(e.FaderNumber, e.Pressed)
			}
		}
	}
}
