package monitorcontroller

// TODO: remove Fader fpr Speaker Level

import (
	"strconv"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/focusriteclient"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
)

// TODO Add Meter Functions
// TODO Add Name Functions & Set Display Text

var log *logger.CustomLogger = logger.WithPackage("monitor-controller")

type Controller struct {
	config *FcConfiguration
	state  *ControllerSate

	fc *focusriteclient.FocusriteClient

	fromRemoteController chan interface{}
	remoteController     []RemoteController
}

// NewMcuState creates a new McuState
func NewController(config *FcConfiguration) *Controller {
	c := &Controller{
		state:                NewDefaultState(),
		config:               DefaultConfiguration(),
		fromRemoteController: make(chan interface{}, 100),
		remoteController:     make([]RemoteController, 0),
		fc:                   focusriteclient.NewFocusriteClient(focusriteclient.UpdateRaw),
	}

	go c.run()
	return c
}

func (c *Controller) run() {
	for {
		select {
		case msg := <-c.fc.FromFocusrite:
			switch m := msg.(type) {
			case focusriteclient.DeviceArrivalMessage:
				c.handleFcDeviceArrivalMsg(focusritexml.Device(m))

			case focusriteclient.DeviceRemovalMessage:
				c.handleFcDeviceRemovalMsg(int(m))

			case focusriteclient.RawUpdateMessage:
				c.handleFcUpdateMsg(focusritexml.Set(m))

				// TODO transport Approval State
				//case focusriteclient.ApprovalMessasge: //inoring approval message from fc
				//case focusriteclient.ConnectionStatusMessage: //ignoring connection status Messages from fc
				//case focusriteclient.DeviceUpdateMessage: //ignoring high level updates from fc
			}

		case remote := <-c.fromRemoteController:
			c.handleRemoteControl(remote)
		}

	}
}

func (c *Controller) handleFcUpdateMsg(set focusritexml.Set) {
	if set.DevID != c.config.FocusriteDeviceId {
		return
	}

	// log.Debugf("New Raw Update Device Arrived. Items: %d", len(set.Items))
	for _, s := range set.Items {
		fcID := FocusriteId(s.ID)
		if c.config.Master.DimSwitch == fcID {
			log.Debugf("Found Dim ID %d", s.ID)
			boolValue, err := strconv.ParseBool(s.Value)
			if err != nil {
				return
			}
			c.setDim(boolValue)
			return
		}

		if c.config.Master.MuteSwitch == fcID {
			log.Debugf("Found Mute ID %d: %v", s.ID, s.Value)
			boolValue, err := strconv.ParseBool(s.Value)
			if err != nil {
				log.Error(err.Error())
				return
			}
			c.setMute(boolValue)
			return
		}

		for spkId, spk := range c.config.Speaker {
			if spk.Name == FocusriteId(s.ID) {
				c.setSpeakerName(spkId, s.Value)
			}

			if spk.Meter == FocusriteId(s.ID) {
				if c.state.Speaker[spkId].Selected && c.state.Speaker[spkId].Type == Speaker {
					level, err := strconv.ParseFloat(s.Value, 32)
					if err != nil {
						log.Error(err.Error())
					}
					c.setMasterLevel(int(level))
				}
			}

			/* Getting Backfire from Focusrite Control when setting a device mute or outputgain
			if spk.Mute == FocusriteId(s.ID) {
			}

			if spk.OutputGain == FocusriteId(s.ID) {
			}
			*/

		}
	}
}

func (c *Controller) handleFcDeviceArrivalMsg(device focusritexml.Device) {
	log.Debugf("New Focusrite Device Arrived ID:%d SN:%s", device.ID, device.SerialNumber)
	if device.SerialNumber == c.config.FocusriteSerialNumber {
		c.config.FocusriteDeviceId = device.ID
		c.initFocusriteDevice()
		log.Debugf("configured device with SN: %s arrived with ID ID:%d", device.SerialNumber, device.ID)
	}
}

func (c *Controller) handleFcDeviceRemovalMsg(deviceId int) {
	log.Debugf("Focusrite Device removed ID:%d", deviceId)
	if deviceId != 0 && deviceId == c.config.FocusriteDeviceId {
		c.config.FocusriteDeviceId = 0
	}
}

// Remote Controls
func (c *Controller) RegisterRemoteController(r RemoteController) *Controller {
	c.remoteController = append(c.remoteController, r)
	r.SetControlChannel(c.fromRemoteController)
	c.fireAllUpdate()
	return c
}
func (c *Controller) handleRemoteControl(remote interface{}) {
	switch r := remote.(type) {
	case RcUpdateRequest:
		log.Warn("RcUpdate not handled")
	case RcSetMute:
		c.setMute(bool(r))
	case RcSetDim:
		c.setDim(bool(r))
	case RcSetVolume:
		c.setMasterVolumeDB(int(r))
	case RcSpeakerSelect:
		c.setSpeakerSelected(r.Id, r.State)
	}
}

