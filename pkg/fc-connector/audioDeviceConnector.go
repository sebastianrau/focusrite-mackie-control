package fcaudioconnector

import (
	"reflect"
	"strconv"

	focusriteclient "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-client"
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

var log *logger.CustomLogger = logger.WithPackage("fc-audio")

type AudioDeviceConnector struct {
	device *focusriteclient.FocusriteClient
	config *FcConfiguration

	state        *monitorcontroller.ControllerSate
	toController chan interface{}
}

func NewAudioDeviceConnector(cfg *FcConfiguration) *AudioDeviceConnector {
	ad := &AudioDeviceConnector{
		config: cfg,
		state:  monitorcontroller.NewDefaultState(),
	}

	ad.device = focusriteclient.NewFocusriteClient(focusriteclient.UpdateRaw)

	go ad.run()

	return ad
}

func (ad *AudioDeviceConnector) run() {
	for msg := range ad.device.FromFocusrite {
		switch m := msg.(type) {
		case focusriteclient.DeviceArrivalMessage:
			ad.handleFcDeviceArrivalMsg(focusritexml.Device(m))

		case focusriteclient.DeviceRemovalMessage:
			ad.handleFcDeviceRemovalMsg(int(m))

		case focusriteclient.RawUpdateMessage:
			ad.handleFcUpdateMsg(focusritexml.Set(m))

		case focusriteclient.DeviceUpdateMessage:
			log.Debugf("Got Device Upadte Message")
			ad.toController <- monitorcontroller.AdSetDeviceStatus{
				DeviceId:        m.ID,
				Model:           m.Model,
				SerialNumber:    m.SerialNumber,
				SampleRate:      m.Clocking.SampleRate.Value,
				ConnectionState: ad.device.Connected(),
			}

		case focusriteclient.ApprovalMessasge:
			if m {
				log.Info("got positive approval from Focusrite Controller Server")
			} else {
				log.Warn("app has no approval from Focusrite Control.")
			}
			// TODO reflect on gui

		case focusriteclient.ConnectionStatusMessage:
			if !m { //connection to Fc Control Server lost
				log.Debugf("Connection to Focusrite Device Control Server lost")
				ad.config.FocusriteDeviceId = 0
				ad.toController <- monitorcontroller.AdSetDeviceStatus{
					DeviceId:        0,
					ConnectionState: ad.device.Connected(),
				}
			}

		default:
			log.Warnf("unhadled %s message", reflect.TypeOf(msg).String())
		}

	}

}

func (ad *AudioDeviceConnector) SetControlChannel(controllerChannel chan interface{}) {
	ad.toController = controllerChannel
}

func (ad *AudioDeviceConnector) handleFcDeviceArrivalMsg(device focusritexml.Device) {
	log.Debugf("New Focusrite Device Arrived ID:%d SN:%s", device.ID, device.SerialNumber)
	if device.SerialNumber == ad.config.FocusriteSerialNumber {
		log.Debugf("configured device with SN: %s arrived with ID ID:%d", device.SerialNumber, device.ID)
		ad.config.FocusriteDeviceId = device.ID

		ad.toController <- monitorcontroller.AdSetDeviceStatus{
			DeviceId:        device.ID,
			Model:           device.Model,
			SerialNumber:    device.SerialNumber,
			SampleRate:      device.Clocking.SampleRate.Value,
			ConnectionState: ad.device.Connected(),
		}
		ad.toController <- monitorcontroller.AdUpdateRequest{}
	}
}

func (ad *AudioDeviceConnector) handleFcDeviceRemovalMsg(deviceId int) {
	log.Debugf("Focusrite Device removed ID:%d", deviceId)
	if deviceId != 0 && deviceId == ad.config.FocusriteDeviceId {
		ad.config.FocusriteDeviceId = 0
	}

	ad.toController <- monitorcontroller.AdSetDeviceStatus{
		DeviceId:        0,
		Model:           "",
		SerialNumber:    "",
		ConnectionState: ad.device.Connected(),
	}
}

func (ad *AudioDeviceConnector) handleFcUpdateMsg(set focusritexml.Set) {
	if set.DevID != ad.config.FocusriteDeviceId {
		return
	}

	for _, s := range set.Items {
		fcID := FocusriteId(s.ID)

		if ad.config.Master.DimSwitch == fcID {
			log.Debugf("Found Dim ID %d", s.ID)
			boolValue, err := strconv.ParseBool(s.Value)
			if err == nil {
				ad.toController <- monitorcontroller.AdSetDim(boolValue)
			}
			continue
		}

		if ad.config.Master.MuteSwitch == fcID {
			log.Debugf("Found Mute ID %d: %v", s.ID, s.Value)
			boolValue, err := strconv.ParseBool(s.Value)
			if err == nil {
				ad.toController <- monitorcontroller.AdSetMute(boolValue)
			}
			continue
		}

		for spkId, spk := range ad.config.Speaker {
			if spk.Name == FocusriteId(s.ID) {
				ad.toController <- monitorcontroller.AdSetSpeakerName{Id: spkId, Name: s.Value}
				continue
			}

			// FIXME Getting Backfire from Focusrite Control when setting a device mute or outputgain
			// if spk.Mute == FocusriteId(s.ID) {}
			// if spk.OutputGain == FocusriteId(s.ID) {}
		}

	}

	// Handle Speaker Level separately and use only first speaker selected
	var spkForLevel *SpeakerFcConfig
	for spkId, spk := range ad.config.Speaker {
		if !ad.state.Speaker[spkId].Disabled && ad.state.Speaker[spkId].Selected && ad.state.Speaker[spkId].Type == monitorcontroller.Speaker {
			spkForLevel = spk
			break //break after found one speaker
		}
	}

	if spkForLevel != nil {
		levelL := -127.0
		levelR := -127.0
		var err error

		for _, s := range set.Items {
			if spkForLevel.MeterL == FocusriteId(s.ID) {
				levelL, err = strconv.ParseFloat(s.Value, 64)
				if err != nil {
					log.Error(err.Error())
				}
			}
			if spkForLevel.MeterR == FocusriteId(s.ID) {
				levelR, err = strconv.ParseFloat(s.Value, 64)
				if err != nil {
					log.Error(err.Error())
				}
			}

			ad.setMasterLevel(int(levelL), int(levelR))
		}
	}

}

func (ad *AudioDeviceConnector) HandleSpeakerName(spkId monitorcontroller.SpeakerID, name string) {
	ad.state.Speaker[spkId].Name = name

	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItemString(int(ad.config.Speaker[spkId].Name), name)
	ad.device.ToFocusrite <- *fcUpdateSet
}

func (ad *AudioDeviceConnector) HandleDim(dim bool) {

	log.Debugf("Handling Dim: %t", dim)
	ad.state.Master.Dim = dim

	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(ad.config.Master.DimSwitch), ad.state.Master.Dim)
	fcUpdateSet.AddItems(ad.getSpeakerVolumeUpdateSet().Items)
	ad.device.ToFocusrite <- *fcUpdateSet

}

