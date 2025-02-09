package monitorcontroller

// TODO: remove Fader fpr Speaker Level

import (
	"math"
	"reflect"
	"slices"
	"strconv"

	faderdb "github.com/sebastianrau/focusrite-mackie-control/pkg/faderDB"
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/focusriteclient"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sebastianrau/gomcu"
)

// TODO Add Meter Functions
// TODO Add Name Functions & Set Display Text

var log *logger.CustomLogger = logger.WithPackage("monitor-controller")

type Controller struct {
	config *Configuration
	fc     *focusriteclient.FocusriteClient

	toMcu   chan interface{}
	fromMcu chan interface{}

	fromRemoteController chan interface{}

	toController     chan interface{} //Command interface for Remote Controller
	remoteController []RemoteController
}

// NewMcuState creates a new McuState
func NewController(

	toMcu chan interface{},
	fromMcu chan interface{},
	config *Configuration) *Controller {

	c := Controller{
		config:  config,
		toMcu:   toMcu,
		fromMcu: fromMcu,

		toController: make(chan interface{}, 100),
	}

	c.fc = focusriteclient.NewFocusriteClient(focusriteclient.UpdateRaw)

	c.config.DefaultValues()

	go c.run()
	return &c
}

func (c *Controller) run() {
	for {
		select {
		case mcu := <-c.fromMcu:
			c.handleMcu(mcu)

		case focusrite := <-c.fc.FromFocusrite:
			c.handleFocusrite(focusrite)

		case remote := <-c.fromRemoteController:
			c.handleRemoteControl(remote)
		}

	}
}

func (c *Controller) handleMcu(msg interface{}) {
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

		// FIXME: remove from controller
		/*
			switch f.KeyNumber {
			case gomcu.Play,
				gomcu.Stop,
				gomcu.FastFwd,
				gomcu.Rewind:
				c.FromController <- TransportMessage(f.KeyNumber)
			}
			log.Infof("Unknown Button: 0x%X %s", f.KeyNumber, f.HotkeyName)
		*/
	case mcu.RawFaderMessage:
		if slices.Contains(c.config.Master.VolumeMcuChannel, f.FaderNumber) {
			c.setMasterVolumeRawValue(f.FaderValue)
		}

	default:
		log.Warnf("Unhandled mcu message %s: %v\n", reflect.TypeOf(msg), msg)

	}
}

func (c *Controller) handleFocusrite(msg interface{}) {
	switch m := msg.(type) {
	case focusriteclient.DeviceArrivalMessage:
		c.handleFocusriteDeviceArrival(focusritexml.Device(m))

	case focusriteclient.DeviceRemovalMessage:
		c.handleFocusriteDeviceRemoval(int(m))

	case focusriteclient.RawUpdateMessage:
		c.handleFocusriteUpdate(focusritexml.Set(m))

	case focusriteclient.ApprovalMessasge:
		//inoring approval message from fc
	case focusriteclient.ConnectionStatusMessage:
		//ignoring connection status Messages from fc
	case focusriteclient.DeviceUpdateMessage:
		//ignoring high level updates from fc
	}

}

