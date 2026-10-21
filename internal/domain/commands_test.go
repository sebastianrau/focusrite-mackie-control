package domain

import "testing"

func snap() Snapshot { return Snapshot{State: DefaultState(), Levels: Silence} }

func selected(s Snapshot) []SpeakerID {
	var ids []SpeakerID
	for _, id := range Speakers {
		if s.State.Speakers[id].Selected {
			ids = append(ids, id)
		}
	}
	return ids
}

func equalIDs(a, b []SpeakerID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSelectSpeaker(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Snapshot)
		cmd   Command
		want  []SpeakerID
	}{
		{
			name: "exclusive speaker replaces the current main speaker but keeps the sub",
			cmd:  SelectSpeaker{ID: SpeakerB, On: true},
			want: []SpeakerID{SpeakerB, Sub},
		},
		{
			name: "non-exclusive speaker only deselects exclusive ones",
			setup: func(s *Snapshot) {
				s.State.Speakers[SpeakerB].Exclusive = false
				s.State.Speakers[SpeakerC].Exclusive = false
				s.State.Speakers[SpeakerC].Selected = true
			},
			cmd:  SelectSpeaker{ID: SpeakerB, On: true},
			want: []SpeakerID{SpeakerB, SpeakerC, Sub},
		},
		{
			name: "disabled speaker ignores selection",
			cmd:  SelectSpeaker{ID: SpeakerD, On: true},
			want: []SpeakerID{SpeakerA, Sub},
		},
		{
			name: "deselect leaves others alone",
			cmd:  SelectSpeaker{ID: SpeakerA, On: false},
			want: []SpeakerID{Sub},
		},
		{
			name: "toggle selects",
			cmd:  ToggleSpeaker{ID: SpeakerC},
			want: []SpeakerID{SpeakerC, Sub},
		},
		{
			name: "toggle sub off",
			cmd:  ToggleSpeaker{ID: Sub},
			want: []SpeakerID{SpeakerA},
		},
		{
			name: "invalid id is ignored",
			cmd:  SelectSpeaker{ID: SpeakerID(42), On: true},
			want: []SpeakerID{SpeakerA, Sub},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := snap()
			if tt.setup != nil {
				tt.setup(&s)
			}
			got := selected(Apply(s, tt.cmd))
			if !equalIDs(got, tt.want) {
				t.Errorf("selected = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeselectingLastMainSpeakerSilencesMeter(t *testing.T) {
	s := snap()
	s.Levels = Levels{LeftDB: -10, RightDB: -12}
	s = Apply(s, SelectSpeaker{ID: SpeakerA, On: false})
	if s.Levels != Silence {
		t.Errorf("levels = %+v, want silence", s.Levels)
	}
}

func TestApplyDoesNotMutateInput(t *testing.T) {
	s := snap()
	_ = Apply(s, ToggleMute{})
	_ = Apply(s, SelectSpeaker{ID: SpeakerB, On: true})
	if s != snap() {
		t.Error("Apply modified its input")
	}
}

func TestMasterCommands(t *testing.T) {
	s := snap()
	s = Apply(s, ToggleMute{})
	if s.State.Master.Mute {
		t.Error("ToggleMute did not unmute")
	}
	s = Apply(s, SetDim{On: true})
	s = Apply(s, ToggleDim{})
	if s.State.Master.Dim {
		t.Error("ToggleDim did not undim")
	}
	for _, tc := range []struct{ in, want int }{{-20, -20}, {5, 0}, {-500, MinVolumeDB}} {
		if got := Apply(s, SetVolume{DB: tc.in}).State.Master.VolumeDB; got != tc.want {
			t.Errorf("SetVolume(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestSetDeviceAbsentSilencesMeter(t *testing.T) {
	s := snap()
	s.Levels = Levels{LeftDB: -3, RightDB: -3}
	s = Apply(s, SetDevice{Device: Device{ServerConnected: true}})
	if s.Levels != Silence {
		t.Errorf("levels = %+v, want silence", s.Levels)
	}
}

func TestOutputGainDB(t *testing.T) {
	tests := []struct {
		volume, dimDB int
		dim           bool
		want          int
	}{
		{volume: -20, want: -20},
		{volume: -20, dim: true, dimDB: 20, want: -40},
		{volume: -120, dim: true, dimDB: 20, want: MinVolumeDB},
		{volume: 6, want: 0},
	}
	for _, tt := range tests {
		s := DefaultState()
		s.Master.VolumeDB, s.Master.Dim, s.Master.DimDB = tt.volume, tt.dim, tt.dimDB
		if got := s.OutputGainDB(); got != tt.want {
			t.Errorf("OutputGainDB(vol=%d dim=%t/%d) = %d, want %d", tt.volume, tt.dim, tt.dimDB, got, tt.want)
		}
	}
}

func TestMeterSource(t *testing.T) {
	s := DefaultState()
	if id, ok := s.MeterSource(); !ok || id != SpeakerA {
		t.Errorf("MeterSource = %v, %t; want SpeakerA", id, ok)
	}
	s.Speakers[SpeakerA].Selected = false
	if id, ok := s.MeterSource(); ok {
		t.Errorf("MeterSource = %v with only the sub selected, want none", id)
	}
}

func TestOutputMuted(t *testing.T) {
	s := DefaultState()
	s.Master.Mute = false
	if s.OutputMuted(SpeakerA) || !s.OutputMuted(SpeakerB) {
		t.Error("selected speaker must play, unselected must be muted")
	}
	s.Master.Mute = true
	if !s.OutputMuted(SpeakerA) {
		t.Error("master mute must mute selected speakers")
	}
}

func TestSpeakerIDString(t *testing.T) {
	want := []string{"Speaker A", "Speaker B", "Speaker C", "Speaker D", "Sub"}
	for i, id := range Speakers {
		if id.String() != want[i] {
			t.Errorf("%d.String() = %q, want %q", int(id), id.String(), want[i])
		}
	}
}

func TestConfigureSpeaker(t *testing.T) {
	s := snap()
	s = Apply(s, ConfigureSpeaker{ID: SpeakerA, Disabled: true, Exclusive: true})
	if s.State.Speakers[SpeakerA].Selected || !s.State.Speakers[SpeakerA].Disabled {
		t.Errorf("speaker A = %+v, want disabled and deselected", s.State.Speakers[SpeakerA])
	}
	if s.Levels != Silence {
		t.Error("disabling the meter source must silence the meter")
	}
	s = Apply(s, ConfigureSpeaker{ID: SpeakerD, Disabled: false, Exclusive: false})
	if s.State.Speakers[SpeakerD].Disabled || s.State.Speakers[SpeakerD].Exclusive {
		t.Errorf("speaker D = %+v", s.State.Speakers[SpeakerD])
	}
}

func TestSetDimDB(t *testing.T) {
	for _, tc := range []struct{ in, want int }{{15, 15}, {-5, 0}, {99, MaxDimDB}} {
		if got := Apply(snap(), SetDimDB{DB: tc.in}).State.Master.DimDB; got != tc.want {
			t.Errorf("SetDimDB(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
