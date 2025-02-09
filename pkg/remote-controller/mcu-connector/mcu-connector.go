package mcuconnector

import (
	"reflect"
	"slices"

	"github.com/go-vgo/robotgo"
	faderdb "github.com/sebastianrau/focusrite-mackie-control/pkg/faderDB"
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
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

	dim           bool
	mute          bool
	faderValueRaw uint16
	speakerSelect []bool
}

func NewMcuConnector(config *McuConnectorConfig) *McuConnector {
	m := &McuConnector{
		config:        config,
		speakerSelect: make([]bool, monitorcontroller.SPEAKER_LEN),
	}
	m.mcu = mcu.InitMcu(&mcu.Configuration{MidiInputPort: config.MidiInputPort, MidiOutputPort: config.MidiOutputPort})

	go m.run()
	return m
}

func (mc *McuConnector) run() {
	for msg := range mc.mcu.FromMcu {
		switch f := msg.(type) {

		case mcu.ConnectionMessage:
			if f.Connection {
				mc.initMcu()
			}

		case mcu.SelectMessage:
			if slices.Contains(mc.config.MasterVolumeChannel, f.FaderNumber) {
				log.Debugf("Channel Select Button detected: %d", f.FaderNumber)
				mc.mcu.ToMcu <- mcu.FaderCommand{Fader: gomcu.Channel(f.FaderNumber), Value: mc.faderValueRaw}
			}

		case mcu.KeyMessage:
			log.Debugf("Key Msg: %s (%d)", f.HotkeyName, f.KeyNumber)

			if mc.isMcuID(mc.config.MasterMuteSwitch, f.KeyNumber) {
				mc.controllerChannel <- monitorcontroller.RcSetMute(!mc.mute)
				return
			}

			if mc.isMcuID(mc.config.MasterDimSwitch, f.KeyNumber) {
				mc.controllerChannel <- monitorcontroller.RcSetDim(!mc.dim)
				return
			}

			for k, spk := range mc.config.SpeakerSelect {
				if mc.isMcuID(spk, f.KeyNumber) {
					log.Debugf("Speaker Select Button %s detected. SpeakerId %d ", f.HotkeyName, k)
					mc.controllerChannel <- monitorcontroller.RcSpeakerSelect{Id: k, State: !mc.speakerSelect[k]}
					return
				}
			}

			// TODO maybe move to monitorController ?
			switch f.KeyNumber {
			case gomcu.Play:
				err := robotgo.KeyTap(robotgo.AudioPlay)
				if err != nil {
					log.Errorf("Keytab error %s", err.Error())
				}
			case gomcu.FastFwd:
				err := robotgo.KeyTap(robotgo.AudioNext)
				if err != nil {
					log.Errorf("Keytab error %s", err.Error())
				}
			case gomcu.Rewind:
				err := robotgo.KeyTap(robotgo.AudioPrev)
				if err != nil {
					log.Errorf("Keytab error %s", err.Error())
				}

				log.Infof("Unknown Button: 0x%X %s", f.KeyNumber, f.HotkeyName)
			}

		case mcu.RawFaderMessage:
			if slices.Contains(mc.config.MasterVolumeChannel, f.FaderNumber) {
				db := faderdb.FaderToDB(f.FaderValue)
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

func (mc *McuConnector) HandleDeviceArrival(*focusritexml.Device) {
	// ignore
}

func (mc *McuConnector) HandleDeviceRemoval() {
	// ignore
}

func (mc *McuConnector) HandleDim(dim bool) {
	mc.dim = dim
	mc.updateAllLeds(mc.config.MasterDimSwitch, mc.dim)
}

func (mc *McuConnector) HandleMute(mute bool) {
	mc.mute = mute
	mc.updateAllLeds(mc.config.MasterMuteSwitch, mc.mute)
}

func (mc *McuConnector) HandleVolume(db int) {
	mc.faderValueRaw = faderdb.DBToFader(float64(db))
	mc.updateAllFader(mc.config.MasterVolumeChannel, mc.faderValueRaw)
}

func (mc *McuConnector) HandleMeter(db int) {
	level := mcu.Db2MeterLevel(float64(db))
	mc.updateAllMeterFader(mc.config.MasterVolumeChannel, level)
}

func (mc *McuConnector) HandleSpeakerSelect(id monitorcontroller.SpeakerID, sel bool) {
	mc.speakerSelect[id] = sel
	mc.updateAllLeds(mc.config.SpeakerSelect[id], sel)
}

func (mc *McuConnector) initMcu() {
	mc.updateAllLeds(mc.config.MasterMuteSwitch, mc.mute)
	mc.updateAllLeds(mc.config.MasterDimSwitch, mc.dim)

	for k, speaker := range mc.config.SpeakerSelect {
		mc.updateAllLeds(speaker, mc.speakerSelect[k])
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