func (c *Controller) handleFocusriteUpdate(set focusritexml.Set) {
	if set.DevID != c.config.FocusriteDeviceId {
		return
	}

	// log.Debugf("New Raw Update Device Arrived. Items: %d", len(set.Items))
	for _, s := range set.Items {
		fcID := FocusriteId(s.ID)
		if c.config.Master.DimSwitch.IsFcID(fcID) {
			log.Debugf("Found Dim ID %d", s.ID)
			boolValue, err := strconv.ParseBool(s.Value)
			if err != nil {
				return
			}
			c.setDim(boolValue)
			return
		}

		if c.config.Master.MuteSwitch.IsFcID(fcID) {
			log.Debugf("Found Mute ID %d: %v", s.ID, s.Value)
			boolValue, err := strconv.ParseBool(s.Value)
			if err != nil {
				log.Error(err.Error())
				return
			}
			c.setMute(boolValue)
			return
		}

		for id, spk := range c.config.Speaker {
			if spk.Name.IsFcID(FocusriteId(s.ID)) {
				log.Debugf("Found Speaker Name from fc %d: %s", s.ID, s.Value)
				spk.Name.ParseItem(s)
				c.fireSpeakerUpdate(id, spk)
			}

			if spk.Mute.IsFcID(FocusriteId(s.ID)) {
				log.Debugf("Found Speaker mute from fc %d: %v", s.ID, s.Value)
				/*
					boolValue, err := strconv.ParseBool(s.Value)
					if err != nil {
						log.Error(err.Error())
						return
					}
					c.setSpeakerEnabled(k, boolValue) //not inverted because value is mute value
				*/
			}

			if spk.OutputGain.IsFcID(FocusriteId(s.ID)) {
				// TOOD relfect gain
			}

			if spk.Meter.IsFcID(FocusriteId(s.ID)) {
				spk.Meter.ParseItem(s)
				// TODO Update Meter Values
			}
		}
	}
}

func (c *Controller) handleFocusriteDeviceArrival(device focusritexml.Device) {
	log.Debugf("New Focusrite Device Arrived ID:%d SN:%s", device.ID, device.SerialNumber)
	if device.SerialNumber == c.config.FocusriteSerialNumber {
		c.config.FocusriteDeviceId = device.ID
		c.initFocusriteDevice()
		log.Debugf("configured device with SN: %s arrived with ID ID:%d", device.SerialNumber, device.ID)
	}
}

func (c *Controller) handleFocusriteDeviceRemoval(deviceId int) {
	log.Debugf("Focusrite Device removed ID:%d", deviceId)
	if deviceId != 0 && deviceId == c.config.FocusriteDeviceId {
		c.config.FocusriteDeviceId = 0
	}
}

//Remote Controls

func (c *Controller) RegisterRemoteController(r RemoteController) *Controller {
	c.remoteController = append(c.remoteController, r)
	c.fireAllUpdate()
	return c
}

func (c *Controller) handleRemoteControl(remote interface{}) {
	switch r := remote.(type) {
	case RcUpdateRequest:

	case RcSetMute:
		c.setMute(bool(r))
	case RcSetDim:
		c.setDim(bool(r))
	case RcSetVolume:
		c.setMasterVolumeDB(int(r))
	case RcSpeakerSelect:
		c.setSpeakerEnabled(r.Id, r.State)
	}
}

func (c *Controller) fireDeviceArrival(dev *focusritexml.Device) {
	for _, rc := range c.remoteController {
		rc.HandleDeviceArrival(dev)
	}
}
func (c *Controller) fireDeviceRemoval() {
	for _, rc := range c.remoteController {
		rc.HandleDeviceRemoval()
	}
}

func (c *Controller) fireDim(state bool) {
	for _, rc := range c.remoteController {
		rc.HandleDim(state)
	}
}
func (c *Controller) fireMute(state bool) {
	for _, rc := range c.remoteController {
		rc.HandleMute(state)
	}
}
func (c *Controller) fireVolume(vol int) {
	for _, rc := range c.remoteController {
		rc.HandleVolume(vol)
	}
}
func (c *Controller) fireMeter(level int) {
	for _, rc := range c.remoteController {
		rc.HandleMeter(level)
	}
}
func (c *Controller) fireSpeakerSelect(id SpeakerID, state bool) {
	for _, rc := range c.remoteController {
		rc.HandleSpeakerSelect(id, state)
	}
}
func (c *Controller) fireSpeakerUpdate(id SpeakerID, spk *SpeakerConfig) {
	for _, rc := range c.remoteController {
		rc.HandleSpeakerUpdate(id, spk)
	}

}
func (c *Controller) fireMasterUpdate(master *MasterConfig) {
	for _, rc := range c.remoteController {
		rc.HandleMasterUpdate(master)
	}
}

