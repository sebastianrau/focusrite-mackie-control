package mcuconnector

import (
	"math"
	"reflect"
	"slices"

	"github.com/go-vgo/robotgo"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
	"github.com/sebastianrau/gomcu"
)

var log *logger.CustomLogger = logger.WithPackage("mcu-connector")

type McuConnector struct {
	mcu    *mcu.Mcu
	config *McuConnectorConfig

	controllerChannel chan interface{}

	state *monitorcontroller.ControllerSate

	//dim           bool
	//mute          bool
	faderValueRaw uint16
	//speakerSelect []bool
	//speakerName   []string
}

func NewMcuConnector(config *McuConnectorConfig) *McuConnector {
	m := &McuConnector{
		config: config,
		state:  monitorcontroller.NewDefaultState(),
		//		speakerSelect: make([]bool, monitorcontroller.SPEAKER_LEN),
		//		speakerName:   make([]string, monitorcontroller.SPEAKER_LEN),
	}

	var err error
	m.mcu, err = mcu.InitMcu(&mcu.Configuration{MidiInputPort: config.MidiInputPort, MidiOutputPort: config.MidiOutputPort})
	if err != nil {
		return nil
	}

	go m.run()
	return m
}

func (mc *McuConnector) run() {
	for msg := range mc.mcu.FromMcu {
		switch f := msg.(type) {

		case mcu.ConnectionMessage:
			if f.Connection {
				mc.initMcu()
				continue
			}

		case mcu.SelectMessage:
			if slices.Contains(mc.config.MasterVolumeChannel, f.FaderNumber) {
				log.Debugf("Channel Select Button detected: %d", f.FaderNumber)
				mc.mcu.ToMcu <- mcu.FaderCommand{Fader: gomcu.Channel(f.FaderNumber), Value: mc.faderValueRaw}
				continue
			}

		case mcu.KeyMessage:
			log.Debugf("Key Msg: %s (%d)", f.HotkeyName, f.KeyNumber)

			if mc.isMcuID(mc.config.MasterMuteSwitch, f.KeyNumber) {
				mc.controllerChannel <- monitorcontroller.RcSetMute(!mc.state.Master.Mute)
				continue
			}

			if mc.isMcuID(mc.config.MasterDimSwitch, f.KeyNumber) {
				mc.controllerChannel <- monitorcontroller.RcSetDim(!mc.state.Master.Dim)
				continue
			}

			for k, spk := range mc.config.SpeakerSelect {
				if mc.isMcuID(spk, f.KeyNumber) {
					log.Debugf("Speaker Select Button %s detected. SpeakerId %d ", f.HotkeyName, k)
					mc.controllerChannel <- monitorcontroller.RcSpeakerSelect{Id: k, State: !mc.state.Speaker[k].Selected}
					continue
				}
			}

			switch f.KeyNumber {
			case gomcu.Play:
				err := robotgo.KeyTap(robotgo.AudioPlay)
				if err != nil {
					log.Errorf("Keytab error %s", err.Error())
				}
				continue
			case gomcu.FastFwd:
				err := robotgo.KeyTap(robotgo.AudioNext)
				if err != nil {
					log.Errorf("Keytab error %s", err.Error())
				}
				continue
			case gomcu.Rewind:
				err := robotgo.KeyTap(robotgo.AudioPrev)
				if err != nil {
					log.Errorf("Keytab error %s", err.Error())
				}
				continue
			}

			log.Infof("Unknown Button: 0x%X %s", f.KeyNumber, f.HotkeyName)

		case mcu.RawFaderMessage:
			if slices.Contains(mc.config.MasterVolumeChannel, f.FaderNumber) {
				db := FaderToDB(f.FaderValue)
				mc.controllerChannel <- monitorcontroller.RcSetVolume(db)
			}

		default:
			log.Warnf("Unhandled mcu message %s: %v\n", reflect.TypeOf(msg), msg)

		}
	}
}

func (mc *McuConnector) SetControlChannel(controllerChannel chan interface{}) {
	mc.controllerChannel = controllerChannel
}

func (mc *McuConnector) HandleDim(dim bool) {
	mc.state.Master.Dim = dim
	mc.updateAllLeds(mc.config.MasterDimSwitch, mc.state.Master.Dim)
}

