package config

import (
	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// legacyConfig is the unversioned file written by releases up to 0.1.x
// (yaml.v2 with lower-cased field names and maps keyed by speaker index).
type legacyConfig struct {
	Midi struct {
		MidiInputPort       string        `yaml:"MidiInputPort"`
		MidiOutputPort      string        `yaml:"MidiOnputPort"`
		SpeakerSelect       map[int]uint8 `yaml:"speakerselect"`
		MasterMuteSwitch    uint8         `yaml:"mastermuteswitch"`
		MasterDimSwitch     uint8         `yaml:"masterdimswitch"`
		MasterVolumeChannel uint8         `yaml:"mastervolumechannel"`
		FaderScaleLog       bool          `yaml:"faderscalelog"`
	} `yaml:"midi"`
	FocusriteDevice struct {
		Speaker map[int]struct {
			Name       int `yaml:"name"`
			Mute       int `yaml:"mute"`
			OutputGain int `yaml:"outputgain"`
			MeterL     int `yaml:"meterl"`
			MeterR     int `yaml:"meterr"`
		} `yaml:"speaker"`
		Master struct {
			MuteSwitch int `yaml:"muteswitch"`
			DimSwitch  int `yaml:"dimswitch"`
		} `yaml:"master"`
		SerialNumber string `yaml:"focusriteserialnumber"`
	} `yaml:"focusritedevice"`
	MonitorController struct {
		Speaker map[int]struct {
			Disabled  bool   `yaml:"disabled"`
			Name      string `yaml:"name"`
			Selected  bool   `yaml:"selected"`
			Exclusive bool   `yaml:"exclusive"`
		} `yaml:"speaker"`
		Master struct {
			Mute      bool `yaml:"mute"`
			Dim       bool `yaml:"dim"`
			VolumeDB  int  `yaml:"volumedb"`
			DimOffset int  `yaml:"dimoffset"`
		} `yaml:"master"`
	} `yaml:"monitorcontroller"`
}

func (l legacyConfig) migrate() Config {
	c := Default()

	mc := l.MonitorController
	c.Controller.Mute = mc.Master.Mute
	c.Controller.Dim = mc.Master.Dim
	c.Controller.VolumeDB = mc.Master.VolumeDB
	c.Controller.DimDB = mc.Master.DimOffset
	for i := range domain.NumSpeakers {
		if s, ok := mc.Speaker[i]; ok {
			c.Controller.Speakers[i] = Speaker{Name: s.Name, Selected: s.Selected, Disabled: s.Disabled, Exclusive: s.Exclusive}
		}
	}

	fd := l.FocusriteDevice
	c.Focusrite.SerialNumber = fd.SerialNumber
	c.Focusrite.MuteID = fd.Master.MuteSwitch
	c.Focusrite.DimID = fd.Master.DimSwitch
	for i := range domain.NumSpeakers {
		if o, ok := fd.Speaker[i]; ok {
			c.Focusrite.Outputs[i] = Output{NameID: o.Name, MuteID: o.Mute, GainID: o.OutputGain, MeterLeftID: o.MeterL, MeterRightID: o.MeterR}
		}
	}

	m := l.Midi
	c.MIDI.InputPort = m.MidiInputPort
	c.MIDI.OutputPort = m.MidiOutputPort
	c.MIDI.FaderChannel = Channel(m.MasterVolumeChannel)
	c.MIDI.MuteKey = Key(m.MasterMuteSwitch)
	c.MIDI.DimKey = Key(m.MasterDimSwitch)
	c.MIDI.LogarithmicFade = m.FaderScaleLog
	for i := range domain.NumSpeakers {
		if k, ok := m.SpeakerSelect[i]; ok {
			c.MIDI.SpeakerKeys[i] = Key(k)
		}
	}
	return c
}
