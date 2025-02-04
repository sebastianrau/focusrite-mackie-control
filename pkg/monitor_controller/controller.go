package monitorcontroller

// TODO: remove Fader fpr Speaker Level

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"

	faderdb "github.com/sebastianrau/focusrite-mackie-control/pkg/faderDB"
	focusriteclient "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-client"
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sirupsen/logrus"
)

var log *logrus.Entry = logger.WithPackage("monitor-controller")

type Controller struct {
	config Configuration

	toMcu   chan interface{}
	fromMcu chan interface{}

	toFocusrite   chan focusritexml.Set
	fromFocusrite chan interface{}

	FromController chan interface{}

	mcuLedState map[gomcu.Switch]gomcu.State
}

// NewMcuState creates a new McuState
func NewController(
	toMcu chan interface{},
	fromMcu chan interface{},
	toFocusrite chan focusritexml.Set,
	fromFocusrite chan interface{}) *Controller {

	c := Controller{
		config: DEFAULT,

		toMcu:   toMcu,
		fromMcu: fromMcu,

		toFocusrite:    toFocusrite,
		fromFocusrite:  fromFocusrite,
		FromController: make(chan interface{}, 100),

		mcuLedState: make(map[gomcu.Switch]gomcu.State),
	}
	c.config.DefaultValues()

	go c.Run()
	return &c
}

