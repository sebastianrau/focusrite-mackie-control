package fcaudioconnector

import "github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"

type FocusriteId int

type SpeakerFcConfig struct {
	Name       FocusriteId
	Mute       FocusriteId
	OutputGain FocusriteId
	MeterL     FocusriteId
	MeterR     FocusriteId
}

type MasterFcConfig struct {
	MuteSwitch FocusriteId
	DimSwitch  FocusriteId
}

type FcConfiguration struct {
	Speaker map[monitorcontroller.SpeakerID]*SpeakerFcConfig
	Master  *MasterFcConfig

	FocusriteSerialNumber string
	FocusriteDeviceId     int `yaml:"-"`
}

func DefaultConfiguration() *FcConfiguration {
	return &FcConfiguration{

		Speaker: map[monitorcontroller.SpeakerID]*SpeakerFcConfig{

			monitorcontroller.SpeakerA: {
				Name:       1456,
				Mute:       1453,
				OutputGain: 1458,
				MeterL:     1450,
				MeterR:     1460,
			},
			monitorcontroller.SpeakerB: {
				Name:       1476,
				Mute:       1473,
				OutputGain: 1478,
				MeterL:     1470,
				MeterR:     1480,
			},
			monitorcontroller.SpeakerC: {
				Name:       1496,
				Mute:       1493,
				OutputGain: 1498,
				MeterL:     1490,
				MeterR:     1500,
			},
			monitorcontroller.SpeakerD: {
				Name:       1516,
				Mute:       1513,
				OutputGain: 1518,
				MeterL:     1510,
				MeterR:     1520,
			},
			monitorcontroller.Sub: {
				Name:       1536,
				Mute:       1533,
				OutputGain: 1538,
				MeterL:     1530,
				MeterR:     1540,
			},
		},
		Master: &MasterFcConfig{
			MuteSwitch: 1679,
			DimSwitch:  1678,
		},
		FocusriteSerialNumber: "P9EAC6K250F325",
	}
}
