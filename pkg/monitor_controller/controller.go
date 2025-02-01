package monitorcontroller

// TODO: remove Fader fpr Speaker Level

import (
	"fmt"
	"reflect"

	faderdb "github.com/sebastianrau/focusrite-mackie-control/pkg/faderDB"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sirupsen/logrus"
)

var log *logrus.Entry = logger.WithPackage("monitor-controller")

type Controller struct {
	speakerEnabled []bool
	ledStates      []gomcu.State

	masterMute    bool
	masterLevel   uint16
	masterLevelDB float64
	masterMeter   gomcu.MeterLevel

	timeDisplay string
	mapping     *Configuration

	toMcu   chan interface{}
	fromMcu chan interface{}

	fromController chan interface{}
}

// NewMcuState creates a new McuState
func NewController(toMcu chan interface{}, fromMcu chan interface{}, fromController chan interface{}) *Controller {

	state := Controller{
		speakerEnabled: make([]bool, faderLength),
		ledStates:      make([]gomcu.State, buttonLength),

		masterMute:    false,
		masterLevel:   0,
		masterLevelDB: -100.0,
		masterMeter:   gomcu.LessThan60,

		timeDisplay: "12345670000890",
		mapping:     DefaultConfiguration(),

		toMcu:          toMcu,
		fromMcu:        fromMcu,
		fromController: fromController,
	}

	go state.Run()

	return &state
}

func (c *Controller) Run() {

	for {
		fm := <-c.fromMcu

		switch f := fm.(type) {

		case mcu.ConnectionMessage:
			if f.Connection {
				c.SetMute(true)
				c.initDisplay()
				c.setDisplayText(c.timeDisplay)
				masterFader, _ := c.mapping.GetMasterFader()
				c.toMcu <- mcu.SelectMessage{FaderNumber: masterFader}

			}

		case mcu.SelectMessage:
			if f.FaderNumber == c.mapping.Master.Mcu.Fader {
				c.toMcu <- mcu.FaderCommand{Fader: gomcu.Channel(f.FaderNumber), Value: c.masterLevel}
			}

		case mcu.KeyMessage:
			log.Debugf("Button: 0x%X %s", f.KeyNumber, f.HotkeyName)

			switch f.KeyNumber {
			case gomcu.Mute1,
				gomcu.Mute2,
				gomcu.Mute3,
				gomcu.Mute4,
				gomcu.Mute5,
				gomcu.Mute6,
				gomcu.Mute7,
				gomcu.Mute8:
				c.ToggleMute()

			case gomcu.Play,
				gomcu.Stop,
				gomcu.FastFwd,
				gomcu.Rewind:
				c.fromController <- TransportMessage{
					Key: f.KeyNumber,
				}
			}

			//Mapped Switches
			key, ok := c.mapping.GetIdBySwitch(f.KeyNumber)

			if ok {
				switch key {
				case SpeakerAEnabled,
					SpeakerBEnabled,
					SpeakerCEnabled,
					SpeakerDEnabled,
					SubAEnabled,
					SubBEnabled:
					c.SetSpeakerEnabled(key, !c.speakerEnabled[key])
				default:
					log.Infof("Unknown Button: 0x%X %s", f.KeyNumber, f.HotkeyName)
				}
			}

		case mcu.RawFaderMessage:
			if f.FaderNumber == c.mapping.Master.Mcu.Fader {
				c.SetMasterLevel(f.FaderValue)
				break
			}

		default:
			log.Warnf("Unhandled mcu message %s: %v\n", reflect.TypeOf(fm), fm)

		}
	}
}

func (c *Controller) Reset() {

	for i := 0; i < faderLength; i++ {
		c.SetSpeakerEnabled(i, false)
		c.SetMasterLevel(0)

		t, _ := c.mapping.GetMcuName(i)
		c.setChannelText(i, t, false)
		c.SetMasterMeter(-99.9)
	}

	for i := 0; i < buttonLength; i++ {
		c.setLed(i, c.ledStates[i])
	}
}

func (c *Controller) SetSpeakerEnabled(id int, state bool) {

	if state != c.speakerEnabled[id] {
		ex := c.mapping.Speaker[id].Exclusive
		ty := c.mapping.Speaker[id].Type

		if state {
			// new speaker is exclusice, disable all other
			if ex {
				for k, v := range c.mapping.Speaker {
					if ty == v.Type {
						c.SetSpeakerEnabled(k, false)
					}
				}
			} else { //Check other speakers for eclusive flag
				for k, v := range c.mapping.Speaker {
					if k != id && ty == v.Type && c.speakerEnabled[k] && v.Exclusive {
						c.SetSpeakerEnabled(k, false)
					}
				}
			}
		}

		c.speakerEnabled[id] = state
		c.setLed(id, mcu.Bool2State(state))
		c.fromController <- SpeakerEnabledMessage{SpeakerID: id, SpeakerEnabled: c.speakerEnabled}
	}
}

func (c *Controller) SetMasterLevel(level uint16) {
	if c.masterLevel != level {
		c.masterLevel = level
		c.masterLevelDB = faderdb.FaderToDB(level)
		c.toMcu <- mcu.FaderCommand{Fader: c.mapping.Master.Mcu.Fader, Value: level}
		c.fromController <- c.NewSpeakerLevelMessage()
	}
}

func (c *Controller) setLed(id int, state gomcu.State) {
	btns, err := c.mapping.GetMcuEnabledSwitch(id)
	if err != nil {
		return
	}

	if c.ledStates[id] != state {
		c.ledStates[id] = state
		for _, btn := range btns {
			c.toMcu <- mcu.LedCommand{Led: btn, State: state}
		}

	}
}

func (c *Controller) setDisplayText(text string) {
	if len(text) > 10 {
		text = text[:10]
	} else {
		text = fmt.Sprintf("%-10s", text)
	}

	if c.timeDisplay != text {
		c.timeDisplay = text
		c.toMcu <- mcu.TimeDisplayCommand{Text: text}
	}
}

func (c *Controller) setChannelText(id int, text string, lower bool) {
	if id >= MasterFader {
		return
	}

	c.toMcu <- mcu.ChannelTextCommand{
		Fader:      gomcu.Channel(id),
		Text:       text,
		BottomLine: lower,
	}

}

func (c *Controller) SetMasterMeter(valueDB float64) {
	out := mcu.Db2MeterLevel(valueDB)
	if c.masterMeter != out {
		c.masterMeter = out
		c.toMcu <- mcu.MeterCommand{Channel: c.mapping.Master.Mcu.Fader, Value: gomcu.MeterLevel(out)}

	}
}

func (c *Controller) ToggleMute() {
	c.SetMute(!c.masterMute)
}

func (c *Controller) SetMute(mute bool) {
	if c.masterMute != mute {
		c.masterMute = mute
		for _, v := range MuteButtons {
			c.toMcu <- mcu.LedCommand{Led: v, State: mcu.Bool2State(mute)}
		}
		c.fromController <- MuteMessage{Mute: c.masterMute}
	}
}

func (c *Controller) initDisplay() {
	for k, v := range c.mapping.Speaker {
		c.setChannelText(k, v.Name, false)
	}
	c.setChannelText(MasterFader, c.mapping.Master.Name, false)
}
