package monitorcontroller

// TODO: remove Fader fpr Speaker Level

import (
	"math"
	"reflect"
	"slices"
	"strconv"

	"github.com/ECUST-XX/xml"
	faderdb "github.com/sebastianrau/focusrite-mackie-control/pkg/faderDB"
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/focusriteclient"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sebastianrau/gomcu"
	"github.com/sirupsen/logrus"
)

// TODO Add Meter Functions
// TODO Add Name Functions & Set Display Text

var log *logrus.Entry = logger.WithPackage("monitor-controller")

type Controller struct {
	config *Configuration

	toMcu   chan interface{}
	fromMcu chan interface{}

	toFocusrite   chan focusritexml.Set
	fromFocusrite chan interface{}

	FromController chan interface{}
}

// NewMcuState creates a new McuState
func NewController(
	toMcu chan interface{},
	fromMcu chan interface{},
	toFocusrite chan focusritexml.Set,
	fromFocusrite chan interface{}, config *Configuration) *Controller {

	c := Controller{
		config: config,

		toMcu:   toMcu,
		fromMcu: fromMcu,

		toFocusrite:   toFocusrite,
		fromFocusrite: fromFocusrite,

		FromController: make(chan interface{}, 100),
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

		for _, spk := range c.config.Speaker {
			if spk.Name.IsFcID(FocusriteId(s.ID)) {
				log.Debugf("Found Speaker Name from fc %d: %s", s.ID, s.Value)
				spk.Name.ParseItem(s)
				// TODO Update MCU Displays
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

		xml, _ := xml.MarshalIndent(device.Outputs.Analogues, "", "    ")
		log.Debugln("\n" + string(xml))
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

	c.UpdateAllLeds(c.config.Master.MuteSwitch.McuButtonsList, mute)

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	fcUpdateSet.AddItemBool(int(c.config.Master.MuteSwitch.FcId), mute)
	c.toFocusrite <- *fcUpdateSet

	c.updateSpeakerMute()
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

	fcUpdateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	c.AddItemsToSet(fcUpdateSet, &c.config.Master.DimSwitch)
	c.toFocusrite <- *fcUpdateSet

	c.updateSpeakerVolume()
	c.FromController <- MuteMessage(c.config.Master.MuteSwitch.Value)

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

	c.FromController <- SpeakerEnabledMessage{SpeakerID: id, SpeakerEnabled: enabled}
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
	c.toFocusrite <- *fcUpdateSet
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
	c.toFocusrite <- *fcUpdateSet

}

// gomcu LED shorts
func (c *Controller) setLedBool(sw gomcu.Switch, state bool) {
	c.setLed(sw, mcu.Bool2State(state))
}
func (c *Controller) setLed(sw gomcu.Switch, state gomcu.State) {
	c.toMcu <- mcu.LedCommand{Led: sw, State: state}
}

func (c *Controller) UpdateAllLeds(switches []gomcu.Switch, state bool) {
	for _, led := range switches {
		c.setLedBool(led, state)
	}
}

func (c *Controller) UpdateAllFader(channel []gomcu.Channel, value uint16) {
	for _, fader := range channel {
		c.toMcu <- mcu.FaderSelectCommand{Channel: gomcu.Channel(fader), ChnnalValue: value}
		c.toMcu <- mcu.FaderCommand{Fader: gomcu.Channel(fader), Value: value}
	}
}

// make Focusrite update Set from Mapping Items
func (c *Controller) AddItemsToSet(set *focusritexml.Set, item Mapping) {
	if item.GetFcID() == 0 {
		return
	}
	set.AddItem(focusritexml.Item{ID: int(item.GetFcID()), Value: item.ValueString()})
}

// TODO add more updates here
func (c *Controller) initFocusriteDevice() {
	updateSet := focusritexml.NewSet(c.config.FocusriteDeviceId)
	c.AddItemsToSet(updateSet, &c.config.Master.MuteSwitch)
	c.AddItemsToSet(updateSet, &c.config.Master.DimSwitch)

	c.toFocusrite <- *updateSet

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

	c.UpdateAllFader(c.config.Master.VolumeMcuChannel, c.config.Master.VolumeMcuRaw)

	//Speaker Updates
	// send Speaker States
	for _, speaker := range c.config.Speaker {
		c.UpdateAllLeds(speaker.Mute.McuButtonsList, !speaker.Mute.Value)
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
