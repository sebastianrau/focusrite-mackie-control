package fcaudioconnector

import (
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

	state        monitorcontroller.ControllerSate
	toController chan interface{}
}

func NewAudioDeviceConnector(cfg *FcConfiguration) *AudioDeviceConnector {
	ad := &AudioDeviceConnector{
		config: DefaultConfiguration(), //TODO
		state:  *monitorcontroller.NewDefaultState(),
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
		ad.toController <- monitorcontroller.AdUpdateRequest{}
	}
}

func (ad *AudioDeviceConnector) handleFcDeviceRemovalMsg(deviceId int) {
	log.Debugf("Focusrite Device removed ID:%d", deviceId)
	if deviceId != 0 && deviceId == ad.config.FocusriteDeviceId {
		ad.config.FocusriteDeviceId = 0
	}
}

func (ad *AudioDeviceConnector) handleFcUpdateMsg(set focusritexml.Set) {
	if set.DevID != ad.config.FocusriteDeviceId {
		return
	}

	// log.Debugf("New Raw Update Device Arrived. Items: %d", len(set.Items))
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

		// Handle Speaker Level separately and use only first speaker selected
		for spkId, spk := range ad.config.Speaker {
			if spk.Meter == FocusriteId(s.ID) {
				if ad.state.Speaker[spkId].Selected && ad.state.Speaker[spkId].Type == monitorcontroller.Speaker {
					level, err := strconv.ParseFloat(s.Value, 32)
					if err != nil {
						log.Error(err.Error())
					}
					ad.setMasterLevel(int(level))
					continue
				}
			}
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
	// TODO Check names
	ad.device.ToFocusrite <- *fcUpdateSet

}
func (ad *AudioDeviceConnector) HandleMasterUpdate(master *monitorcontroller.MasterState) {
	ad.state.Master = master
	fcUpdateSet := focusritexml.NewSet(ad.config.FocusriteDeviceId)
	fcUpdateSet.AddItems(ad.getSpeakerMuteUpdateSet().Items)
	fcUpdateSet.AddItemBool(int(ad.config.Master.DimSwitch), ad.state.Master.Dim)
	fcUpdateSet.AddItemBool(int(ad.config.Master.MuteSwitch), ad.state.Master.Mute)
	ad.device.ToFocusrite <- *fcUpdateSet
}

// Setters
func (ad *AudioDeviceConnector) setMasterLevel(level int) {
	ad.state.Master.Level = level
	ad.toController <- monitorcontroller.AdSetLevel(ad.state.Master.Level)
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
	for _, spk := range ad.config.Speaker {
		fcUpdateSet.AddItemInt(int(spk.OutputGain), volume)
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
		ad.setMasterLevel(-127)
	}

	for id, spk := range ad.config.Speaker {
		state := mute || !ad.state.Speaker[id].Selected
		fcUpdateSet.AddItemBool(int(spk.Mute), state)
		log.Debugf("setting focusrite speaker %d Mute to %t", id, state)
	}
	return fcUpdateSet

}
