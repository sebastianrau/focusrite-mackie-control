package monitorcontroller

import (
	"fmt"
	"log"
	"reflect"

	"github.com/normen/obs-mcu/gomcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
)

type Controller struct {
	speakerEnabled []bool
	speakerLevel   []uint16
	mute           bool

	meterLevels []gomcu.MeterLevel
	ledStates   []gomcu.State

	timeDisplay string

	mapping *ControllerMapping

	toMcu   chan interface{}
	fromMcu chan interface{}

	fromController chan interface{}
}

// NewMcuState creates a new McuState
func NewController(toMcu chan interface{}, fromMcu chan interface{}, fromController chan interface{}) *Controller {

	state := Controller{
		speakerEnabled: make([]bool, faderLength),
		speakerLevel:   make([]uint16, faderLength),
		meterLevels:    make([]gomcu.MeterLevel, faderLength),
		ledStates:      make([]gomcu.State, buttonLength),
		mute:           false,
		timeDisplay:    "12345670000890",
		mapping:        DefaultMapping(),
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

				//m := ControllerMapping.Master
				//c.mapping.Speaker[]

				c.toMcu <- mcu.LedCommand{Led: gomcu.FaderMaster, State: gomcu.StateOn}
			}

		case mcu.KeyMessage:
			log.Printf("Button: 0x%X %s", f.KeyNumber, f.HotkeyName)

			// GOMCU Raw Messages
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
			default:
				fmt.Println("Test")
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
					log.Printf("Unknown Button: 0x%X %s", f.KeyNumber, f.HotkeyName)
				}
			}

		case mcu.RawFaderMessage:
			fader, ok := c.mapping.GetIdByFader(f.FaderNumber)
			if ok {
				c.SetSpeakerLevel(fader, f.FaderValue)
				break
			}

		default:
			log.Printf("%s: %v\n", reflect.TypeOf(fm), fm)

		}
	}
}

func (c *Controller) Reset() {

	for i := 0; i < faderLength; i++ {
		c.SetSpeakerEnabled(i, false)
		c.SetSpeakerLevel(i, 0)

		t, _ := c.mapping.GetMcuName(i)
		c.setChannelText(i, t, false)
		c.SetMeter(i, -99.9)
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

func (c *Controller) SetSpeakerLevel(id int, level uint16) {
	fader, err := c.mapping.GetMcuFader(id)

	if err != nil {
		return
	}

	if c.speakerLevel[id] != level {
		c.speakerLevel[id] = level
		c.toMcu <- mcu.FaderCommand{Fader: fader, Value: level}
		c.fromController <- SpeakerLevelMessage{SpeakerID: id, SpeakerLevel: c.speakerLevel}
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

func (c *Controller) SetMeter(id int, valueDB float64) {

	out := mcu.Db2MeterLevel(valueDB)

	fader, err := c.mapping.GetMcuFader(id)
	if err != nil {
		return
	}

	fmt.Printf("Setting Meter to %f (%d)\n", valueDB, byte(out))
	if c.meterLevels[id] != out {
		c.meterLevels[id] = out
		c.toMcu <- mcu.MeterCommand{Channel: fader, Value: gomcu.MeterLevel(out)}

	}
}

func (c *Controller) ToggleMute() {
	c.SetMute(!c.mute)
}

func (c *Controller) SetMute(mute bool) {
	if c.mute != mute {
		c.mute = mute
		for _, v := range MuteButtons {
			c.toMcu <- mcu.LedCommand{Led: v, State: mcu.Bool2State(mute)}
		}
		c.fromController <- MuteMessage{Mute: c.mute}
	}
}

func (c *Controller) initDisplay() {
	for k, v := range c.mapping.Speaker {
		c.setChannelText(k, v.Name, false)
	}
	c.setChannelText(MasterFader, c.mapping.Master.Name, false)
}
