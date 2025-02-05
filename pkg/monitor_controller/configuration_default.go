package monitorcontroller

import "github.com/sebastianrau/gomcu"

var (
	DEFAULT Configuration = Configuration{

		Speaker: map[int]*SpeakerConfig{

			SpeakerA: {
				Name: MappingString{
					FcIdsList: []FocusriteId{1456},
				},
				Mute: MappingBool{
					FcIdsList:      []FocusriteId{1453},
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignTrack,*/ gomcu.Trim},
					Value:          true,
				},
				OutputGain: MappingInt{
					FcIdsList: []FocusriteId{1458},
				},
				Type:      Speaker,
				Exclusive: true,
			},

			SpeakerB: {
				Name: MappingString{
					FcIdsList: []FocusriteId{},
				},
				Mute: MappingBool{
					FcIdsList:      []FocusriteId{},
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignSend,*/ gomcu.Touch},
					Value:          true,
				},
				Type:      Speaker,
				Exclusive: true,
			},
			SpeakerC: {
				Name: MappingString{
					FcIdsList: []FocusriteId{},
				},
				Mute: MappingBool{
					FcIdsList:      []FocusriteId{},
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignPan,*/ gomcu.Write},
					Value:          true,
				},
				Type:      Speaker,
				Exclusive: true,
			},
			SpeakerD: {
				Name: MappingString{
					FcIdsList: []FocusriteId{},
				},
				Mute: MappingBool{
					FcIdsList:      []FocusriteId{},
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignPlugin*/ },
					Value:          true,
				},
				Type:      Speaker,
				Exclusive: true,
			},
			SubA: {
				Name: MappingString{
					FcIdsList: []FocusriteId{},
				},
				Mute: MappingBool{
					FcIdsList:      []FocusriteId{},
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignEQ,*/ gomcu.Read},
					Value:          true,
				},
				Type:      Subwoofer,
				Exclusive: true,
			},
			SubB: {
				Name: MappingString{
					FcIdsList: []FocusriteId{},
				},
				Mute: MappingBool{
					FcIdsList:      []FocusriteId{},
					McuButtonsList: []gomcu.Switch{ /*gomcu.AssignInstrument*/ },
					Value:          true,
				},
				Type:      Subwoofer,
				Exclusive: true,
			},
		},
		Master: MasterConfig{
			MuteSwitch: MappingBool{
				McuButtonsList: []gomcu.Switch{gomcu.Mute1, gomcu.Mute2, gomcu.Mute3, gomcu.Mute4, gomcu.Mute5, gomcu.Mute6, gomcu.Mute7, gomcu.Mute8},
				FcIdsList:      []FocusriteId{1679},
			},
			DimSwitch: MappingBool{
				McuButtonsList: []gomcu.Switch{gomcu.Solo1, gomcu.Solo2, gomcu.Solo3, gomcu.Solo4, gomcu.Solo5, gomcu.Solo6, gomcu.Solo7, gomcu.Solo8},
				FcIdsList:      []FocusriteId{1678},
			},
			/* TODO add Meter Level
			Meter: MappingInt{
				McuButtonsList: []gomcu.Switch{},
				FcIdsList:      []FocusriteId{},
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
