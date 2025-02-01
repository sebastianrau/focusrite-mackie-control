package mcu

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/config"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sirupsen/logrus"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

var log *logrus.Entry = logger.WithPackage("mcu")

type Mcu struct {
	config       *config.Config
	waitGroup    *sync.WaitGroup
	midiInput    drivers.In
	midiOutput   drivers.Out
	midiStop     func()
	connectRetry *time.Timer

	interrupt  chan os.Signal
	connection chan int

	decodeButtons bool
	toMcu         chan interface{}
	fromMcu       chan interface{}

	displayStringUpper []byte
	displayStringLower []byte
	selectedChannel    gomcu.Channel
}

// Initialize the MCU runloop
func InitMcu(fromMcu chan interface{}, toMcu chan interface{}, interrupt chan os.Signal, wg *sync.WaitGroup, cfg config.Config) *Mcu {
	m := Mcu{
		config:             &cfg,
		waitGroup:          wg,
		fromMcu:            fromMcu,
		toMcu:              toMcu,
		interrupt:          interrupt,
		connection:         make(chan int, 1),
		displayStringUpper: make([]byte, 56),
		displayStringLower: make([]byte, 56),
		selectedChannel:    gomcu.Channel1,
	}

	for i := 0; i < 8; i++ {
		m.updateLcdText(gomcu.Channel(i), fmt.Sprintf("%-6s", " "), false)
		m.updateLcdText(gomcu.Channel(i), fmt.Sprintf("%-6s", " "), true)
	}
	m.DecodeChannelSelect(uint8(gomcu.Channel1))

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
		log.Infof("Could not find MIDI Input '%s'", m.config.MidiInputPort)
		m.retryConnect()
		return
	}

	m.midiOutput, err = midi.FindOutPort(m.config.MidiOutputPort)
	if err != nil {
		log.Infof("Could not find MIDI Output '%s'", m.config.MidiOutputPort)
		m.retryConnect()
		return
	}

	err = m.midiInput.Open()
	if err != nil {
		log.Errorf("Could not open MIDI Input '%s'", m.config.MidiInputPort)
		m.retryConnect()
		return
	}
	err = m.midiOutput.Open()
	if err != nil {
		log.Errorf("Could not open MIDI Output '%s'", m.config.MidiOutputPort)
		m.retryConnect()
		return
	}

	gomcu.Reset(m.midiOutput)

	m.midiStop, err = midi.ListenTo(m.midiInput, m.receiveMidi)
	if err != nil {
		log.Errorf(err.Error())
		m.retryConnect()
		return
	}

	send, err := midi.SendTo(m.midiOutput)
	if err != nil {
		log.Errorf(err.Error())
		m.retryConnect()
		return
	}

	msg := []midi.Message{}
	msg = append(msg, gomcu.SetTimeDisplay("Monitor Control")...)
	for _, ms := range msg {
		send(ms)
	}

	m.fromMcu <- ConnectionMessage{Connection: true}
}

// disconnects from the MCU, called from runloop
func (m *Mcu) disconnect() {
	if m.midiStop != nil {
		m.midiStop()
		m.midiStop = nil
	}
	if m.midiInput != nil {
		err := m.midiInput.Close()
		if err != nil {
			log.Errorf(err.Error())
		}
		m.midiInput = nil
	}
	if m.midiOutput != nil {
		err := m.midiOutput.Close()
		if err != nil {
			log.Errorf(err.Error())
		}
		m.midiOutput = nil
	}
}

// retry connection after 3 seconds
func (m *Mcu) retryConnect() {
	log.Infof("Retry MIDI connection ....")
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
		log.Warnf(err.Error())
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
		// avoid noteoffs for the other commands
		if v == 0 {
			return
		}

		if m.DecodeChannelSelect(k) {
			return
		}

		if m.decodeButtons {
			m.DecodeButtons(k, v)
		} else {
			fieldName := gomcu.Names[k]
			m.fromMcu <- KeyMessage{
				KeyNumber:  gomcu.Switch(k),
				Pressed:    true,
				HotkeyName: fieldName,
			}
		}

	} else if message.GetControlChange(&c, &k, &v) {
		if inRange(k, gomcu.Mute1, gomcu.Mute8) {
			amount := 0
			if v < 65 {
				amount = int(v)
			} else {
				amount = -1 * (int(v) - 64)
			}
			m.fromMcu <- VPotChangeMessage{
				FaderNumber:  k - byte(gomcu.Mute1),
				ChangeAmount: amount,
			}
		}

	} else if message.GetPitchBend(&c, &val, &uval) {
		m.fromMcu <- RawFaderMessage{
			FaderNumber: gomcu.Channel(c),
			FaderValue:  uval,
		}
	}

}

