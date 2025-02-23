package monitorcontroller

import (
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
)

var log *logger.CustomLogger = logger.WithPackage("monitor-controller")

type Controller struct {
	state *ControllerSate

	fromAudioInterface chan interface{}
	audioDevice        AudioDevice

	fromRemoteController chan interface{}
	remoteController     []RemoteController
}

// NewMcuState creates a new McuState
func NewController(audioDevice AudioDevice, config *ControllerSate) *Controller {
	c := &Controller{
		state: config,

		fromAudioInterface: make(chan interface{}, 100),
		audioDevice:        audioDevice,

		fromRemoteController: make(chan interface{}, 100),
		remoteController:     make([]RemoteController, 0),
	}

	if c.audioDevice == nil {
		return nil
	}

	c.audioDevice.SetControlChannel(c.fromAudioInterface)

	go c.run()
	return c
}

func (c *Controller) run() {
	for {
		select {

		case remote := <-c.fromAudioInterface:
			switch r := remote.(type) {
			case AdUpdateRequest:
				c.audioDevice.HandleMasterUpdate(c.state.Master)
				for spkId, spk := range c.state.Speaker {
					c.audioDevice.HandleSpeakerUpdate(spkId, spk)
				}
			case AdSetMute:
				log.Debugf("setting mute: %t", bool(r))
				c.setMute(bool(r))
			case AdSetDim:
				log.Debugf("setting dim: %t", bool(r))
				c.setDim(bool(r))
			case AdSetVolume:
				c.setMasterVolumeDB(int(r))
			case AdSetSpeakerName:
				c.setSpeakerName(r.Id, r.Name)
			case AdSpeakerSelect:
				c.setSpeakerSelected(r.Id, r.State)
			case AdSetLevel:
				c.setMasterLevel(r.Left, r.Right)
			case AdSetDeviceStatus:
				c.fireDeviceUpdate(r)

			}

		case remote := <-c.fromRemoteController:
			switch r := remote.(type) {
			case RcUpdateRequest:
				c.fireMasterUpdate(c.state.Master)
				for spkId := range c.state.Speaker {
					c.fireSpeakerUpdate(spkId)
				}
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

	}
}

// Remote Controls
func (c *Controller) RegisterRemoteController(r RemoteController) *Controller {
	c.remoteController = append(c.remoteController, r)
	r.SetControlChannel(c.fromRemoteController)
	c.fireAllUpdate()
	return c
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
		go rc.HandleMeter(c.state.Master.LevelLeft, c.state.Master.LevelRight)
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

func (c *Controller) fireDeviceUpdate(status AdSetDeviceStatus) {

	dev := &DeviceInfo{
		DeviceId:        status.DeviceId,
		SerialNumber:    status.SerialNumber,
		Model:           status.Model,
		ConnectionState: status.ConnectionState,
	}
	for _, rc := range c.remoteController {
		go rc.HandleDeviceUpdate(dev)
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

//Functional Methods

func (c *Controller) setMute(mute bool) {
	if c.state.Master.Mute == mute {
		log.Debugf("Set Mute: %t, but no change", mute)
		return
	}

	c.state.Master.Mute = mute
	c.audioDevice.HandleMute(mute)
	c.fireMute()
}

func (c *Controller) setDim(dim bool) {
	if c.state.Master.Dim == dim {
		log.Debugf("Set Dim: %t, but no change", dim)
		return
	}
	c.state.Master.Dim = dim

	c.audioDevice.HandleDim(dim)
	c.fireDim()
}

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

	if speaker.Disabled {
		log.Debugf("Speaker is disabled. no action required")
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
				if speaker.Type == spk.Type && id != spkId && spk.Exclusive && spk.Selected {
					c.setSpeakerSelected(spkId, false)
				}
			}
		}
	}

	c.audioDevice.HandleSpeakerSelect(id, speaker.Selected)
	c.fireSpeakerSelect(id)

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

	if speaker.Disabled {
		log.Debugf("Speaker is disabled. no action required")
		return
	}
	speaker.Name = name

	c.audioDevice.HandleSpeakerUpdate(id, speaker)
	c.fireSpeakerUpdate(id)
}

func (c *Controller) setMasterVolumeDB(vol int) {
	if c.state.Master.VolumeDB == vol {
		return
	}
	c.state.Master.VolumeDB = vol

	c.audioDevice.HandleVolume(vol)
	c.fireVolume()
}

func (c *Controller) setMasterLevel(left, right int) {
	c.state.Master.LevelLeft = left
	c.state.Master.LevelRight = right
	c.fireLevel()
}
