package config

import (
	"os"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/mcu"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"

	"gopkg.in/yaml.v2"
)

var log *logger.CustomLogger = logger.WithPackage("focusrite-config")

const (
	subfolder string = "Monitor-Controller"
	filename  string = "monitor-controller.yaml"
)

type Config struct {
	Midi       *mcu.Configuration
	Controller *monitorcontroller.Configuration
}

func getPath() (string, error) {
	userFolder, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return userFolder + "/" + subfolder + "/", nil
}

func getPathAndFile() (string, error) {
	path, err := getPath()
	if err != nil {
		return "", err
	}

	return path + filename, nil
}

func Default() *Config {
	return &Config{
		Midi:       &mcu.DEFAULT_CONFIGURATION,
		Controller: &monitorcontroller.DEFAULT_CONFIGURATION,
	}
}

func Load() (*Config, error) {

	var config Config

	path, err := getPathAndFile()
	if err != nil {
		return nil, err
	}

	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Save() error {

	path, err := getPath()
	if err != nil {
		return err
	}
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	filename, _ := getPathAndFile()
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	buf, err := yaml.Marshal(*c)
	if err != nil {
		return err
	}
	_, err = file.Write(buf)
	if err != nil {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}

/*
func UserConfigure() (*Config, bool) {

	config := &Config{}

	fmt.Println("*** CONFIGURING MIDI ***")
	fmt.Println("")
	inputs := getMidiInputs()
	for i, v := range inputs {
		fmt.Printf("MIDI Input %v: %s\n", i+1, v)
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Println()
	fmt.Print("Enter MIDI input port number and press [enter]: ")
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	num, err := strconv.Atoi(text)
	if err != nil || num <= 0 || num > len(inputs) {
		fmt.Println("Please enter only valid numbers")
		return nil, false
	}
	config.MidiInputPort = inputs[num-1]

	fmt.Println()
	outputs := getMidiOutputs()
	for i, v := range outputs {
		fmt.Printf("MIDI Output %v: %s\n", i+1, v)
	}
	fmt.Println()
	fmt.Print("Enter MIDI output port number and press [enter]: ")
	text, _ = reader.ReadString('\n')
	text = strings.TrimSpace(text)
	num, err = strconv.Atoi(text)
	if err != nil || num <= 0 || num > len(outputs) {
		fmt.Println("Please enter only valid numbers")
		return nil, false
	}
	config.MidiOutputPort = outputs[num-1]

	err = config.Save()
	if err != nil {
		log.Error(err.Error())
		return nil, false
	}

	return config, true
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
*/