func (c *Controller) fireDeviceArrival(dev *focusritexml.Device) {
	for _, rc := range c.remoteController {
		go rc.HandleDeviceArrival(dev)
	}
}
func (c *Controller) fireDeviceRemoval() {
	for _, rc := range c.remoteController {
		go rc.HandleDeviceRemoval()
	}
}
func (c *Controller) fireDim() {
	for _, rc := range c.remoteController {
		go rc.HandleDim(c.state.Master.Dim)
	}
}
func (c *Controller) fireMute() {
	for _, rc := range c.remoteController {
		go rc.HandleMute(c.state.Master.Mute)
	}
}
func (c *Controller) fireVolume() {
	for _, rc := range c.remoteController {
		go rc.HandleVolume(c.state.Master.VolumeDB)
	}
}
func (c *Controller) fireLevel() {
	for _, rc := range c.remoteController {
		go rc.HandleMeter(c.state.Master.Level)
	}
}
func (c *Controller) fireSpeakerSelect(id SpeakerID) {
	for _, rc := range c.remoteController {
		go rc.HandleSpeakerSelect(id, c.state.Speaker[id].Selected)
	}
}
func (c *Controller) fireSpeakerUpdate(id SpeakerID) {
	for _, rc := range c.remoteController {
		go rc.HandleSpeakerUpdate(id, c.state.Speaker[id])
	}

}
func (c *Controller) fireMasterUpdate(master *MasterState) {
	for _, rc := range c.remoteController {
		go rc.HandleMasterUpdate(master)
	}
}
func (c *Controller) fireAllUpdate() {
	for spkId, spk := range c.state.Speaker {
		for _, rc := range c.remoteController {
			go rc.HandleSpeakerUpdate(spkId, spk)
		}
	}
	for _, rc := range c.remoteController {
		go rc.HandleMasterUpdate(c.state.Master)
	}

}

// Mute function
func (c *Controller) setMute(mute bool) {
	if c.state.Master.Mute == mute {
		log.Debugf("Set Mute: %t, but no change", mute)
		return
	}

	c.state.Master.Mute = mute

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(c.config.Master.MuteSwitch), mute)
	c.fc.ToFocusrite <- *fcUpdateSet

	c.updateFcSpeakerMute()
	c.fireMute()
}

// Dim function
func (c *Controller) setDim(dim bool) {
	if c.state.Master.Dim == dim {
		log.Debugf("Set Dim: %t, but no change", dim)
		return
	}

	c.state.Master.Dim = dim

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(c.config.Master.DimSwitch), c.state.Master.Dim)
	c.fc.ToFocusrite <- *fcUpdateSet

	c.updateFcSpeakerVolume()
	c.fireDim()

}

// setter function
func (c *Controller) setSpeakerSelected(id SpeakerID, sel bool) {
	speaker, ok := c.state.Speaker[id]
	if !ok {
		log.Warnf("No speaker to select: %d", id)
		return
	}

	if speaker.Selected == sel {
		log.Debugf("Set Speaker %d Enable: %t, but no change needed", id, sel)
		return
	}

	speaker.Selected = sel

	if speaker.Selected {
		//if selected speaker is exclusive, disable all others with same type
		if speaker.Exclusive {
			for spkId, spk := range c.state.Speaker {
				if speaker.Type == spk.Type && id != spkId && spk.Selected {
					c.setSpeakerSelected(spkId, false)
				}
			}
		} else {
			//Check if other speakers set set to eclusive and must be deselected
			for spkId, spk := range c.state.Speaker {
				if speaker.Type == spk.Type && id != spkId && spk.Exclusive && !spk.Selected {
					c.setSpeakerSelected(spkId, false)
				}
			}
		}
	}

	c.fireSpeakerSelect(id)
	c.updateFcSpeakerMute()

}
func (c *Controller) setSpeakerName(id SpeakerID, name string) {
	speaker, ok := c.state.Speaker[id]
	if !ok {
		log.Warnf("No speaker to select: %d", id)
		return
	}

	if speaker.Name == name {
		log.Debugf("Set Speaker %d name: %s, but no change needed", id, name)
		return
	}

	speaker.Name = name

	c.fireSpeakerUpdate(id)

}

func (c *Controller) setMasterVolumeDB(vol int) {
	if c.state.Master.VolumeDB == vol {
		return
	}
	c.state.Master.VolumeDB = vol

	c.fireVolume()
	c.updateFcSpeakerVolume()
}
func (c *Controller) setMasterLevel(vol int) {
	if c.state.Master.Level == vol {
		return
	}
	c.state.Master.Level = vol
	c.fireLevel()
}

// Fc Update Functions
func (c *Controller) updateFcSpeakerVolume() {
	volume := int(c.state.Master.VolumeDB)
	if volume >= 0 {
		volume = 0
	}
	if c.state.Master.Dim {
		volume = volume - int(c.state.Master.DimOffset)
	}
	if volume < -127 {
		volume = -127
	}

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	for _, spk := range c.config.Speaker {
		fcUpdateSet.AddItemInt(int(spk.OutputGain), volume)
	}
	c.fc.ToFocusrite <- *fcUpdateSet
	log.Debugf("Sending speaker level %d", volume)
}

func (c *Controller) updateFcSpeakerMute() {

	mute := c.state.Master.Mute

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(c.config.Master.MuteSwitch), mute)

	for id, spk := range c.config.Speaker {
		state := mute || !c.state.Speaker[id].Selected
		fcUpdateSet.AddItemBool(int(spk.Mute), state)
		log.Debugf("setting focusrite speaker %d Mute to %t", id, state)
	}
	c.fc.ToFocusrite <- *fcUpdateSet

	countEnabledSpeaker := 0
	for _, spk := range c.state.Speaker {
		if spk.Type == Speaker && spk.Selected {
			countEnabledSpeaker++
		}
	}
	if countEnabledSpeaker == 0 {
		c.setMasterLevel(-127)
	}

}
func (c *Controller) initFocusriteDevice() {
	updateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	updateSet.AddItemBool(int(c.config.Master.MuteSwitch), c.state.Master.Mute)
	updateSet.AddItemBool(int(c.config.Master.DimSwitch), c.state.Master.Dim)
	c.fc.ToFocusrite <- *updateSet

	c.updateFcSpeakerVolume()
	c.updateFcSpeakerMute()

}
