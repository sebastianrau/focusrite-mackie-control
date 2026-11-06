// Package config defines the settings file and keeps it on disk.
package config

import (
	"fmt"
	"strconv"

	"github.com/sebastianrau/gomcu"
	"gopkg.in/yaml.v3"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite"
	"github.com/sebastianrau/focusrite-mackie-control/internal/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/internal/osc"
)

// Version is the current file format. Files without a version are from
// releases up to 0.1.x and are migrated on load.
const Version = 2

// Config is the settings file.
type Config struct {
	Version    int        `yaml:"version"`
	Controller Controller `yaml:"controller"`
	Focusrite  Focusrite  `yaml:"focusrite"`
	MIDI       MIDI       `yaml:"midi"`
	OSC        OSC        `yaml:"osc"`
}

// Controller holds the monitor state restored at startup.
type Controller struct {
	Mute      bool `yaml:"mute"`
	Dim       bool `yaml:"dim"`
	VolumeDB  int  `yaml:"volume_db"`
	DimDB     int  `yaml:"dim_db"`
	MeterPost bool `yaml:"meter_post"`
	// MeterFloorDB is the meter range; files without it get the default.
	MeterFloorDB int       `yaml:"meter_floor_db"`
	Speakers     []Speaker `yaml:"speakers"` // A, B, C, D, Sub
	Scene        string    `yaml:"scene"`    // active scene: "A" or "B"
	Scenes       []Scene   `yaml:"scenes"`   // A, B
}

// Scene is a stored monitor scene. Files written before scenes existed
// get the stored state in both scenes.
type Scene struct {
	Mute     bool   `yaml:"mute"`
	Dim      bool   `yaml:"dim"`
	VolumeDB int    `yaml:"volume_db"`
	Speakers []bool `yaml:"speakers"` // selected: A, B, C, D, Sub
}

type Speaker struct {
	Name      string `yaml:"name"`
	Selected  bool   `yaml:"selected"`
	Disabled  bool   `yaml:"disabled"`
	Exclusive bool   `yaml:"exclusive"`
}

// Focusrite selects the interface and its element IDs.
type Focusrite struct {
	SerialNumber string   `yaml:"serial_number"`
	MuteID       int      `yaml:"mute_id"`
	DimID        int      `yaml:"dim_id"`
	Outputs      []Output `yaml:"outputs"` // A, B, C, D, Sub
}

type Output struct {
	NameID       int `yaml:"name_id"`
	MuteID       int `yaml:"mute_id"`
	GainID       int `yaml:"gain_id"`
	MeterLeftID  int `yaml:"meter_left_id"`
	MeterRightID int `yaml:"meter_right_id"`
}

// MIDI maps controller functions to surface controls.
type MIDI struct {
	InputPort       string  `yaml:"input_port"`
	OutputPort      string  `yaml:"output_port"`
	FaderChannel    Channel `yaml:"fader_channel"`
	MuteKey         Key     `yaml:"mute_key"`
	DimKey          Key     `yaml:"dim_key"`
	MeterPostKey    Key     `yaml:"meter_post_key"`
	SceneKey        Key     `yaml:"scene_key"`
	SpeakerKeys     []Key   `yaml:"speaker_keys"` // A, B, C, D, Sub
	LogarithmicFade bool    `yaml:"logarithmic_fader"`
}

func Default() Config {
	var c Config
	c.Version = Version
	c.SetState(domain.DefaultState())
	c.SetFocusrite(focusrite.Config{})
	c.SetMCU(mcu.DefaultConfig())
	c.SetOSC(true, osc.DefaultConfig())
	return c
}

// State returns the stored monitor state. Missing speakers get defaults.
func (c Config) State() domain.State {
	s := domain.DefaultState()
	s.Master = domain.Master{Mute: c.Controller.Mute, Dim: c.Controller.Dim, VolumeDB: c.Controller.VolumeDB,
		DimDB: c.Controller.DimDB, MeterPost: c.Controller.MeterPost,
		MeterFloorDB: domain.ValidMeterFloor(c.Controller.MeterFloorDB)}
	for i, spk := range c.Controller.Speakers {
		if i < domain.NumSpeakers {
			s.Speakers[i] = domain.Speaker(spk)
		}
	}
	if c.Controller.Scene == "B" {
		s.Scene = domain.SceneB
	}
	for i := range s.Scenes {
		s.Scenes[i] = s.CurrentScene()
		if i < len(c.Controller.Scenes) {
			sc := c.Controller.Scenes[i]
			s.Scenes[i].Mute, s.Scenes[i].Dim, s.Scenes[i].VolumeDB = sc.Mute, sc.Dim, sc.VolumeDB
			for j := range s.Scenes[i].Selected {
				s.Scenes[i].Selected[j] = j < len(sc.Speakers) && sc.Speakers[j]
			}
		}
	}
	// The live state belongs to the active scene.
	s.Scenes[s.Scene] = s.CurrentScene()
	return s
}