func (m *Mcu) DecodeChannelSelect(k uint8) bool {
	if inRange(k, gomcu.Select1, gomcu.Select8) {

		newChannel := gomcu.Channel(k - uint8(gomcu.Select1))
		oldLed := gomcu.Switch(m.selectedChannel) + gomcu.Select1
		newLed := gomcu.Switch(k)

		if newChannel != m.selectedChannel {
			m.selectedChannel = gomcu.Channel(newChannel)
			m.toMcu <- LedCommand{Led: oldLed, State: gomcu.StateOff}
			m.toMcu <- LedCommand{Led: newLed, State: gomcu.StateOn}
		}

		m.fromMcu <- SelectMessage{
			FaderNumber: newChannel,
		}
		return true
	}

	return false
}

func (m *Mcu) DecodeButtons(k uint8, v uint8) {
	if inRange(k, gomcu.Fader1, gomcu.FaderMaster) {
		m.fromMcu <- RawFaderTouchMessage{Channel: k - byte(gomcu.Fader1), Pressed: v == 127}
	} else if inRange(k, gomcu.BankL, gomcu.ChannelR) {
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
		m.fromMcu <- BankMessage{Offset: amount}

	} else if inRange(k, gomcu.V1, gomcu.V8) {
		m.fromMcu <- VPotButtonMessage{FaderNumber: k - byte(gomcu.V1)}
	} else if inRange(k, gomcu.Mute1, gomcu.Mute8) {
		m.fromMcu <- MuteMessage{FaderNumber: k - byte(gomcu.Mute1)}
	} else if inRange(k, gomcu.Rec1, gomcu.Rec8) {
		m.fromMcu <- RecMessage{FaderNumber: k}
	} else if inRange(k, gomcu.Solo1, gomcu.Solo8) {
		m.fromMcu <- SoloMessage{FaderNumber: k - byte(gomcu.Solo1)}
	} else if inRange(k, gomcu.AssignTrack, gomcu.AssignInstrument) {
		m.fromMcu <- AssignMessage{Mode: k - byte(gomcu.AssignTrack)}
	} else {
		m.fromMcu <- KeyMessage{KeyNumber: gomcu.Switch(k), HotkeyName: gomcu.Names[k]}
	}
}

// run the MCU
func (m *Mcu) run() {
	for {
		select {

		case state := <-m.connection:
			if state == 0 {
				m.connect()
				m.fromMcu <- ConnectionMessage{Connection: false}
			} else {
				m.fromMcu <- ConnectionMessage{Connection: true}
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

			case LedCommand:
				m.sendMidi([]midi.Message{gomcu.SetLED(e.Led, e.State)})
			case FaderCommand:
				m.sendMidi([]midi.Message{gomcu.SetFaderPos(e.Fader, e.Value)})

			case TimeDisplayCommand:
				m.sendMidi(gomcu.SetTimeDisplay(e.Text))

			case ChannelTextCommand:
				m.updateLcdText(e.Fader, e.Text, e.BottomLine)

				if e.BottomLine {
					m.sendMidi([]midi.Message{gomcu.SetLCD(56, string(m.displayStringLower))})
				} else {
					m.sendMidi([]midi.Message{gomcu.SetLCD(0, string(m.displayStringUpper))})
				}

			case VPotLedCommand:
				m.sendMidi([]midi.Message{gomcu.SetVPot(e.Channel, e.Mode, e.Led)})

			case MeterCommand:
				m.sendMidi([]midi.Message{gomcu.SetMeter(e.Channel, e.Value)})

			case FaderSelectCommand:

				for i := gomcu.Select1; i <= gomcu.Select8; i++ {
					m.sendMidi([]midi.Message{gomcu.SendOff(gomcu.Switch(i))})
				}
				m.sendMidi([]midi.Message{gomcu.SetLED(gomcu.Switch(e.Channel)+gomcu.Select1, gomcu.StateOn)})

			}

		}
	}
}

func (c *Mcu) updateLcdText(channel gomcu.Channel, text string, lower bool) {
	text = ShortenText(text) + " "

	offset := int(channel) * 7
	if lower {
		copy(c.displayStringLower[offset:], text)
	} else {
		copy(c.displayStringUpper[offset:], text)
	}
}

func inRange(val byte, low gomcu.Switch, high gomcu.Switch) bool {
	return gomcu.Switch(val) >= low && gomcu.Switch(val) <= high
}
