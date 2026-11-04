package domain

// Command is a request to change the controller state. Adapters submit
// commands; only the controller applies them.
type Command interface {
	apply(s *Snapshot)
}

// Apply returns the snapshot that results from applying cmd to s. Every
// change is stored in the active scene as well.
func Apply(s Snapshot, cmd Command) Snapshot {
	cmd.apply(&s)
	if s.State.Scene.Valid() {
		s.State.Scenes[s.State.Scene] = s.State.CurrentScene()
	}
	return s
}

type SetMute struct{ On bool }

func (c SetMute) apply(s *Snapshot) { s.State.Master.Mute = c.On }

type ToggleMute struct{}

func (ToggleMute) apply(s *Snapshot) { s.State.Master.Mute = !s.State.Master.Mute }

type SetDim struct{ On bool }

func (c SetDim) apply(s *Snapshot) { s.State.Master.Dim = c.On }

type ToggleDim struct{}

func (ToggleDim) apply(s *Snapshot) { s.State.Master.Dim = !s.State.Master.Dim }

// SetVolume sets the master volume; values outside the valid range are clamped.
type SetVolume struct{ DB int }

func (c SetVolume) apply(s *Snapshot) {
	s.State.Master.VolumeDB = min(max(c.DB, MinVolumeDB), MaxVolumeDB)
}

// SelectSpeaker selects or deselects a speaker. Selecting an exclusive
// speaker deselects all others of the same kind; selecting a non-exclusive one
// deselects only the exclusive ones. Disabled speakers ignore the command.
type SelectSpeaker struct {
	ID SpeakerID
	On bool
}

func (c SelectSpeaker) apply(s *Snapshot) {
	if !c.ID.Valid() || s.State.Speakers[c.ID].Disabled {
		return
	}
	spk := &s.State.Speakers[c.ID]
	spk.Selected = c.On
	if c.On {
		for _, other := range Speakers {
			o := &s.State.Speakers[other]
			if other != c.ID && other.IsSub() == c.ID.IsSub() && (spk.Exclusive || o.Exclusive) {
				o.Selected = false
			}
		}
	}
	if _, ok := s.State.MeterSource(); !ok {
		s.Levels = Silence
	}
}

type ToggleSpeaker struct{ ID SpeakerID }

func (c ToggleSpeaker) apply(s *Snapshot) {
	if c.ID.Valid() {
		SelectSpeaker{ID: c.ID, On: !s.State.Speakers[c.ID].Selected}.apply(s)
	}
}

type RenameSpeaker struct {
	ID   SpeakerID
	Name string
}

func (c RenameSpeaker) apply(s *Snapshot) {
	if c.ID.Valid() {
		s.State.Speakers[c.ID].Name = c.Name
	}
}

type SetLevels struct{ Levels Levels }

func (c SetLevels) apply(s *Snapshot) { s.Levels = c.Levels }

// SetDevice reports a new connection or interface status. Levels drop to
// silence while no interface is present.
type SetDevice struct{ Device Device }

func (c SetDevice) apply(s *Snapshot) {
	s.Device = c.Device
	if !c.Device.Present {
		s.Levels = Silence
	}
}

// ConfigureSpeaker changes how a speaker takes part in selection. Disabling
// a speaker deselects it.
type ConfigureSpeaker struct {
	ID        SpeakerID
	Disabled  bool
	Exclusive bool
}

func (c ConfigureSpeaker) apply(s *Snapshot) {
	if !c.ID.Valid() {
		return
	}
	spk := &s.State.Speakers[c.ID]
	spk.Disabled, spk.Exclusive = c.Disabled, c.Exclusive
	if c.Disabled {
		spk.Selected = false
		if _, ok := s.State.MeterSource(); !ok {
			s.Levels = Silence
		}
	}
}

// SetMeterPost switches the meters between pre-fader and post-fader.
type SetMeterPost struct{ On bool }

func (c SetMeterPost) apply(s *Snapshot) { s.State.Master.MeterPost = c.On }

type ToggleMeterPost struct{}

func (ToggleMeterPost) apply(s *Snapshot) { s.State.Master.MeterPost = !s.State.Master.MeterPost }

const MaxDimDB = 60

// SetDimDB sets the dim attenuation, clamped to 0..MaxDimDB.
type SetDimDB struct{ DB int }

func (c SetDimDB) apply(s *Snapshot) { s.State.Master.DimDB = min(max(c.DB, 0), MaxDimDB) }

// SelectScene switches to a scene and recalls it.
type SelectScene struct{ ID SceneID }

func (c SelectScene) apply(s *Snapshot) {
	if !c.ID.Valid() {
		return
	}
	s.State.Scene = c.ID
	s.State.recall(s.State.Scenes[c.ID])
	if _, ok := s.State.MeterSource(); !ok {
		s.Levels = Silence
	}
}

// ToggleScene switches between scene A and B.
type ToggleScene struct{}

func (ToggleScene) apply(s *Snapshot) {
	SelectScene{ID: (s.State.Scene + 1) % SceneID(NumScenes)}.apply(s)
}

// StoreScene stores the live state in a scene without switching to it.
type StoreScene struct{ ID SceneID }

func (c StoreScene) apply(s *Snapshot) {
	if c.ID.Valid() {
		s.State.Scenes[c.ID] = s.State.CurrentScene()
	}
}
