package monitorcontroller

type FocusriteId int

type SpeakerFcConfig struct {
	Name       FocusriteId
	Mute       FocusriteId
	OutputGain FocusriteId
	Meter      FocusriteId
}

type MasterFcConfig struct {
	MuteSwitch FocusriteId
	DimSwitch  FocusriteId
}

type FcConfiguration struct {
	Speaker map[SpeakerID]*SpeakerFcConfig
	Master  *MasterFcConfig

	FocusriteSerialNumber string
	FocusriteDeviceId     int
}

func DefaultConfiguration() *FcConfiguration {
	return &FcConfiguration{

		Speaker: map[SpeakerID]*SpeakerFcConfig{

			SpeakerA: {
				Name:       1456,
				Mute:       1453,
				OutputGain: 1458,
				Meter:      1450,
			},
			SpeakerB: {
				Name:       1476,
				Mute:       1473,
				OutputGain: 1478,
				Meter:      1470,
			},
			SpeakerC: {
				Name:       1496,
				Mute:       1493,
				Meter:      1490,
				OutputGain: 1498,
			},
			SpeakerD: {
				Name:       1516,
				Mute:       1513,
				OutputGain: 1518,
				Meter:      1510,
			},
			Sub: {
				Name:       1536,
				Mute:       1533,
				OutputGain: 1538,
				Meter:      1530,
			},
		},
		Master: &MasterFcConfig{
			MuteSwitch: 1679,
			DimSwitch:  1678,
		},
		FocusriteSerialNumber: "P9EAC6K250F325",
	}
}
