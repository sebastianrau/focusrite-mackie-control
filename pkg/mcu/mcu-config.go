package mcu

type Configuration struct {
	MidiInputPort  string `yaml:"MidiInputPort"`
	MidiOutputPort string `yaml:"MidiOnputPort"`
}

var (
	DEFAULT_CONFIGURATION Configuration = Configuration{
		MidiInputPort:  "",
		MidiOutputPort: "",
	}
)