func (c *Controller) Run() {
	for {
		select {
		case mcu := <-c.fromMcu:
			c.handleMcu(mcu)

		case focusrite := <-c.fromFocusrite:
			c.handleFocusrite(focusrite)
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
			log.Debugf("Fader %d: %d ", f.FaderNumber, f.FaderValue)
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

	log.Debugf("New Raw Update Device Arrived. Items: %d", len(set.Items))

	for _, s := range set.Items {
		if slices.Contains(c.config.Master.DimSwitch.FcIdsList, FocusriteId(s.ID)) {
			log.Debugf("Found Dim ID %d", s.ID)
			boolValue, err := strconv.ParseBool(s.Value)
			if err != nil {
				return
			}
			c.setDim(boolValue)
			return
		}

		if slices.Contains(c.config.Master.MuteSwitch.FcIdsList, FocusriteId(s.ID)) {
			log.Debugf("Found Mute ID %d: %v", s.ID, s.Value)
			boolValue, err := strconv.ParseBool(s.Value)
			if err != nil {
				log.Error(err.Error())
				return
			}

			c.setMute(boolValue)
			return
		}

		for k, spk := range c.config.Speaker {
			if spk.Mute.IsFcID(FocusriteId(s.ID)) {
				log.Debugf("Found Speaker mute from fc %d: %v", s.ID, s.Value)
				boolValue, err := strconv.ParseBool(s.Value)
				if err != nil {
					log.Error(err.Error())
					return
				}
				c.setSpeakerEnabled(k, boolValue) //not inverted because value is mute value
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

// Mute function
func (c *Controller) setMute(mute bool) {
	if c.config.Master.MuteSwitch.Value == mute {
		log.Debugf("Set Mute: %t, but no change", mute)
		return
	}

	c.config.Master.MuteSwitch.Value = mute

	for _, sw := range c.config.Master.MuteSwitch.McuButtonsList {
		log.Debugf("Setting Led: %d to %t", sw, mute)
		c.setLedBool(sw, mute)
	}

	fcUpdateSet, err := focusritexml.NewSet(c.config.FocusriteDeviceId)
	if err != nil {
		return
	}

	for _, fc := range c.config.Master.MuteSwitch.FcIdsList {
		fcUpdateSet.AddItemBool(int(fc), mute)
	}

	for _, spk := range c.config.Speaker {
		for _, spkMuteId := range spk.Mute.FcIdsList {
			state := mute || spk.Mute.Value
			fcUpdateSet.AddItemBool(int(spkMuteId), state)
		}
	}
	c.toFocusrite <- *fcUpdateSet
	c.FromController <- MuteMessage(c.config.Master.MuteSwitch.Value)

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

	fcUpdateSet, err := focusritexml.NewSet(c.config.FocusriteDeviceId)
	if err != nil {
		return
	}

	c.AddItemsToSet(fcUpdateSet, &c.config.Master.DimSwitch)
	c.updateSpeakerVolume()
	c.toFocusrite <- *fcUpdateSet
	c.FromController <- MuteMessage(c.config.Master.MuteSwitch.Value)

}

// Speaker Enable functions
func (c *Controller) setSpeakerEnabled(id int, enabled bool) {
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

	fcUpdateItems := []focusritexml.Item{}
	for _, fcID := range speaker.Mute.FcIdsList {
		state := mute || c.config.Master.MuteSwitch.Value //mute speaker or master muter
		fcUpdateItems = append(fcUpdateItems, focusritexml.Item{ID: int(fcID), Value: fmt.Sprintf("%t", state)})
	}
	c.toFocusrite <- focusritexml.Set{Items: fcUpdateItems}
	log.Debugf("Send: %v", fcUpdateItems)

	for _, v := range speaker.Mute.McuButtonsList {
		c.setLedBool(v, enabled)
	}

	c.FromController <- SpeakerEnabledMessage{SpeakerID: id, SpeakerEnabled: enabled}
}
func (c *Controller) toggleSpeakerEnabled(id int) {
	speaker, ok := c.config.Speaker[id]
	if !ok {
		return
	}
	c.setSpeakerEnabled(id, speaker.Mute.Value) //use mute value to invert
}

// Volume function
func (c *Controller) setMasterVolumeRawValue(vol uint16) {
	if c.config.Master.VolumeMcuRaw == vol {
		return
	}

	db := faderdb.FaderToDB(vol)
	c.config.Master.VolumeMcuRaw = vol
	c.config.Master.VolumeDB = int(db)
	log.Debugf("New Master Volume %d (%d db)", c.config.Master.VolumeMcuRaw, c.config.Master.VolumeDB)

	c.updateSpeakerVolume()
}

func (c *Controller) setMasterVolumeDB(vol int) {
	if c.config.Master.VolumeDB == vol {
		return
	}
	c.config.Master.VolumeDB = vol
	c.config.Master.VolumeMcuRaw = faderdb.DBToFader(float64(vol))

	c.updateSpeakerVolume()
}

func (c *Controller) updateSpeakerVolume() {

	fcUpdateSet, err := focusritexml.NewSet(c.config.FocusriteDeviceId)
	if err != nil {
		return
	}

	volume := int(c.config.Master.VolumeDB)
	if c.config.Master.DimSwitch.Value {
		volume = volume - int(c.config.Master.DimVolumeOffset)
	}

	if volume >= 0 {
		volume = 0
	}

	if volume < -127 {
		volume = 127
	}

	for _, spk := range c.config.Speaker {
		for _, fcId := range spk.OutputGain.FcIdsList {
			fcUpdateSet.AddItemInt(int(fcId), volume)
		}
	}

	c.toFocusrite <- *fcUpdateSet
	log.Debugf("setting focusrite speaker Level to %d", volume)

}

// gomcu LED shorts
func (c *Controller) setLedBool(sw gomcu.Switch, state bool) {
	c.setLed(sw, mcu.Bool2State(state))
}
func (c *Controller) setLed(sw gomcu.Switch, state gomcu.State) {
	s, ok := c.mcuLedState[sw]
	if s != state || !ok { //new state or new entry
		c.mcuLedState[sw] = state
		c.toMcu <- mcu.LedCommand{Led: sw, State: state}
	}
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
	for _, v := range c.config.Master.VolumeMcuChannel {
		c.toMcu <- mcu.FaderSelectCommand{Channel: v, ChnnalValue: DEFAULT.Master.VolumeMcuRaw}
		c.toMcu <- mcu.FaderCommand{Fader: v, Value: c.config.Master.VolumeMcuRaw}
	}

	//Speaker Updates
	// send Speaker States
	for _, speaker := range c.config.Speaker {
		for _, mId := range speaker.Mute.McuButtonsList {
			c.setLedBool(mId, !speaker.Mute.Value)
		}
	}

}

// TODO add more updates here
func (c *Controller) initFocusriteDevice() {
	updateSet, err := focusritexml.NewSet(c.config.FocusriteDeviceId)
	if err != nil {
		return
	}
	c.AddItemsToSet(updateSet, &c.config.Master.MuteSwitch)
	c.AddItemsToSet(updateSet, &c.config.Master.DimSwitch)

	c.toFocusrite <- *updateSet

	c.updateSpeakerVolume()
}

// make Focusrite update Set from Mapping Items
func (c *Controller) AddItemsToSet(set *focusritexml.Set, item Mapping) {
	for _, id := range item.FcIds() {
		set.AddItem(focusritexml.Item{ID: int(id), Value: item.ValueString()})
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



func (c *Controller) SetMasterLevel(level uint16) {
	if c.masterLevel != level {
		c.masterLevel = level
		c.masterLevelDB = faderdb.FaderToDB(level)
		c.toMcu <- mcu.FaderCommand{Fader: c.mapping.Master.Mcu.Fader, Value: level}
		c.FromController <- c.NewSpeakerLevelMessage()
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

func (c *Controller) SetMasterMeter(valueDB float64) {
	out := mcu.Db2MeterLevel(valueDB)
	if c.masterMeter != out {
		c.masterMeter = out
		c.toMcu <- mcu.MeterCommand{Channel: c.mapping.Master.Mcu.Fader, Value: gomcu.MeterLevel(out)}

	}
}

*/
