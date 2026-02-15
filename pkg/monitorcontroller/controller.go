package monitorcontroller

import (
	"errors"
	"sync"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
)

var log *logger.CustomLogger = logger.WithPackage("monitor-controller")

type Controller struct {
	mu    sync.RWMutex
	state *ControllerSate

	fromAudioInterface chan interface{}
	audioDevice        AudioDevice

	fromRemoteController chan interface{}
	remoteController     []RemoteController
}

// snapshotRemotes returns a stable copy of the currently registered remotes.
func (c *Controller) snapshotRemotes() []RemoteController {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]RemoteController(nil), c.remoteController...)
}

func (c *Controller) cloneMaster(m *MasterState) *MasterState {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

func (c *Controller) cloneSpeaker(s *SpeakerState) *SpeakerState {
	if s == nil {
		return nil
	}
	cp := *s
	return &cp
}

// NewMcuState creates a new McuState
func NewController(audioDevice AudioDevice, config *ControllerSate) (*Controller, error) {
	c := &Controller{

		state: config,
		mu:    sync.RWMutex{},

		fromAudioInterface: make(chan interface{}, 100),
		audioDevice:        audioDevice,

		fromRemoteController: make(chan interface{}, 100),
		remoteController:     make([]RemoteController, 0),
	}

	if audioDevice == nil {
		return nil, errors.New("audioDevice is nil")
	}
	if config == nil {
		return nil, errors.New("controller config is nil")
	}

	c.audioDevice.SetControlChannel(c.fromAudioInterface)

	go c.run()
	return c, nil
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
	c.mu.Lock()
	c.remoteController = append(c.remoteController, r)
	c.mu.Unlock()

	r.SetControlChannel(c.fromRemoteController)
	c.fireAllUpdate()

	return c
}

func (c *Controller) fireDim() {
	remotes := c.snapshotRemotes()

	c.mu.RLock()
	dim := c.state.Master.Dim
	c.mu.RUnlock()

	for _, rc := range remotes {
		go rc.HandleDim(dim)
	}
}
func (c *Controller) fireMute() {
	remotes := c.snapshotRemotes()

	c.mu.RLock()
	mute := c.state.Master.Mute
	c.mu.RUnlock()

	for _, rc := range remotes {
		go rc.HandleMute(mute)
	}
}
func (c *Controller) fireVolume() {

	remotes := c.snapshotRemotes()

	c.mu.RLock()
	vol := c.state.Master.VolumeDB
	c.mu.RUnlock()

	for _, rc := range remotes {
		go rc.HandleVolume(vol)
	}
}

func (c *Controller) fireLevel() {

	remotes := c.snapshotRemotes()

	c.mu.RLock()
	levelLeft := c.state.Master.LevelLeft
	levelRight := c.state.Master.LevelRight
	c.mu.RUnlock()

	for _, rc := range remotes {
		go rc.HandleMeter(levelLeft, levelRight)
	}
}

func (c *Controller) fireSpeakerSelect(id SpeakerID) {
	remotes := c.snapshotRemotes()
	c.mu.RLock()
	selId := id
	sel := c.state.Speaker[id].Selected
	c.mu.RUnlock()

	for _, rc := range remotes {
		go rc.HandleSpeakerSelect(selId, sel)
	}
}
func (c *Controller) fireSpeakerUpdate(id SpeakerID) {
	remotes := c.snapshotRemotes()
	c.mu.RLock()
	spk := c.cloneSpeaker(c.state.Speaker[id])
	c.mu.RUnlock()

	for _, rc := range remotes {
		cp := c.cloneSpeaker(spk)
		go rc.HandleSpeakerUpdate(id, cp)
	}
}

func (c *Controller) fireMasterUpdate(master *MasterState) {

	remotes := c.snapshotRemotes()
	c.mu.RLock()
	mst := c.cloneMaster(c.state.Master)
	c.mu.RUnlock()

	for _, rc := range remotes {
		m := c.cloneMaster(mst)
		go rc.HandleMasterUpdate(m)
	}
}

func (c *Controller) fireDeviceUpdate(status AdSetDeviceStatus) {
	dev := &DeviceInfo{
		DeviceId:        status.DeviceId,
		SerialNumber:    status.SerialNumber,
		Model:           status.Model,
		ConnectionState: status.ConnectionState,
	}

	remotes := c.snapshotRemotes()
	for _, rc := range remotes {
		cp := dev
		go rc.HandleDeviceUpdate(cp)
	}

}

func (c *Controller) fireAllUpdate() {
	remotes := c.snapshotRemotes()
	c.mu.RLock()

	// snapshot all speakers
	type sp struct {
		id    SpeakerID
		state *SpeakerState
	}

	speakers := make([]sp, 0, len(c.state.Speaker))
	for id, s := range c.state.Speaker {
		speakers = append(speakers, sp{id: id, state: c.cloneSpeaker(s)})
	}
	master := c.cloneMaster(c.state.Master)
	c.mu.RUnlock()

	for _, s := range speakers {
		for _, rc := range remotes {
			cp := c.cloneSpeaker(s.state)
			go rc.HandleSpeakerUpdate(s.id, cp)
		}
	}
	for _, rc := range remotes {
		cp := c.cloneMaster(master)
		go rc.HandleMasterUpdate(cp)
	}
}

//Functional Methods

func (c *Controller) setMute(mute bool) {
	c.mu.Lock()
	if c.state.Master.Mute == mute {
		log.Debugf("Set Mute: %t, but no change", mute)
		return
	}

	c.state.Master.Mute = mute
	c.mu.Unlock()

	c.audioDevice.HandleMute(mute)
	c.fireMute()
}

func (c *Controller) setDim(dim bool) {
	c.mu.Lock()
	if c.state.Master.Dim == dim {
		log.Debugf("Set Dim: %t, but no change", dim)
		return
	}
	c.state.Master.Dim = dim
	c.mu.Unlock()

	c.audioDevice.HandleDim(dim)
	c.fireDim()
}

func (c *Controller) setSpeakerSelected(id SpeakerID, sel bool) {
	c.mu.Lock()

	speaker, ok := c.state.Speaker[id]
	if !ok {
		log.Warnf("No speaker to select: %d", id)
		c.mu.Unlock()
		return
	}

	if speaker.Selected == sel {
		log.Debugf("Set Speaker %d Enable: %t, but no change needed", id, sel)
		c.mu.Unlock()
		return
	}

	if speaker.Disabled {
		log.Debugf("Speaker is disabled. no action required")
		c.mu.Unlock()
		return
	}

	speaker.Selected = sel
	c.mu.Unlock()

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
	c.mu.Lock()

	speaker, ok := c.state.Speaker[id]
	if !ok {
		log.Warnf("No speaker to select: %d", id)
		c.mu.Unlock()
		return
	}

	if speaker.Name == name {
		log.Debugf("Set Speaker %d name: %s, but no change needed", id, name)
		c.mu.Unlock()
		return
	}

	if speaker.Disabled {
		log.Debugf("Speaker is disabled. no action required")
		c.mu.Unlock()
		return
	}
	speaker.Name = name
	c.mu.Unlock()

	c.audioDevice.HandleSpeakerUpdate(id, speaker)
	c.fireSpeakerUpdate(id)
}

func (c *Controller) setMasterVolumeDB(vol int) {
	c.mu.Lock()
	if c.state.Master.VolumeDB == vol {
		c.mu.Unlock()
		return
	}
	c.state.Master.VolumeDB = vol
	c.mu.Unlock()

	c.audioDevice.HandleVolume(vol)
	c.fireVolume()
}

func (c *Controller) setMasterLevel(left, right int) {
	c.mu.Lock()
	c.state.Master.LevelLeft = left
	c.state.Master.LevelRight = right
	c.mu.Unlock()

	c.fireLevel()
}

func (c *Controller) Close() error {
	if c == nil {
		return nil
	}

	for _, r := range c.remoteController {
		if closer, ok := r.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}

	if c.audioDevice != nil {
		return c.audioDevice.Close()
	}
	return nil
}
