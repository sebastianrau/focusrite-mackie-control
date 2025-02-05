package monitorcontroller

import "github.com/sebastianrau/gomcu"

var (
	DEFAULT Configuration = Configuration{

		Speaker: map[int]*SpeakerConfig{

			SpeakerA: {
				Name: MappingString{
					FcId: 1456,
				},
				Mute: MappingBool{
					FcId:           1453,
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignTrack,*/ gomcu.Trim},
					Value:          true,
				},
				OutputGain: MappingInt{
					FcId: 1458,
				},
				Meter: MappingInt{
					FcId:           1450,
					McuButtonsList: []gomcu.Switch{},
				},
				Type:      Speaker,
				Exclusive: true,
			},

			SpeakerB: {
				Name: MappingString{
					FcId: 1476,
				},
				Mute: MappingBool{
					FcId:           1473,
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignSend,*/ gomcu.Touch},
					Value:          true,
				},
				OutputGain: MappingInt{
					FcId: 1478,
				},
				Meter: MappingInt{
					McuButtonsList: []gomcu.Switch{},
					FcId:           1470,
				},
				Type:      Speaker,
				Exclusive: true,
			},
			SpeakerC: {
				Name: MappingString{
					FcId: 1496,
				},
				Mute: MappingBool{
					FcId:           1493,
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignPan,*/ gomcu.Write},
				},
				OutputGain: MappingInt{
					FcId: 1498,
				},
				Meter: MappingInt{
					FcId: 1490,
				},
				Type:      Speaker,
				Exclusive: true,
			},
			SpeakerD: {
				Name: MappingString{
					FcId: 1518,
				},
				Mute: MappingBool{
					FcId:           1513,
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignPlugin*/ },
					Value:          true,
				},
				OutputGain: MappingInt{
					FcId: 1518,
				},
				Meter: MappingInt{
					FcId: 1510,
				},
				Type:      Speaker,
				Exclusive: true,
			},
			SubA: {
				Name: MappingString{
					FcId: 1536,
				},
				Mute: MappingBool{
					FcId:           1533,
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignEQ,*/ gomcu.Read},
					Value:          true,
				},
				OutputGain: MappingInt{
					FcId: 1538,
				},
				Meter: MappingInt{
					FcId: 1530,
				},
				Type:      Subwoofer,
				Exclusive: false,
			},
			SubB: {
				Name: MappingString{
					FcId: 0,
				},
				Mute: MappingBool{
					FcId:           0,
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignInstrument*/ },
					Value:          true,
				},
				Type:      Subwoofer,
				Exclusive: false,
			},
		},
		Master: MasterConfig{
			MuteSwitch: MappingBool{
				McuButtonsList: []gomcu.Switch{gomcu.Mute1, gomcu.Mute2, gomcu.Mute3, gomcu.Mute4, gomcu.Mute5, gomcu.Mute6, gomcu.Mute7, gomcu.Mute8},
				FcId:           1679,
			},
			DimSwitch: MappingBool{
				McuButtonsList: []gomcu.Switch{gomcu.Solo1, gomcu.Solo2, gomcu.Solo3, gomcu.Solo4, gomcu.Solo5, gomcu.Solo6, gomcu.Solo7, gomcu.Solo8},
				FcId:           1678,
			},
			/* TODO add Meter Level
			Meter: MappingInt{
				McuButtonsList: []gomcu.Switch{},
				FcId:      ,
			},
			*/
			VolumeMcuChannel: []gomcu.Channel{gomcu.Channel1, gomcu.Channel2, gomcu.Channel3, gomcu.Channel4, gomcu.Channel5, gomcu.Channel6, gomcu.Channel7, gomcu.Channel8, gomcu.Master},
			DimVolumeOffset:  20.0,
		},
		FocusriteSerialNumber: "P9EAC6K250F325", // my 18i20
	}
)

// TODO For Testing only
func DefaultConfiguration() *Configuration {
	return &DEFAULT
}
