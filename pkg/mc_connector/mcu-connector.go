package rcConnector

import (
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

type MCUconnector struct {
	mcu *mcu.Mcu

	toMcu   chan interface{}
	fromMcu chan interface{}

	config *mcu.Configuration
}

func (mc *MCUconnector) SetControlChannel(controllerChannel chan interface{}) {

}

// configured Device Arrived
func (mc *MCUconnector) HandleDeviceArrival(*focusritexml.Device) {

}

// configured Device removed
func (mc *MCUconnector) HandleDeviceRemoval() {

}

// sNew Dim State
func (mc *MCUconnector) HandleDim(bool) {

}

// New Mute State
func (mc *MCUconnector) HandleMute(bool) {

}

// Volume -127 .. 0 dB
func (mc *MCUconnector) HandleVolume(int) {

}

// Meter Value in DB
func (mc *MCUconnector) HandleMeter(int) {

}

// Speaker with given ID new selection State
func (mc *MCUconnector) HandleSpeakerSelect(monitorcontroller.SpeakerID, bool) {

}

/*
func (mc *MCUconnector) handleMcu(msg interface{}) {
	switch f := msg.(type) {

	case mcu.ConnectionMessage:
		if f.Connection {
			c.initMcu()
		}

	case mcu.SelectMessage:
		if slices.Contains(c.config.Master.VolumeMcuChannel, f.FaderNumber) {
			log.Debugf("Channel Select Button detected: %d", f.FaderNumber)
			c.toMcu <- mcu.FaderCommand{Fader: gomcu.Channel(f.FaderNumber), Value: c.config.Master.VolumeMcuRaw}
		}

	case mcu.KeyMessage:
		log.Debugf("Key Msg: %s (%d)", f.HotkeyName, f.KeyNumber)
		if c.config.Master.MuteSwitch.IsMcuID(f.KeyNumber) {
			c.setMute(!c.config.Master.MuteSwitch.Value)
			return
		}

		if c.config.Master.DimSwitch.IsMcuID(f.KeyNumber) {
			c.setDim(!c.config.Master.DimSwitch.Value)
			return
		}

		for k, spk := range c.config.Speaker {
			if spk.Mute.IsMcuID(f.KeyNumber) {
				log.Debugf("Speaker Select Button %s detected. SpeakerId %d ", f.HotkeyName, k)
				c.toggleSpeakerEnabled(k)
				return
			}
		}

		switch f.KeyNumber {
		case gomcu.Play,
			gomcu.Stop,
			gomcu.FastFwd,
			gomcu.Rewind:
			c.FromController <- TransportMessage(f.KeyNumber)
		}
		log.Infof("Unknown Button: 0x%X %s", f.KeyNumber, f.HotkeyName)

	case mcu.RawFaderMessage:
		if slices.Contains(c.config.Master.VolumeMcuChannel, f.FaderNumber) {
			c.setMasterVolumeRawValue(f.FaderValue)
		}

	default:
		log.Warnf("Unhandled mcu message %s: %v\n", reflect.TypeOf(msg), msg)

	}
}
*/
