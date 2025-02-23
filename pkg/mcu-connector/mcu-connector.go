package mcuconnector

import (
	"math"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
	"github.com/sebastianrau/gomcu"
)

var log *logger.CustomLogger = logger.WithPackage("mcu-connector")

const LEVEL_RATE_LIMIT_TIME time.Duration = 100 * time.Millisecond

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

	mu                 sync.Mutex
	meterValue         gomcu.MeterLevel
	meterUpdateRequest bool
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
	go m.runSendMeterValues()
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
			if mc.config.MasterVolumeChannel == f.FaderNumber {
				log.Debugf("Channel Select Button detected: %d", f.FaderNumber)
				mc.mcu.ToMcu <- mcu.FaderCommand{Fader: gomcu.Channel(f.FaderNumber), Value: mc.faderValueRaw}
				continue
			}

		case mcu.KeyMessage:
			log.Debugf("Key Msg: %s (%d)", f.HotkeyName, f.KeyNumber)

			if mc.config.MasterMuteSwitch == f.KeyNumber {
				mc.controllerChannel <- monitorcontroller.RcSetMute(!mc.state.Master.Mute)
				continue
			}

			if mc.config.MasterDimSwitch == f.KeyNumber {
				mc.controllerChannel <- monitorcontroller.RcSetDim(!mc.state.Master.Dim)
				continue
			}

			for k, spk := range mc.config.SpeakerSelect {
				if spk == f.KeyNumber {
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
			if mc.config.MasterVolumeChannel == f.FaderNumber {
				db := 0.0
				if mc.config.FaderScaleLog {
					db = FaderToDBLog(f.FaderValue)
				} else {
					db = FaderToDB(f.FaderValue)
				}
				mc.controllerChannel <- monitorcontroller.RcSetVolume(db)
			}

		default:
			log.Warnf("Unhandled mcu message %s: %v\n", reflect.TypeOf(msg), msg)
		}
	}
}

func (mc *McuConnector) runSendMeterValues() {
	for {
		time.Sleep(LEVEL_RATE_LIMIT_TIME)
		mc.mu.Lock()
		if mc.meterUpdateRequest {
			mc.mcu.ToMcu <- mcu.MeterCommand{Channel: mc.config.MasterVolumeChannel, Value: mc.meterValue}
			mc.meterUpdateRequest = false
		}
		mc.mu.Unlock()
	}
}

func (mc *McuConnector) SetControlChannel(controllerChannel chan interface{}) {
	mc.controllerChannel = controllerChannel
}

func (mc *McuConnector) HandleDim(dim bool) {
	mc.state.Master.Dim = dim
	mc.updateMcuLed(mc.config.MasterDimSwitch, mc.state.Master.Dim)
}

func (mc *McuConnector) HandleMute(mute bool) {
	mc.SetMute(mute)
}

func (mc *McuConnector) HandleVolume(db int) {

	if mc.config.FaderScaleLog {
		mc.SetVolume(DBToFaderLog(float64(db)))
	} else {
		mc.SetVolume(DBToFader(float64(db)))
	}

}

func (mc *McuConnector) HandleMeter(left, right int) {
	level := mcu.Db2MeterLevel(math.Max(float64(left), float64(right)))
	mc.updateAllMeterFader(level)
}

func (mc *McuConnector) HandleSpeakerSelect(id monitorcontroller.SpeakerID, sel bool) {
	mc.state.Speaker[id].Selected = sel
	mc.updateMcuLed(mc.config.SpeakerSelect[id], sel)
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

	if mc.config.FaderScaleLog {
		mc.SetVolume(DBToFaderLog(float64(master.VolumeDB)))
	} else {
		mc.SetVolume(DBToFader(float64(master.VolumeDB)))
	}

}

func (mc *McuConnector) HandleDeviceUpdate(dev *monitorcontroller.DeviceInfo) {
	mc.initMcu()
}

//Setter

func (mc *McuConnector) SetMute(mute bool) {
	mc.state.Master.Mute = mute
	mc.updateMcuLed(DefaultConfiguration().MasterMuteSwitch, mc.state.Master.Mute)
}

func (mc *McuConnector) SetDim(dim bool) {
	mc.state.Master.Dim = dim
	mc.updateMcuLed(DefaultConfiguration().MasterDimSwitch, mc.state.Master.Dim)
}

func (mc *McuConnector) SetVolume(vol uint16) {
	mc.faderValueRaw = vol
	mc.updateMcuFader(mc.config.MasterVolumeChannel, mc.faderValueRaw)
}

func (mc *McuConnector) SetSpeakerSelect(id monitorcontroller.SpeakerID, sel bool) {
	mc.state.Speaker[id].Selected = sel
	mc.updateMcuLed(mc.config.SpeakerSelect[id], sel)
}

func (mc *McuConnector) SetSpeakerName(id monitorcontroller.SpeakerID, name string) {
	mc.state.Speaker[id].Name = name
}

func (mc *McuConnector) initMcu() {
	mc.updateMcuLed(mc.config.MasterMuteSwitch, mc.state.Master.Mute)
	mc.updateMcuLed(mc.config.MasterDimSwitch, mc.state.Master.Dim)

	for k, speaker := range mc.config.SpeakerSelect {
		mc.updateMcuLed(speaker, mc.state.Speaker[k].Selected)
	}

	mc.updateMcuFader(mc.config.MasterVolumeChannel, mc.faderValueRaw)
}

// MCU Led & Fader Hacks
func (mc *McuConnector) setLedBool(sw gomcu.Switch, state bool) {
	mc.setLed(sw, mcu.Bool2State(state))
}

func (mc *McuConnector) setLed(sw gomcu.Switch, state gomcu.State) {
	mc.mcu.ToMcu <- mcu.LedCommand{Led: sw, State: state}
}

func (c *McuConnector) updateMcuLed(sw gomcu.Switch, state bool) {
	c.setLedBool(sw, state)
}

func (mc *McuConnector) updateMcuFader(channel gomcu.Channel, value uint16) {
	mc.mcu.ToMcu <- mcu.FaderSelectCommand{Channel: channel, ChnnalValue: value}
	mc.mcu.ToMcu <- mcu.FaderCommand{Fader: channel, Value: value}
}

func (mc *McuConnector) updateAllMeterFader(level gomcu.MeterLevel) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.meterValue = max(mc.meterValue, level)
	mc.meterUpdateRequest = true
}

func (mc *McuConnector) isMcuID(a []gomcu.Switch, k gomcu.Switch) bool {
	return slices.Contains(a, k)
}
