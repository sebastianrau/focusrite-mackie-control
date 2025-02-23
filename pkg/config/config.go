package config

import (
	"os"
	"time"

	fcaudioconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-connector"
	mcuconnector "github.com/sebastianrau/focusrite-mackie-control/pkg/mcu-connector"
	"github.com/snksoft/crc"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"

	"gopkg.in/yaml.v2"
)

var log *logger.CustomLogger = logger.WithPackage("config")

const (
	subfolder    string        = "Monitor-Controller"
	filename     string        = "monitor-controller.yaml"
	autoSaveTime time.Duration = 1 * time.Minute
)

type Config struct {
	Midi              *mcuconnector.McuConnectorConfig
	FocusriteDevice   *fcaudioconnector.FcConfiguration
	MonitorController *monitorcontroller.ControllerSate
	crc               uint64 `yaml:"-"`
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
	c := &Config{
		Midi:              mcuconnector.DefaultConfiguration(),
		FocusriteDevice:   fcaudioconnector.DefaultConfiguration(),
		MonitorController: monitorcontroller.NewDefaultState(),
	}
	return c
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

func (c *Config) RunAutoSave() {
	t := time.NewTicker(autoSaveTime)
	for range t.C {
		if c.UpdateChanged() {
			err := c.Save()
			if err != nil {
				log.Error(err)
			}
			log.Debugf("Auto save done.")
		} else {
			log.Debug("No change. Autosave skipped")
		}
	}
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

	log.Debugf("Config file saved to %s", path)
	return nil
}

func (c *Config) UpdateChanged() bool {
	buf, err := yaml.Marshal(c)
	if err != nil {
		log.Error(err)
	}
	crc := crc.CalculateCRC(crc.CCITT, buf)
	change := c.crc != crc
	c.crc = crc
	log.Debugf("CRC is 0x%04X - changed: %t", crc, change) // prints "CRC is 0x29B1"
	return change
}

func (c *Config) DeepCopy() (*Config, error) {
	var dst Config

	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &dst)
	return &dst, err
}
