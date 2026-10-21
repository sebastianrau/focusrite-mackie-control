// Package domain holds the monitor controller state and the rules that change
// it. Everything here is plain values and pure functions; the controller
// package owns the live state and the adapters translate to and from devices.
package domain

import "fmt"

// SpeakerID identifies one of the fixed speaker slots.
type SpeakerID int

const (
	SpeakerA SpeakerID = iota
	SpeakerB
	SpeakerC
	SpeakerD
	Sub

	NumSpeakers = int(Sub) + 1
)

// Speakers lists all speaker IDs in display order.
var Speakers = [NumSpeakers]SpeakerID{SpeakerA, SpeakerB, SpeakerC, SpeakerD, Sub}

func (id SpeakerID) Valid() bool { return id >= 0 && int(id) < NumSpeakers }

// IsSub reports whether the slot is the subwoofer. Exclusive selection only
// applies between speakers of the same kind, so selecting a main speaker never
// deselects the sub.
func (id SpeakerID) IsSub() bool { return id == Sub }

func (id SpeakerID) String() string {
	switch id {
	case SpeakerA, SpeakerB, SpeakerC, SpeakerD:
		return fmt.Sprintf("Speaker %c", 'A'+rune(id))
	case Sub:
		return "Sub"
	}
	return fmt.Sprintf("SpeakerID(%d)", int(id))
}

const (
	MinVolumeDB = -127
	MaxVolumeDB = 0

	// SilenceDB is the meter value reported when nothing is playing or no
	// speaker is selected.
	SilenceDB = -127.0
)

type Speaker struct {
	Name      string
	Selected  bool
	Disabled  bool
	Exclusive bool
}

type Master struct {
	Mute     bool
	Dim      bool
	VolumeDB int
	// DimDB is the attenuation applied while Dim is on.
	DimDB int
}

// State is the persistent part of the controller: what the user set.
type State struct {
	Master   Master
	Speakers [NumSpeakers]Speaker
}

func DefaultState() State {
	s := State{
		Master: Master{Mute: true, VolumeDB: MinVolumeDB, DimDB: 20},
	}
	for _, id := range Speakers {
		s.Speakers[id] = Speaker{Name: id.String(), Exclusive: true}
	}
	s.Speakers[SpeakerA].Selected = true
	s.Speakers[SpeakerD].Disabled = true
	s.Speakers[Sub].Selected = true
	return s
}

// OutputGainDB is the gain every enabled output is set to: the master volume
// minus the dim attenuation, clamped to the valid range.
func (s State) OutputGainDB() int {
	v := min(s.Master.VolumeDB, MaxVolumeDB)
	if s.Master.Dim {
		v -= s.Master.DimDB
	}
	return max(v, MinVolumeDB)
}

// OutputMuted reports whether the output of a speaker must be muted.
func (s State) OutputMuted(id SpeakerID) bool {
	return s.Master.Mute || !s.Speakers[id].Selected
}

// MeterSource returns the speaker whose output drives the level meter: the
// first selected, enabled main speaker.
func (s State) MeterSource() (SpeakerID, bool) {
	for _, id := range Speakers {
		spk := s.Speakers[id]
		if !id.IsSub() && spk.Selected && !spk.Disabled {
			return id, true
		}
	}
	return 0, false
}

// Levels is the current output meter reading in dBFS.
type Levels struct {
	LeftDB  float64
	RightDB float64
}

var Silence = Levels{LeftDB: SilenceDB, RightDB: SilenceDB}

// Device describes the connection to Focusrite Control and the configured
// interface.
type Device struct {
	// ServerConnected is true while connected to Focusrite Control.
	ServerConnected bool
	// Present is true while the configured interface is attached.
	Present      bool
	Model        string
	SerialNumber string
	SampleRate   string
}

// Snapshot is everything the controller publishes. It is a plain value and
// comparable with ==.
type Snapshot struct {
	State  State
	Levels Levels
	Device Device
}