func (ad *AudioDeviceConnector) HandleMute(mute bool) {
	ad.state.Master.Mute = mute

	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(ad.config.Master.MuteSwitch), mute)
	fcUpdateSet.AddItems(ad.getSpeakerMuteUpdateSet().Items)
	ad.device.ToFocusrite <- *fcUpdateSet
}

func (ad *AudioDeviceConnector) HandleVolume(vol int) {
	ad.state.Master.VolumeDB = vol

	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItems(ad.getSpeakerVolumeUpdateSet().Items)
	ad.device.ToFocusrite <- *fcUpdateSet
}

func (ad *AudioDeviceConnector) HandleMeter(level int) {
	//ignore meter values - nothing to show on device
}

func (ad *AudioDeviceConnector) HandleSpeakerSelect(spkId monitorcontroller.SpeakerID, sel bool) {
	ad.state.Speaker[spkId].Selected = sel

	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItems(ad.getSpeakerMuteUpdateSet().Items)
	ad.device.ToFocusrite <- *fcUpdateSet

}
func (ad *AudioDeviceConnector) HandleSpeakerUpdate(spkId monitorcontroller.SpeakerID, spk *monitorcontroller.SpeakerState) {
	ad.state.Speaker[spkId] = spk

	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItems(ad.getSpeakerMuteUpdateSet().Items)
	fcUpdateSet.AddItems(ad.getSpeakerNameUpdateSet().Items)

	ad.device.ToFocusrite <- *fcUpdateSet

}

