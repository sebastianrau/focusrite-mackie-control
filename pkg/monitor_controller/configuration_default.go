package monitorcontroller

import "github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"

var (
	PLATFORM_NANO Configuration = Configuration{
		Speaker: map[int]Speaker{
			SpeakerA: {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignTrack}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: true},
			SpeakerB: {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignSend}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: true},
			SpeakerC: {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignPan}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: true},
			SpeakerD: {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignPlugin}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: true},
			SubA:     {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignEQ}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: true},
			SubB:     {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.AssignInstrument}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: true},
		},
		Master: Master{
			Name: SpeakerNames[MasterFader],
			Mcu: McuMaster{
				Fader: gomcu.Master,
			},
		},
	}

	FADERPORT_CFG Configuration = Configuration{
		Speaker: map[int]Speaker{
			SpeakerA: {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.Trim}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: true},
			SpeakerB: {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.Touch}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: false},
			SpeakerC: {Name: SpeakerNames[SpeakerA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.Write}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: false},
			SpeakerD: {Name: SpeakerNames[SpeakerB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{}}, Focusrite: FocusriteConfiguration{}, Type: SpeakerNormal, Exclusive: false},
			SubA:     {Name: SpeakerNames[SubA], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{gomcu.Read}}, Focusrite: FocusriteConfiguration{}, Type: SubwooferNormal, Exclusive: true},
			SubB:     {Name: SpeakerNames[SubB], Mcu: McuConfiguration{EnableButton: []gomcu.Switch{}}, Focusrite: FocusriteConfiguration{}, Type: SubwooferNormal, Exclusive: true},
		},
		Master: Master{
			Name: SpeakerNames[MasterFader],
			Mcu: McuMaster{
				Fader: gomcu.Channel8,
			},
		},
	}
)

// For Testing only
func DefaultConfiguration() *Configuration {
	//return &PLATFORM_NANO
	return &FADERPORT_CFG
}
