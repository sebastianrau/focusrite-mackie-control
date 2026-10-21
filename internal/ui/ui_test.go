package ui

import (
	"image/color"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/sebastianrau/focusrite-mackie-control/internal/config"
	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

func newTestApp(t *testing.T) (*App, *[]domain.Command) {
	t.Helper()
	store, err := config.Open(filepath.Join(t.TempDir(), "settings.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cmds []domain.Command
	a := newApp(test.NewTempApp(t), Options{
		Submit: func(c domain.Command) { cmds = append(cmds, c) },
		Store:  store,
	})
	return a, &cmds
}

func connected() domain.Device {
	return domain.Device{ServerConnected: true, Present: true, Model: "Scarlett 18i20 (3rd Gen)", SampleRate: "96 kHz"}
}

func TestRenderWithoutInterface(t *testing.T) {
	a, _ := newTestApp(t)
	a.render(domain.Snapshot{State: domain.DefaultState(), Levels: domain.Silence})

	if a.speakers[domain.SpeakerA].enabled || a.mute.enabled || !a.fader.Disabled() {
		t.Error("controls must be locked without an interface")
	}
	if a.status.Text != infoNoServer || a.dot.FillColor != colTextDisabled {
		t.Errorf("status = %q", a.status.Text)
	}
	a.render(domain.Snapshot{State: domain.DefaultState(), Device: domain.Device{ServerConnected: true}})
	if a.status.Text != infoNoDevice || a.dot.FillColor != colYellow {
		t.Errorf("status = %q", a.status.Text)
	}
}

func TestRenderFollowsState(t *testing.T) {
	a, _ := newTestApp(t)
	s := domain.Snapshot{State: domain.DefaultState(), Device: connected()}
	s.State.Speakers[domain.SpeakerC].Name = "Nearfield"
	s.State.Master.Mute = false
	s.State.Master.VolumeDB = -30
	a.render(s)

	if got := a.speakers[domain.SpeakerC].Text(); got != "Nearfield" {
		t.Errorf("speaker C = %q", got)
	}
	if got := a.speakers[domain.SpeakerD].Text(); got != "Speaker D · disabled" || a.speakers[domain.SpeakerD].enabled {
		t.Errorf("disabled speaker D = %q, enabled %t", got, a.speakers[domain.SpeakerD].enabled)
	}
	selected, idle := a.speakers[domain.SpeakerA], a.speakers[domain.SpeakerB]
	if selected.bg.StrokeColor != colGreen || !selected.label.TextStyle.Bold || selected.bg.FillColor != colGreenBg {
		t.Error("selected speaker must have the green outline, bold text and tinted fill")
	}
	if idle.bg.StrokeWidth != 0 || idle.label.TextStyle.Bold {
		t.Error("idle speaker must be plain")
	}
	if a.volume.Text != "-30" || a.volNote.Text != "" || a.fader.Value != -30 {
		t.Errorf("volume = %q note %q fader %v", a.volume.Text, a.volNote.Text, a.fader.Value)
	}
	if a.mute.bg.FillColor != colRaised {
		t.Error("mute shown as on")
	}
	if a.status.Text != "Scarlett 18i20 (3rd Gen)" || a.dot.FillColor != colGreen {
		t.Errorf("status = %q", a.status.Text)
	}

	s.State.Master.Dim = true
	a.render(s)
	if a.dim.bg.FillColor != colYellow || a.volNote.Text != "dimmed −20 dB" {
		t.Errorf("dim: fill %v note %q", a.dim.bg.FillColor, a.volNote.Text)
	}
	s.State.Master.Mute = true
	a.render(s)
	if a.mute.bg.FillColor != colRed || a.volNote.Text != "muted" {
		t.Errorf("mute: fill %v note %q", a.mute.bg.FillColor, a.volNote.Text)
	}
}

func TestInputBecomesCommands(t *testing.T) {
	a, cmds := newTestApp(t)
	a.render(domain.Snapshot{State: domain.DefaultState(), Device: connected()})

	test.Tap(a.speakers[domain.SpeakerB])
	test.Tap(a.speakers[domain.SpeakerD]) // disabled: ignored
	test.Tap(a.mute)
	test.Tap(a.dim)
	a.fader.SetValue(-12)

	want := []domain.Command{
		domain.ToggleSpeaker{ID: domain.SpeakerB},
		domain.ToggleMute{},
		domain.ToggleDim{},
		domain.SetVolume{DB: -12},
	}
	if len(*cmds) != len(want) {
		t.Fatalf("commands = %#v", *cmds)
	}
	for i := range want {
		if (*cmds)[i] != want[i] {
			t.Errorf("command %d = %#v, want %#v", i, (*cmds)[i], want[i])
		}
	}
}

func TestFaderSetDBDoesNotReport(t *testing.T) {
	test.NewTempApp(t)
	called := false
	f := NewFader(func(int) { called = true })
	f.SetDB(-20)
	if called || f.Value != -20 {
		t.Errorf("SetDB reported a change or did not move: %v", f.Value)
	}
}

func TestMeterDecay(t *testing.T) {
	test.NewTempApp(t)
	m := NewMeter()
	m.SetLevels(-6, -12)
	m.SetLevels(-30, -30) // lower levels decay instead of jumping
	if m.left != -6 || m.right != -12 {
		t.Fatalf("levels = %v/%v", m.left, m.right)
	}
	m.lastTick = time.Now().Add(-time.Second)
	m.tick()
	if m.left >= -6 || m.left < -6-meterDecayDBPerSecond-0.5 {
		t.Errorf("left after 1s = %v", m.left)
	}
	m.SetLevels(-200, 50)
	if m.right != meterMaxDB {
		t.Errorf("right = %v, want clamped to %v", m.right, meterMaxDB)
	}
}

// Segments light up to the level in the zone colours of the design.
func TestMeterSegments(t *testing.T) {
	test.NewTempApp(t)
	m := NewMeter()
	r := m.CreateRenderer().(*meterRenderer)
	m.left, m.right = -3, meterMinDB
	r.Refresh()

	lit := func(ch int) (n int, top color.Color) {
		for i := range meterSegments {
			if c := r.segs[ch*meterSegments+i].FillColor; c != colRaised {
				n, top = i+1, c
			}
		}
		return n, top
	}
	if n, top := lit(0); n != 29 || top != colOrange {
		t.Errorf("left: %d segments lit, top %v; want 29 up to orange", n, top)
	}
	if n, _ := lit(1); n != 0 {
		t.Errorf("right: %d segments lit at silence", n)
	}
	if zoneColor(-20) != colGreen || zoneColor(-10) != colYellow || zoneColor(-3) != colOrange {
		t.Error("zone colours changed")
	}

	m.SetDisabled(true)
	r.Refresh()
	if n, _ := lit(0); n != 0 {
		t.Error("a disabled meter must be dark")
	}
}

func TestTrayIconFollowsState(t *testing.T) {
	if trayIconFor(false, false) != trayNormal || trayIconFor(false, true) != trayDim || trayIconFor(true, true) != trayMute {
		t.Error("tray icon must show normal, dim, and mute (which wins over dim)")
	}
}