func (mc *McuConnector) HandleMute(mute bool) {
	mc.SetMute(mute)
}

func (mc *McuConnector) HandleVolume(db int) {
	mc.SetVolume(DBToFader(float64(db)))
}

func (mc *McuConnector) HandleMeter(left, right int) {
	level := mcu.Db2MeterLevel(math.Max(float64(left), float64(right)))
	mc.updateAllMeterFader(mc.config.MasterVolumeChannel, level)
}

func (mc *McuConnector) HandleSpeakerSelect(id monitorcontroller.SpeakerID, sel bool) {
	mc.state.Speaker[id].Selected = sel
	mc.updateAllLeds(mc.config.SpeakerSelect[id], sel)
}

func (mc *McuConnector) HandleSpeakerName(id monitorcontroller.SpeakerID, name string) {
	mc.SetSpeakerName(id, name)
}

func (mc *McuConnector) HandleSpeakerUpdate(id monitorcontroller.SpeakerID, spk *monitorcontroller.SpeakerState) {
	mc.SetSpeakerSelect(id, spk.Selected)
	mc.SetSpeakerName(id, spk.Name)
}

func (mc *McuConnector) HandleMasterUpdate(master *monitorcontroller.MasterState) {
	mc.SetMute(master.Mute)
	mc.SetDim(master.Dim)
	mc.SetVolume(DBToFader(float64(master.VolumeDB)))
}

//Setter

func (mc *McuConnector) SetMute(mute bool) {
	mc.state.Master.Mute = mute
	mc.updateAllLeds(DefaultConfiguration().MasterMuteSwitch, mc.state.Master.Mute)
}

func (mc *McuConnector) SetDim(dim bool) {
	mc.state.Master.Dim = dim
	mc.updateAllLeds(DefaultConfiguration().MasterDimSwitch, mc.state.Master.Dim)
}

func (mc *McuConnector) SetVolume(vol uint16) {
	mc.faderValueRaw = vol
	mc.updateAllFader(mc.config.MasterVolumeChannel, mc.faderValueRaw)
}

func (mc *McuConnector) SetSpeakerSelect(id monitorcontroller.SpeakerID, sel bool) {
	mc.state.Speaker[id].Selected = sel
	mc.updateAllLeds(mc.config.SpeakerSelect[id], sel)
}

func (mc *McuConnector) SetSpeakerName(id monitorcontroller.SpeakerID, name string) {
	mc.state.Speaker[id].Name = name
	// TODO Update LCD
}

func (mc *McuConnector) initMcu() {
	mc.updateAllLeds(mc.config.MasterMuteSwitch, mc.state.Master.Mute)
	mc.updateAllLeds(mc.config.MasterDimSwitch, mc.state.Master.Dim)

	for k, speaker := range mc.config.SpeakerSelect {
		mc.updateAllLeds(speaker, mc.state.Speaker[k].Selected)
	}

	mc.updateAllFader(mc.config.MasterVolumeChannel, mc.faderValueRaw)
}

// MCU Led & Fader Hacks
func (mc *McuConnector) setLedBool(sw gomcu.Switch, state bool) {
	mc.setLed(sw, mcu.Bool2State(state))
}

func (mc *McuConnector) setLed(sw gomcu.Switch, state gomcu.State) {
	mc.mcu.ToMcu <- mcu.LedCommand{Led: sw, State: state}
}

func (c *McuConnector) updateAllLeds(switches []gomcu.Switch, state bool) {
	for _, led := range switches {
		c.setLedBool(led, state)
	}
}

func (mc *McuConnector) updateAllFader(channel []gomcu.Channel, value uint16) {
	for _, fader := range channel {
		mc.mcu.ToMcu <- mcu.FaderSelectCommand{Channel: gomcu.Channel(fader), ChnnalValue: value}
		mc.mcu.ToMcu <- mcu.FaderCommand{Fader: gomcu.Channel(fader), Value: value}
	}
}

func (mc *McuConnector) updateAllMeterFader(channel []gomcu.Channel, level gomcu.MeterLevel) {
	for _, fader := range channel {
		mc.mcu.ToMcu <- mcu.MeterCommand{Channel: fader, Value: level}
	}
}

func (mc *McuConnector) isMcuID(a []gomcu.Switch, k gomcu.Switch) bool {
	return slices.Contains(a, k)
}
