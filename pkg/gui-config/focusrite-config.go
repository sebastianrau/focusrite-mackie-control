package guiconfig

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	focusriteclient "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-client"
	fcaudioconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-connector"
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

type FocusriteConfigGui struct {
	newConfig *fcaudioconnector.FcConfiguration

	fClient *focusriteclient.FocusriteClient

	fcDeviceList           map[string]*focusritexml.Device
	fcSelectedDeviceString string
	fcSelectedDevice       *focusritexml.Device

	SpeakerASel   *widget.Select
	SpeakerBSel   *widget.Select
	SpeakerCSel   *widget.Select
	SpeakerDSel   *widget.Select
	SpeakerSubSel *widget.Select

	DeviceSelect       *widget.Select
	DeviceSnLabel      *widget.Label
	DeviceMuteCheckbox *widget.Check
	DeviceDimCheckbox  *widget.Check

	Container *widget.AccordionItem
}

func NewFocusriteConfigGui(cfg *fcaudioconnector.FcConfiguration) *FocusriteConfigGui {

	fc := &FocusriteConfigGui{
		newConfig:        cfg,
		fcSelectedDevice: nil,
		fcDeviceList:     map[string]*focusritexml.Device{},
	}

	fc.fClient = focusriteclient.NewFocusriteClient(focusriteclient.UpdateRaw)

	fc.DeviceSelect = widget.NewSelect([]string{}, func(selected string) {
		if selected != "" {
			dev, ok := fc.fcDeviceList[selected]
			if !ok {
				log.Errorf("Selected Device: %s not found!", selected)
				return
			}

			fc.newConfig.FocusriteSerialNumber = dev.SerialNumber
			fc.newConfig.Master.MuteSwitch = fcaudioconnector.FocusriteId(dev.Monitoring.HardwareControls.Controls.Mute.ID)
			fc.newConfig.Master.DimSwitch = fcaudioconnector.FocusriteId(dev.Monitoring.HardwareControls.Controls.Dim.ID)

			fc.fcSelectedDevice = dev
			fc.fcSelectedDeviceString = fmt.Sprintf("%s (%s)", dev.Model, dev.SerialNumber)
		}
		fc.updateDeviceDetails()
		fc.updateAllSpeakerSelect()
	})

	fc.DeviceSnLabel = widget.NewLabel("")
	fc.DeviceMuteCheckbox = widget.NewCheck("", func(b bool) {})
	fc.DeviceDimCheckbox = widget.NewCheck("", func(b bool) {})

	fc.SpeakerASel = widget.NewSelect([]string{}, func(s string) {
		fc.updateSpeakerConfig(s, monitorcontroller.SpeakerA)
		log.Debugf("Selected: %s", s)
	})

	fc.SpeakerBSel = widget.NewSelect([]string{}, func(s string) {
		fc.updateSpeakerConfig(s, monitorcontroller.SpeakerB)
		log.Debugf("Selected: %s", s)
	})

	fc.SpeakerCSel = widget.NewSelect([]string{}, func(s string) {
		fc.updateSpeakerConfig(s, monitorcontroller.SpeakerC)
		log.Debugf("Selected: %s", s)
	})

	fc.SpeakerDSel = widget.NewSelect([]string{}, func(s string) {
		fc.updateSpeakerConfig(s, monitorcontroller.SpeakerD)
		log.Debugf("Selected: %s", s)
	})

	fc.SpeakerSubSel = widget.NewSelect([]string{}, func(s string) {
		fc.updateSpeakerConfig(s, monitorcontroller.Sub)
		log.Debugf("Selected: %s", s)
	})

	fc.Container = widget.NewAccordionItem("Focusrite:",
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Device:"), fc.DeviceSelect,
			widget.NewLabel("Serial Number"), fc.DeviceSnLabel,
			widget.NewLabelWithStyle("Master:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), layout.NewSpacer(),
			widget.NewLabel("Master Mute"), fc.DeviceMuteCheckbox,
			widget.NewLabel("Master Dim"), fc.DeviceDimCheckbox,
			widget.NewLabelWithStyle("Speaker:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), layout.NewSpacer(),
			widget.NewLabel("Speaker A:"), fc.SpeakerASel,
			widget.NewLabel("Speaker B:"), fc.SpeakerBSel,
			widget.NewLabel("Speaker C:"), fc.SpeakerCSel,
			widget.NewLabel("Speaker D:"), fc.SpeakerDSel,
			widget.NewLabel("Subwoofer:"), fc.SpeakerSubSel,
		),
	)

	widget.NewLabelWithStyle("test", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	go fc.run()

	return fc
}

func (fc *FocusriteConfigGui) run() {
	for msg := range fc.fClient.FromFocusrite {
		switch m := msg.(type) {
		case focusriteclient.ApprovalMessasge:
		case focusriteclient.ConnectionStatusMessage:

		case focusriteclient.DeviceArrivalMessage:
			log.Debugf("Device Arrived len:%d", len(fc.fcDeviceList))
			fc.updateDeviceSelect(fc.fClient.DeviceList)

		case focusriteclient.DeviceRemovalMessage:
			log.Debugf("Device Removel: %d", m)
			fc.updateDeviceSelect(fc.fClient.DeviceList)

		case focusriteclient.DeviceUpdateMessage:
			log.Debugf("Device Update: %s", m.SerialNumber)
			fc.updateDeviceSelect(fc.fClient.DeviceList)

		case focusriteclient.RawUpdateMessage:
		}
	}
}

func (fc *FocusriteConfigGui) updateDeviceSelect(list focusriteclient.DeviceList) {

	if len(fc.fcDeviceList) == len(list){
		log.Debugf("no new devices in list: %d", len(list))
		return
	}
	options := make([]string, 0)
	fc.fcDeviceList = make(map[string]*focusritexml.Device)
	fc.fcSelectedDevice = nil
	fc.fcSelectedDeviceString = ""
	for _, device := range list {
		text := fmt.Sprintf("%s (%s)", device.Model, device.SerialNumber)
		options = append(options, text)
		fc.fcDeviceList[text] = device
		if fc.newConfig.FocusriteSerialNumber == device.SerialNumber {
			fc.fcSelectedDeviceString = text
			fc.fcSelectedDevice = device
			log.Debugf("Selected Device: %s", text)
		}
	}
	fc.DeviceSelect.SetOptions(options)
	fc.DeviceSelect.Refresh()

	fc.DeviceSelect.SetSelected(fc.fcSelectedDeviceString)

	if fc.fcSelectedDeviceString != "" {
		fc.updateDeviceDetails()
		fc.updateAllSpeakerSelect()
	}

}

func (c *FocusriteConfigGui) updateDeviceDetails() {

	if c.fcSelectedDevice == nil {
		c.DeviceSnLabel.SetText("-")
		c.DeviceDimCheckbox.Enable()
		c.DeviceDimCheckbox.Checked = false
		c.DeviceDimCheckbox.Disable()

		c.DeviceMuteCheckbox.Enable()
		c.DeviceMuteCheckbox.Checked = false
		c.DeviceMuteCheckbox.Disable()
		return
	}

	// Set Config
	c.DeviceSnLabel.SetText(c.newConfig.FocusriteSerialNumber)
	c.DeviceDimCheckbox.Enable()
	c.DeviceDimCheckbox.Checked = c.newConfig.Master.DimSwitch != 0
	c.DeviceDimCheckbox.Disable()

	c.DeviceMuteCheckbox.Enable()
	c.DeviceMuteCheckbox.Checked = c.newConfig.Master.MuteSwitch != 0
	c.DeviceMuteCheckbox.Disable()
}

func (fc *FocusriteConfigGui) updateAllSpeakerSelect() {
	fc.updateSpeakerSelect(fc.SpeakerASel, monitorcontroller.SpeakerA)
	fc.updateSpeakerSelect(fc.SpeakerBSel, monitorcontroller.SpeakerB)
	fc.updateSpeakerSelect(fc.SpeakerCSel, monitorcontroller.SpeakerC)
	fc.updateSpeakerSelect(fc.SpeakerDSel, monitorcontroller.SpeakerD)
	fc.updateSpeakerSelect(fc.SpeakerSubSel, monitorcontroller.Sub)
}

func (c *FocusriteConfigGui) updateSpeakerConfig(selected string, spkID monitorcontroller.SpeakerID) {

	if c.fcSelectedDevice == nil {
		return
	}

	if selected == "none" {
		c.newConfig.Speaker[spkID].Name = 0
		c.newConfig.Speaker[spkID].MeterL = 0
		c.newConfig.Speaker[spkID].MeterR = 0
		c.newConfig.Speaker[spkID].Mute = 0
		c.newConfig.Speaker[spkID].OutputGain = 0

	}

	dev := c.fcSelectedDevice
	for i, anOut := range dev.Outputs.Analogues {
		if anOut.StereoName == selected /*&& anOut.Available.Value == true*/ {
			c.newConfig.Speaker[spkID].Name = fcaudioconnector.FocusriteId(anOut.Nickname.ID)
			c.newConfig.Speaker[spkID].MeterL = fcaudioconnector.FocusriteId(anOut.Meter.ID)
			c.newConfig.Speaker[spkID].Mute = fcaudioconnector.FocusriteId(anOut.Mute.ID)
			c.newConfig.Speaker[spkID].OutputGain = fcaudioconnector.FocusriteId(anOut.Gain.ID)
			c.newConfig.Speaker[spkID].MeterR = fcaudioconnector.FocusriteId(dev.Outputs.Analogues[i+1].Meter.ID)
		}
	}

	log.Debugf("New Config: %v", c.newConfig.Speaker[spkID])
}

func (c *FocusriteConfigGui) updateSpeakerSelect(sel *widget.Select, spkId monitorcontroller.SpeakerID) {
	if c.fcSelectedDevice == nil {
		return
	}
	log.Debugf("Updating Speaker Selector %s - %s", monitorcontroller.SpeakerName[spkId], monitorcontroller.SpeakerName[spkId])

	outputList := make([]string, 0)

	outputList = append(outputList, "none")
	selectedOutput := ""

	for _, outs := range c.fcSelectedDevice.Outputs.Analogues {
		if outs.StereoName != "" {
			outputList = append(outputList, outs.StereoName)

			if outs.Mute.ID == int(c.newConfig.Speaker[spkId].Mute) {
				selectedOutput = outs.StereoName
			}
		}
	}
	sel.SetOptions(outputList)
	sel.Refresh()
	sel.SetSelected(selectedOutput)
}
