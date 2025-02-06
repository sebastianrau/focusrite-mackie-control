package mcu

type Configuration struct {
	MidiInputPort  string `yaml:"MidiInputPort"`
	MidiOutputPort string `yaml:"MidiOnputPort"`
}

var (
	DEFAULT_CONFIGURATION Configuration = Configuration{
		MidiInputPort:  "PreSonus FP2",
		MidiOutputPort: "PreSonus FP2",
	}
)