func (ad *AudioDeviceConnector) HandleMasterUpdate(master *monitorcontroller.MasterState) {
	ad.state.Master = master
	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItems(ad.getSpeakerMuteUpdateSet().Items)
	fcUpdateSet.AddItems(ad.getSpeakerVolumeUpdateSet().Items)
	fcUpdateSet.AddItemBool(int(ad.config.Master.DimSwitch), ad.state.Master.Dim)
	fcUpdateSet.AddItemBool(int(ad.config.Master.MuteSwitch), ad.state.Master.Mute)
	ad.device.ToFocusrite <- *fcUpdateSet
}

// Setters
func (ad *AudioDeviceConnector) setMasterLevel(levelLeft, levelRight int) {
	ad.state.Master.LevelLeft = levelLeft
	ad.state.Master.LevelRight = levelRight
	ad.toController <- monitorcontroller.AdSetLevel{Left: ad.state.Master.LevelLeft, Right: ad.state.Master.LevelRight}
}

// Fc XNL Set generator functions
func (ad *AudioDeviceConnector) getSpeakerVolumeUpdateSet() *focusritexml.Set {
	volume := int(ad.state.Master.VolumeDB)
	if volume >= 0 {
		volume = 0
	}
	if ad.state.Master.Dim {
		volume = volume - int(ad.state.Master.DimOffset)
	}
	if volume < -127 {
		volume = -127
	}

	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	for spkId, spk := range ad.config.Speaker {
		if !ad.state.Speaker[spkId].Disabled {
			fcUpdateSet.AddItemInt(int(spk.OutputGain), volume)
		}

	}
	log.Debugf("Sending speaker level %d", volume)
	return fcUpdateSet

}

func (ad *AudioDeviceConnector) getSpeakerMuteUpdateSet() *focusritexml.Set {

	mute := ad.state.Master.Mute

	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(ad.config.Master.MuteSwitch), mute)

	// reset level if no speaker is selected
	countEnabledSpeaker := 0
	for _, spk := range ad.state.Speaker {
		if spk.Type == monitorcontroller.Speaker && spk.Selected {
			countEnabledSpeaker++
		}
	}
	if countEnabledSpeaker == 0 {
		ad.setMasterLevel(-127, -127)
	}

	for spkId, spk := range ad.config.Speaker {
		if !ad.state.Speaker[spkId].Disabled {
			state := mute || !ad.state.Speaker[spkId].Selected
			fcUpdateSet.AddItemBool(int(spk.Mute), state)
			log.Debugf("setting focusrite speaker %d Mute to %t", spkId, state)
		}

	}
	return fcUpdateSet
}

func (ad *AudioDeviceConnector) getSpeakerNameUpdateSet() *focusritexml.Set {
	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	for spkId, spk := range ad.config.Speaker {
		name := ad.state.Speaker[spkId].Name
		fcUpdateSet.AddItemString(int(spk.Name), name)
		log.Debugf("setting focusrite speaker %d name to %s", spkId, name)
	}
	return fcUpdateSet
}
