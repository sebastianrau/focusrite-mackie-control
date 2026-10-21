package ui

import (
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

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

func TestRenderFollowsState(t *testing.T) {
	a, _ := newTestApp(t)

	s := domain.Snapshot{State: domain.DefaultState(), Levels: domain.Silence}
	s.State.Speakers[domain.SpeakerC].Name = "Nearfield"
	a.render(s)

	if got := a.speakers[domain.SpeakerC].Text; got != "Nearfield" {
		t.Errorf("speaker C label = %q", got)
	}
	if a.speakers[domain.SpeakerD].Visible() {
		t.Error("disabled speaker D is visible")
	}
	if !a.speakers[domain.SpeakerA].Disabled() || !a.fader.slider.Disabled() {
		t.Error("controls must be locked without an interface")
	}
	if a.status.Text != "Focusrite Control not running" {
		t.Errorf("status = %q", a.status.Text)
	}

	s.Device = domain.Device{ServerConnected: true, Present: true, Model: "Scarlett 18i20 (3rd Gen)", SampleRate: "48000"}
	s.State.Master.Mute = false
	s.State.Master.VolumeDB = -30
	a.render(s)

	if a.speakers[domain.SpeakerA].Disabled() {
		t.Error("controls still locked with an interface")
	}
	if a.speakers[domain.SpeakerA].Importance != widget.SuccessImportance || a.speakers[domain.SpeakerB].Importance != widget.MediumImportance {
		t.Error("selected speaker not highlighted")
	}
	if a.mute.Importance != widget.MediumImportance {
		t.Error("mute shown as on")
	}
	if a.fader.slider.Value != -30 {
		t.Errorf("fader = %v", a.fader.slider.Value)
	}
	if a.status.Text != "Scarlett 18i20 (3rd Gen) · 48000" {
		t.Errorf("status = %q", a.status.Text)
	}
}

func TestInputBecomesCommands(t *testing.T) {
	a, cmds := newTestApp(t)
	a.render(domain.Snapshot{State: domain.DefaultState(), Device: domain.Device{ServerConnected: true, Present: true}})

	test.Tap(a.speakers[domain.SpeakerB])
	test.Tap(a.mute)
	test.Tap(a.dim)
	a.fader.slider.SetValue(-12)

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
	if called {
		t.Error("SetDB reported a user change")
	}
	if f.label.Text != "-20 dB" {
		t.Errorf("label = %q", f.label.Text)
	}
}

func TestMeterDecay(t *testing.T) {
	test.NewTempApp(t)
	m := NewMeter()
	m.SetLevels(-6, -12)
	m.SetLevels(-30, -30) // lower levels decay instead of jumping
	if m.left != -6 || m.right != -12 || m.peak != -6 {
		t.Fatalf("levels = %v/%v peak %v", m.left, m.right, m.peak)
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
