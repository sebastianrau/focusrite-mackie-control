package mcu

import (
	"slices"
	"testing"

	"github.com/sebastianrau/gomcu"
	"gitlab.com/gomidi/midi/v2"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

type fakeSurface struct{ sent []Command }

func (f *fakeSurface) Events() <-chan Event { return nil }
func (f *fakeSurface) Send(c Command)       { f.sent = append(f.sent, c) }

func (f *fakeSurface) take() []Command {
	c := f.sent
	f.sent = nil
	return c
}

type harness struct {
	a    *Adapter
	surf *fakeSurface
	cmds []domain.Command
}

// A custom mapping, so a regression to hard-coded defaults shows up.
var testConfig = Config{
	FaderChannel: gomcu.Channel3,
	MuteKey:      gomcu.Rec2,
	DimKey:       gomcu.Rec3,
	SpeakerKeys:  [domain.NumSpeakers]gomcu.Switch{gomcu.F1, gomcu.F2, gomcu.F3, gomcu.F4, gomcu.F5},
}

func newHarness() *harness {
	h := &harness{surf: &fakeSurface{}}
	h.a = NewAdapter(testConfig, h.surf, func(c domain.Command) { h.cmds = append(h.cmds, c) }, nil)
	h.snapshot(domain.Snapshot{State: domain.DefaultState(), Levels: domain.Silence})
	return h
}

func (h *harness) snapshot(s domain.Snapshot) {
	h.a.snap, h.a.ready = s, true
	h.a.render()
}

func TestConnectRendersEverything(t *testing.T) {
	h := newHarness()
	if len(h.surf.take()) != 0 {
		t.Fatal("rendered while disconnected")
	}
	h.a.handle(Connected{Connected: true})
	got := h.surf.take()
	want := []Command{
		SetLED{Key: gomcu.Select3, On: true},
		SetLED{Key: gomcu.Rec2, On: true},  // mute
		SetLED{Key: gomcu.Rec3, On: false}, // dim
		SetLED{Key: gomcu.F1, On: true},
		SetLED{Key: gomcu.F2, On: false},
		SetLED{Key: gomcu.F3, On: false},
		SetLED{Key: gomcu.F4, On: false},
		SetLED{Key: gomcu.F5, On: true},
		SetFader{Channel: gomcu.Channel3, Position: 0},
	}
	if !slices.Equal(got, want) {
		t.Errorf("rendered\n%v\nwant\n%v", got, want)
	}
}

func TestOnlyChangesAreRendered(t *testing.T) {
	h := newHarness()
	h.a.handle(Connected{Connected: true})
	h.surf.take()

	s := h.a.snap
	s.State = domain.Apply(s, domain.SelectSpeaker{ID: domain.SpeakerB, On: true}).State
	h.snapshot(s)
	want := []Command{SetLED{Key: gomcu.F1, On: false}, SetLED{Key: gomcu.F2, On: true}}
	if got := h.surf.take(); !slices.Equal(got, want) {
		t.Errorf("rendered %v, want %v", got, want)
	}
}

func TestKeysBecomeCommands(t *testing.T) {
	h := newHarness()
	for _, k := range []gomcu.Switch{gomcu.Rec2, gomcu.Rec3, gomcu.F3, gomcu.Play} {
		h.a.handle(KeyPressed{Key: k})
	}
	want := []domain.Command{domain.ToggleMute{}, domain.ToggleDim{}, domain.ToggleSpeaker{ID: domain.SpeakerC}}
	if !slices.Equal(h.cmds, want) {
		t.Errorf("commands = %#v, want %#v", h.cmds, want)
	}
}

func TestFaderDoesNotFightTheUser(t *testing.T) {
	h := newHarness()
	h.a.handle(Connected{Connected: true})
	h.surf.take()

	pos := dbToFader(-20, false)
	h.a.handle(FaderMoved{Channel: gomcu.Channel3, Position: pos})
	h.a.handle(FaderMoved{Channel: gomcu.Channel1, Position: 0}) // other strip
	if want := []domain.Command{domain.SetVolume{DB: -20}}; !slices.Equal(h.cmds, want) {
		t.Fatalf("commands = %#v", h.cmds)
	}

	s := h.a.snap
	s.State.Master.VolumeDB = -20 // the controller echoes the fader's volume
	h.snapshot(s)
	if got := h.surf.take(); len(got) != 0 {
		t.Errorf("echoed fader volume was sent back: %v", got)
	}

	s.State.Master.VolumeDB = -30 // changed elsewhere, e.g. in the app window
	h.snapshot(s)
	if got := h.surf.take(); !slices.Equal(got, []Command{SetFader{Channel: gomcu.Channel3, Position: dbToFader(-30, false)}}) {
		t.Errorf("rendered %v", got)
	}

	h.a.handle(KeyPressed{Key: gomcu.Select3})
	if got := h.surf.take(); len(got) != 1 {
		t.Errorf("select key should resend the fader, got %v", got)
	}
}

func TestMeter(t *testing.T) {
	h := newHarness()
	h.a.handle(Connected{Connected: true})
	h.surf.take()

	h.a.sendMeter()
	if got := h.surf.take(); len(got) != 0 {
		t.Errorf("silence sent %v", got)
	}
	s := h.a.snap
	s.Levels = domain.Levels{LeftDB: -25, RightDB: -9}
	h.snapshot(s)
	h.a.sendMeter()
	if got := h.surf.take(); !slices.Equal(got, []Command{SetMeter{Channel: gomcu.Channel3, Level: gomcu.MoreThan10}}) {
		t.Errorf("meter sent %v", got)
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		msg  midi.Message
		want Event
	}{
		{midi.NoteOn(0, uint8(gomcu.Mute1), 127), KeyPressed{Key: gomcu.Mute1}},
		{midi.NoteOn(0, uint8(gomcu.Mute1), 0), nil},
		{midi.Pitchbend(2, 8191), FaderMoved{Channel: gomcu.Channel3, Position: 16383}},
		{midi.ControlChange(0, 16, 1), nil},
	}
	for _, tt := range tests {
		got, ok := decode(tt.msg)
		if ok != (tt.want != nil) || got != tt.want {
			t.Errorf("decode(%v) = %#v, %t; want %#v", tt.msg, got, ok, tt.want)
		}
	}
}

func TestDefaultConfigMatchesStreamDeckProfile(t *testing.T) {
	c := DefaultConfig()
	if c.MuteKey != gomcu.Mute1 || c.DimKey != gomcu.Solo1 || c.SpeakerKeys[domain.Sub] != gomcu.Read {
		t.Errorf("default mapping changed: %+v", c)
	}
}