func (c *Config) SetState(s domain.State) {
	c.Controller = Controller{Mute: s.Master.Mute, Dim: s.Master.Dim, VolumeDB: s.Master.VolumeDB,
		DimDB: s.Master.DimDB, MeterPost: s.Master.MeterPost, MeterFloorDB: s.Master.MeterFloorDB}
	for _, spk := range s.Speakers {
		c.Controller.Speakers = append(c.Controller.Speakers, Speaker(spk))
	}
	c.Controller.Scene = string(rune('A' + s.Scene))
	for _, sc := range s.Scenes {
		c.Controller.Scenes = append(c.Controller.Scenes, Scene{Mute: sc.Mute, Dim: sc.Dim, VolumeDB: sc.VolumeDB, Speakers: sc.Selected[:]})
	}
}

func (c Config) FocusriteConfig() focusrite.Config {
	f := focusrite.Config{SerialNumber: c.Focusrite.SerialNumber, MuteID: c.Focusrite.MuteID, DimID: c.Focusrite.DimID}
	for i, o := range c.Focusrite.Outputs {
		if i < domain.NumSpeakers {
			f.Outputs[i] = focusrite.Output(o)
		}
	}
	return f
}

func (c *Config) SetFocusrite(f focusrite.Config) {
	c.Focusrite = Focusrite{SerialNumber: f.SerialNumber, MuteID: f.MuteID, DimID: f.DimID}
	for _, o := range f.Outputs {
		c.Focusrite.Outputs = append(c.Focusrite.Outputs, Output(o))
	}
}

func (c Config) MCUConfig() mcu.Config {
	m := mcu.Config{
		InputPort:       c.MIDI.InputPort,
		OutputPort:      c.MIDI.OutputPort,
		FaderChannel:    gomcu.Channel(c.MIDI.FaderChannel),
		MuteKey:         gomcu.Switch(c.MIDI.MuteKey),
		DimKey:          gomcu.Switch(c.MIDI.DimKey),
		MeterPostKey:    gomcu.Switch(c.MIDI.MeterPostKey),
		SceneKey:        gomcu.Switch(c.MIDI.SceneKey),
		LogarithmicFade: c.MIDI.LogarithmicFade,
	}
	def := mcu.DefaultConfig()
	for i := range m.SpeakerKeys {
		if i < len(c.MIDI.SpeakerKeys) {
			m.SpeakerKeys[i] = gomcu.Switch(c.MIDI.SpeakerKeys[i])
		} else {
			m.SpeakerKeys[i] = def.SpeakerKeys[i]
		}
	}
	return m
}

func (c *Config) SetMCU(m mcu.Config) {
	c.MIDI = MIDI{
		InputPort:       m.InputPort,
		OutputPort:      m.OutputPort,
		FaderChannel:    Channel(m.FaderChannel),
		MuteKey:         Key(m.MuteKey),
		DimKey:          Key(m.DimKey),
		MeterPostKey:    Key(m.MeterPostKey),
		SceneKey:        Key(m.SceneKey),
		LogarithmicFade: m.LogarithmicFade,
	}
	for _, k := range m.SpeakerKeys {
		c.MIDI.SpeakerKeys = append(c.MIDI.SpeakerKeys, Key(k))
	}
}

// OSC configures the OSC server. Files written before OSC existed get the
// defaults (on, local only).
type OSC struct {
	Enabled      bool     `yaml:"enabled"`
	Port         int      `yaml:"port"`
	AllowNetwork bool     `yaml:"allow_network"`
	Targets      []string `yaml:"targets"`
}

// OSCConfig returns the server settings and whether the server should run.
func (c Config) OSCConfig() (osc.Config, bool) {
	return osc.Config{
		Port:         c.OSC.Port,
		AllowNetwork: c.OSC.AllowNetwork,
		Targets:      append([]string(nil), c.OSC.Targets...),
	}, c.OSC.Enabled
}

func (c *Config) SetOSC(enabled bool, o osc.Config) {
	c.OSC = OSC{Enabled: enabled, Port: o.Port, AllowNetwork: o.AllowNetwork, Targets: append([]string(nil), o.Targets...)}
}

// Key is an MCU button, stored by its name (for example "Mute1").
type Key gomcu.Switch

func (k Key) MarshalYAML() (any, error) {
	if int(k) < len(gomcu.Names) {
		return gomcu.Names[k], nil
	}
	return int(k), nil
}

func (k *Key) UnmarshalYAML(n *yaml.Node) error {
	if id, ok := gomcu.IDs[n.Value]; ok {
		*k = Key(id)
		return nil
	}
	v, err := strconv.ParseUint(n.Value, 0, 8)
	if err != nil {
		return fmt.Errorf("line %d: unknown MCU key %q", n.Line, n.Value)
	}
	*k = Key(v)
	return nil
}

// Channel is an MCU fader strip, stored by its name (for example "Channel1").
type Channel gomcu.Channel

func (c Channel) MarshalYAML() (any, error) {
	if int(c) < len(gomcu.ChannelNames) {
		return gomcu.ChannelNames[c], nil
	}
	return int(c), nil
}

func (c *Channel) UnmarshalYAML(n *yaml.Node) error {
	if id, ok := gomcu.ChannelIDs[n.Value]; ok {
		*c = Channel(id)
		return nil
	}
	v, err := strconv.ParseUint(n.Value, 0, 8)
	if err != nil || v >= gomcu.LenChannels {
		return fmt.Errorf("line %d: unknown MCU channel %q", n.Line, n.Value)
	}
	*c = Channel(v)
	return nil
}
