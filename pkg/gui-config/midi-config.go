package guiconfig

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	mcuconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/mcu-connector"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
	"github.com/sebastianrau/gomcu"
	"gitlab.com/gomidi/midi/v2"
)

type MidiConfigGui struct {
	newConfig *mcuconnector.McuConnectorConfig

	inputSelect  *widget.Select
	outputSelect *widget.Select

	masterMuteSelect  *widget.Select
	masterDimSelect   *widget.Select
	masterFaderSelect *widget.Select

	speakerASelect   *widget.Select
	speakerBSelect   *widget.Select
	speakerCSelect   *widget.Select
	speakerDSelect   *widget.Select
	speakerSubSelect *widget.Select

	Container *widget.AccordionItem
}

func NewMidiConfigGui(cfg *mcuconnector.McuConnectorConfig) *MidiConfigGui {
	mc := &MidiConfigGui{
		newConfig: cfg,
	}

	// Midi Config
	mc.inputSelect = widget.NewSelect(getMidiInputs(), func(selected string) {
		mc.newConfig.MidiInputPort = selected
	})

	mc.outputSelect = widget.NewSelect(getMidiOutputs(), func(selected string) {
		mc.newConfig.MidiOutputPort = selected
	})

	mc.masterDimSelect = widget.NewSelect(getMcuSwtiches(), func(s string) {
		sw, ok := gomcu.IDs[s]
		if !ok {
			log.Error("cant find midi ID")
			return
		}
		mc.newConfig.MasterDimSwitch = sw
	})

	mc.masterMuteSelect = widget.NewSelect(getMcuSwtiches(), func(s string) {
		sw, ok := gomcu.IDs[s]
		if !ok {
			log.Error("cant find midi ID")
			return
		}
		mc.newConfig.MasterMuteSwitch = sw
	})

	mc.masterFaderSelect = widget.NewSelect(getMcuChannels(), func(s string) {
		sw, ok := gomcu.ChannelIDs[s]
		if !ok {
			log.Error("cant find midi ID")
			return
		}
		mc.newConfig.MasterVolumeChannel = sw
	})

	mc.speakerASelect = widget.NewSelect(getMcuSwtiches(), func(s string) {
		sw, ok := gomcu.IDs[s]
		if !ok {
			log.Error("cant find midi ID")
			return
		}
		mc.newConfig.SpeakerSelect[monitorcontroller.SpeakerA] = sw
	})
	mc.speakerBSelect = widget.NewSelect(getMcuSwtiches(), func(s string) {
		sw, ok := gomcu.IDs[s]
		if !ok {
			log.Error("cant find midi ID")
			return
		}
		mc.newConfig.SpeakerSelect[monitorcontroller.SpeakerB] = sw
	})
	mc.speakerCSelect = widget.NewSelect(getMcuSwtiches(), func(s string) {
		sw, ok := gomcu.IDs[s]
		if !ok {
			log.Error("cant find midi ID")
			return
		}
		mc.newConfig.SpeakerSelect[monitorcontroller.SpeakerC] = sw
	})
	mc.speakerDSelect = widget.NewSelect(getMcuSwtiches(), func(s string) {
		sw, ok := gomcu.IDs[s]
		if !ok {
			log.Error("cant find midi ID")
			return
		}
		mc.newConfig.SpeakerSelect[monitorcontroller.SpeakerD] = sw
	})
	mc.speakerSubSelect = widget.NewSelect(getMcuSwtiches(), func(s string) {
		sw, ok := gomcu.IDs[s]
		if !ok {
			log.Error("cant find midi ID")
			return
		}
		mc.newConfig.SpeakerSelect[monitorcontroller.Sub] = sw
	})

	// Speaker Section:
	//((SpeakerSelect map[monitorcontroller.SpeakerID][]gomcu.Switch

	mc.Container = widget.NewAccordionItem("Midi:",
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Input Port:"), mc.inputSelect,
			widget.NewLabel("Output Port:"), mc.outputSelect,
			layout.NewSpacer(), layout.NewSpacer(),
			widget.NewLabel("Fader:"), mc.masterFaderSelect,
			widget.NewLabel("Mute:"), mc.masterMuteSelect,
			widget.NewLabel("Dim:"), mc.masterDimSelect,
			layout.NewSpacer(), layout.NewSpacer(),
			widget.NewLabel("Speaker A:"), mc.speakerASelect,
			widget.NewLabel("Speaker B:"), mc.speakerBSelect,
			widget.NewLabel("Speaker C:"), mc.speakerCSelect,
			widget.NewLabel("Speaker D:"), mc.speakerDSelect,
			widget.NewLabel("Subwoofer:"), mc.speakerSubSelect,
		),
	)

	mc.Update()

	return mc
}

func (mc *MidiConfigGui) Update() {
	mc.UpdateMidiPorts()
	mc.UpdateSwtich(mc.newConfig.MasterDimSwitch, mc.masterDimSelect)
	mc.UpdateSwtich(mc.newConfig.MasterMuteSwitch, mc.masterMuteSelect)
	mc.UpdateChannel(mc.newConfig.MasterVolumeChannel, mc.masterFaderSelect)

	mc.UpdateSwtich(mc.newConfig.SpeakerSelect[monitorcontroller.SpeakerA], mc.speakerASelect)
	mc.UpdateSwtich(mc.newConfig.SpeakerSelect[monitorcontroller.SpeakerB], mc.speakerBSelect)
	mc.UpdateSwtich(mc.newConfig.SpeakerSelect[monitorcontroller.SpeakerC], mc.speakerCSelect)
	mc.UpdateSwtich(mc.newConfig.SpeakerSelect[monitorcontroller.SpeakerD], mc.speakerDSelect)
	mc.UpdateSwtich(mc.newConfig.SpeakerSelect[monitorcontroller.Sub], mc.speakerSubSelect)

}

func (mc *MidiConfigGui) UpdateSwtich(sw gomcu.Switch, sel *widget.Select) {
	name := gomcu.Names[sw]
	sel.SetSelected(name)
}

func (mc *MidiConfigGui) UpdateChannel(sw gomcu.Channel, sel *widget.Select) {
	name := gomcu.ChannelNames[sw]
	sel.SetSelected(name)
}

func (mc *MidiConfigGui) UpdateMidiPorts() {

	inputs := make([]string, 0)
	inputs = append(inputs, "none")
	inputs = append(inputs, getMidiInputs()...)
	mc.inputSelect.SetOptions(inputs)
	mc.inputSelect.Refresh()

	outputs := make([]string, 0)
	outputs = append(outputs, "none")
	outputs = append(outputs, getMidiOutputs()...)
	mc.outputSelect.SetOptions(outputs)
	mc.outputSelect.Refresh()

	mc.inputSelect.SetSelected(mc.newConfig.MidiInputPort)
	mc.outputSelect.SetSelected(mc.newConfig.MidiOutputPort)

}

// get a list of midi outputs
func getMidiOutputs() []string {
	outs := midi.GetOutPorts()
	var names []string
	for _, output := range outs {
		names = append(names, output.String())
	}
	return names
}

// get a list of midi inputs
func getMidiInputs() []string {
	ins := midi.GetInPorts()
	var names []string
	for _, input := range ins {
		names = append(names, input.String())
	}
	return names
}

// get a list of MCU Swtiches
func getMcuSwtiches() []string {
	return gomcu.Names
}

// get a list of MCU Channel
func getMcuChannels() []string {
	return gomcu.ChannelNames
}
