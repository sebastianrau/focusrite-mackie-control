package monitorcontroller

type ControllerSate struct {
	Speaker map[SpeakerID]*SpeakerState
	Master  *MasterState
}

type SpeakerState struct {
	Disabled  bool
	Name      string
	Selected  bool
	Type      SpeakerType
	Exclusive bool
}

type MasterState struct {
	Mute bool
	Dim  bool

	VolumeDB   int
	LevelLeft  int
	LevelRight int
	DimOffset  int
}

func NewDefaultState() *ControllerSate {
	s := &ControllerSate{
		Master: &MasterState{
			Mute:       true,
			Dim:        false,
			VolumeDB:   -127,
			LevelLeft:  -127,
			LevelRight: -127,
			DimOffset:  20,
		},
		Speaker: make(map[SpeakerID]*SpeakerState),
	}

	for spkId, name := range SpeakerName {
		s.Speaker[spkId] = &SpeakerState{
			Name:      name,
			Selected:  false,
			Type:      Speaker,
			Exclusive: true,
		}
	}

	s.Speaker[SpeakerA].Selected = true
	s.Speaker[SpeakerD].Disabled = true
	s.Speaker[Sub].Selected = true
	s.Speaker[Sub].Type = Subwoofer

	return s

}