func (c *Controller) fireAllUpdate() {
	for spkId, spk := range c.config.Speaker {
		for _, rc := range c.remoteController {
			rc.HandleSpeakerUpdate(spkId, spk)
		}
	}
	for _, rc := range c.remoteController {
		rc.HandleMasterUpdate(c.config.Master)
	}

}

// Mute function
func (c *Controller) setMute(mute bool) {
	if c.config.Master.MuteSwitch.Value == mute {
		log.Debugf("Set Mute: %t, but no change", mute)
		return
	}

	c.config.Master.MuteSwitch.Value = mute

	c.updateAllLeds(c.config.Master.MuteSwitch.McuButtonsList, mute)

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(c.config.Master.MuteSwitch.FcId), mute)
	c.fc.ToFocusrite <- *fcUpdateSet

	c.updateSpeakerMute()
	c.fireMute(c.config.Master.MuteSwitch.Value)
}

// Dim function
func (c *Controller) setDim(dim bool) {
	if c.config.Master.DimSwitch.Value == dim {
		log.Debugf("Set Dim: %t, but no change", dim)
		return
	}

	c.config.Master.DimSwitch.Value = dim

	for _, sw := range c.config.Master.DimSwitch.McuButtonsList {
		log.Debugf("Setting Led: %d to %t", sw, dim)
		c.setLedBool(sw, dim)
	}

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	c.addItemsToSet(fcUpdateSet, &c.config.Master.DimSwitch)
	c.fc.ToFocusrite <- *fcUpdateSet

	c.updateSpeakerVolume()
	c.fireDim(dim)

}

// Speaker Enable functions
func (c *Controller) setSpeakerEnabled(id SpeakerID, enabled bool) {
	mute := !enabled
	speaker, ok := c.config.Speaker[id]
	if !ok || mute == speaker.Mute.Value {
		log.Debugf("Set Speaker %d Enable: %t, but no change needed", id, enabled)
		return
	}

	speaker.Mute.Value = mute
	speakerExclusive := speaker.Exclusive
	speakerType := speaker.Type

	if enabled {
		// speaker will be turned on, if is exclusice, disable all others with same type
		if speakerExclusive {

			for k, v := range c.config.Speaker {
				if speakerType == v.Type && k != id && !v.Mute.Value {
					c.setSpeakerEnabled(k, false)
				}
			}
		} else {
			//Check other speakers for eclusive flag
			for k, v := range c.config.Speaker {
				if k == id { // no action for own speaker
					continue
				}
				//if same speaker type other speaker is exclusive and enabled --> mute it
				if speakerType == v.Type && v.Exclusive && !v.Mute.Value && k != id {
					c.setSpeakerEnabled(k, false)
				}
			}
		}
	}

	c.updateSpeakerMute()
	for _, v := range speaker.Mute.McuButtonsList {
		c.setLedBool(v, enabled)
	}

	c.fireSpeakerSelect(id, enabled)
}

func (c *Controller) toggleSpeakerEnabled(id SpeakerID) {
	speaker, ok := c.config.Speaker[id]
	if !ok {
		return
	}
	c.setSpeakerEnabled(id, speaker.Mute.Value) //use mute value to invert
}

// Volume function
func (c *Controller) setMasterVolumeRawValue(vol uint16) {
	if math.Abs(float64(c.config.Master.VolumeMcuRaw-vol)) < 50 { //skip small changes for performance reasons
		return
	}

	db := faderdb.FaderToDB(vol)
	c.config.Master.VolumeMcuRaw = vol
	c.config.Master.VolumeDB = int(db)

	c.fireVolume(c.config.Master.VolumeDB)
	c.updateSpeakerVolume()
}

func (c *Controller) setMasterVolumeDB(vol int) {
	if c.config.Master.VolumeDB == vol {
		return
	}
	c.config.Master.VolumeDB = vol
	c.config.Master.VolumeMcuRaw = faderdb.DBToFader(float64(vol))

	c.fireVolume(c.config.Master.VolumeDB)
	c.updateSpeakerVolume()
}

