package monitorcontroller

import (
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

var (
	PLATFORM_NANO Configuration = Configuration{
		Speaker: map[int]Speaker{
			SpeakerA: {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignTrack}}, Type: SpeakerNormal, Exclusive: true},
			SpeakerB: {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignSend}}, Type: SpeakerNormal, Exclusive: true},
			SpeakerC: {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignPan}}, Type: SpeakerNormal, Exclusive: true},
			SpeakerD: {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignPlugin}}, Type: SpeakerNormal, Exclusive: true},
			SubA:     {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignEQ}}, Type: SpeakerNormal, Exclusive: true},
			SubB:     {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignInstrument}}, Type: SpeakerNormal, Exclusive: true},
		},
		Master: Master{
			Name: SpeakerNames[MasterFader],
			Mcu: McuMaster{
				Fader: gomcu.Master,
			},
			Focusrite: FocusriteMaster{
				Mute: focusritexml.ElementBool{ID: 1679},
			},
		},
		SerialNumber: focusritexml.ElementString{Value: "P9EAC6K250F325"},
	}

	FADERPORT_CFG Configuration = Configuration{
		Speaker: map[int]Speaker{
			SpeakerA: {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.Trim}}, Type: SpeakerNormal, Exclusive: true},
			SpeakerB: {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.Touch}}, Type: SpeakerNormal, Exclusive: false},
			SpeakerC: {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.Write}}, Type: SpeakerNormal, Exclusive: false},
			SpeakerD: {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{}}, Type: SpeakerNormal, Exclusive: false},
			SubA:     {Name: SpeakerNames[SubA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.Read}}, Type: SubwooferNormal, Exclusive: true},
			SubB:     {Name: SpeakerNames[SubB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{}}, Type: SubwooferNormal, Exclusive: true},
		},
		Master: Master{
			Name: SpeakerNames[MasterFader],
			Mcu: McuMaster{
				Fader: gomcu.Channel8,
			},
			Focusrite: FocusriteMaster{
				Mute: focusritexml.ElementBool{ID: 1679},
			},
		},
		SerialNumber: focusritexml.ElementString{Value: "P9EAC6K250F325"},
	}
)

// For Testing only
func DefaultConfiguration() *Configuration {
	//return &PLATFORM_NANO
	return &FADERPORT_CFG
}