func (c *Controller) updateSpeakerVolume() {
	volume := int(c.config.Master.VolumeDB)
	if volume >= 0 {
		volume = 0
	}
	if c.config.Master.DimSwitch.Value {
		volume = volume - int(c.config.Master.DimVolumeOffset)
	}
	if volume < -127 {
		volume = -127
	}

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	for _, spk := range c.config.Speaker {
		fcUpdateSet.AddItemInt(int(spk.OutputGain.FcId), volume)
	}
	c.fc.ToFocusrite <- *fcUpdateSet
}

func (c *Controller) updateSpeakerMute() {

	mute := c.config.Master.MuteSwitch.Value

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(c.config.Master.MuteSwitch.FcId), mute)

	for k, spk := range c.config.Speaker {
		state := mute || spk.Mute.Value
		fcUpdateSet.AddItemBool(int(spk.Mute.FcId), state)
		log.Debugf("setting focusrite speaker %d Level to %t", k, state)
	}
	c.fc.ToFocusrite <- *fcUpdateSet

}

// gomcu LED shorts
func (c *Controller) setLedBool(sw gomcu.Switch, state bool) {
	c.setLed(sw, mcu.Bool2State(state))
}

func (c *Controller) setLed(sw gomcu.Switch, state gomcu.State) {
	c.toMcu <- mcu.LedCommand{Led: sw, State: state}
}

func (c *Controller) updateAllLeds(switches []gomcu.Switch, state bool) {
	for _, led := range switches {
		c.setLedBool(led, state)
	}
}

func (c *Controller) updateAllFader(channel []gomcu.Channel, value uint16) {
	for _, fader := range channel {
		c.toMcu <- mcu.FaderSelectCommand{Channel: gomcu.Channel(fader), ChnnalValue: value}
		c.toMcu <- mcu.FaderCommand{Fader: gomcu.Channel(fader), Value: value}
	}
}

// make Focusrite update Set from Mapping Items
func (c *Controller) addItemsToSet(set *focusritexml.Set, item Mapping) {
	if item.GetFcID() == 0 {
		return
	}
	set.AddItem(focusritexml.Item{ID: int(item.GetFcID()), Value: item.ValueString()})
}

// TODO add more updates here
func (c *Controller) initFocusriteDevice() {
	updateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	c.addItemsToSet(updateSet, &c.config.Master.MuteSwitch)
	c.addItemsToSet(updateSet, &c.config.Master.DimSwitch)

	c.fc.ToFocusrite <- *updateSet

	c.updateSpeakerVolume()
}

// TODO  update MCU Values for init
func (c *Controller) initMcu() {

	//Master Updaste
	// send Mute States
	for _, mId := range c.config.Master.MuteSwitch.McuButtonsList {
		c.setLedBool(mId, c.config.Master.MuteSwitch.Value)
	}

	// send Dim Switches
	for _, mId := range c.config.Master.DimSwitch.McuButtonsList {
		c.setLedBool(mId, c.config.Master.DimSwitch.Value)
	}

	// send Fader Values

	c.updateAllFader(c.config.Master.VolumeMcuChannel, c.config.Master.VolumeMcuRaw)

	//Speaker Updates
	// send Speaker States
	for _, speaker := range c.config.Speaker {
		c.updateAllLeds(speaker.Mute.McuButtonsList, !speaker.Mute.Value)
	}

}

/*
func (c *Controller) initDisplay() {
	for k, v := range c.mapping.Speaker {
		c.setChannelText(k, v.Name, false)
	}
	c.setChannelText(MasterFader, c.mapping.Master.Name, false)
}

func (d *Controller) UpdateMap() {
	if d.focusriteElementMap == nil {
		d.focusriteElementMap = make(map[int]focusritexml.Elements)
	}
	d.focusriteElementMap[d.mapping.Master.Focusrite.Mute.ID] = &d.mapping.Master.Focusrite.Mute
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


*/
