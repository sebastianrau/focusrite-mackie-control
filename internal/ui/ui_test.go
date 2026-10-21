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

func TestRenderFollowsState(t *testing.T) {
	a, _ := newTestApp(t)

	s := domain.Snapshot{State: domain.DefaultState(), Levels: domain.Silence}
	s.State.Speakers[domain.SpeakerC].Name = "Nearfield"
	a.render(s)

	if got := a.speakers[domain.SpeakerC].Text(); got != "Nearfield" {
		t.Errorf("speaker C label = %q", got)
	}
	if a.speakers[domain.SpeakerD].Visible() {
		t.Error("disabled speaker D is visible")
	}
	if !a.speakers[domain.SpeakerA].Disabled() || !a.fader.slider.Disabled() {
		t.Error("controls must be locked without an interface")
	}
	if a.status.Text != infoNoServer {
		t.Errorf("status = %q", a.status.Text)
	}

	s.Device = domain.Device{ServerConnected: true, Present: true, Model: "Scarlett 18i20 (3rd Gen)", SampleRate: "96 kHz"}
	s.State.Master.Mute = false
	s.State.Master.VolumeDB = -30
	a.render(s)

	if a.speakers[domain.SpeakerA].Disabled() {
		t.Error("controls still locked with an interface")
	}
	if !a.speakers[domain.SpeakerA].On() || a.speakers[domain.SpeakerB].On() || a.mute.On() {
		t.Error("button states do not follow the state")
	}
	if a.speakers[domain.SpeakerA].label.Color != green || a.speakers[domain.SpeakerB].label.Color == green {
		t.Error("selected speaker label must be green, others not")
	}
	if a.fader.slider.Value != -30 || a.fader.LabelText() != "-30.0 dB" {
		t.Errorf("fader = %v / %q", a.fader.slider.Value, a.fader.LabelText())
	}
	if a.status.Text != "Scarlett 18i20 (3rd Gen)" {
		t.Errorf("status = %q", a.status.Text)
	}
}

func TestInputBecomesCommands(t *testing.T) {
	a, cmds := newTestApp(t)
	a.render(domain.Snapshot{State: domain.DefaultState(), Device: domain.Device{ServerConnected: true, Present: true}})

	test.Tap(a.speakers[domain.SpeakerB].button)
	test.Tap(a.mute.button)
	test.Tap(a.dim.button)
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
}

func TestMeterDecay(t *testing.T) {
	test.NewTempApp(t)
	m := NewMeter()
	m.SetLevels(-6, -12)
	m.SetLevels(-30, -30) // lower levels decay instead of jumping
	if m.left != -6 || m.right != -12 || m.peak != -6 {
		t.Fatalf("levels = %v/%v peak %v", m.left, m.right, m.peak)
	}
	m.lastTick, m.peakAt = time.Now().Add(-time.Second), time.Now()
	m.tick()
	if m.left >= -6 || m.left < -6-meterDecayDBPerSecond-0.5 {
		t.Errorf("left after 1s = %v", m.left)
	}
	if m.peak != -6 {
		t.Errorf("peak released early: %v", m.peak)
	}
	m.SetLevels(-200, 50)
	if m.right != meterMaxDB {
		t.Errorf("right = %v, want clamped to %v", m.right, meterMaxDB)
	}
}

// The original colours: dark green at the bottom, green at -15 dB, yellow at
// -6 dB, orange at 0 dB, blended in between.
func TestLevelGradient(t *testing.T) {
	rgb := func(c color.Color) [3]uint32 {
		r, g, b, _ := c.RGBA()
		return [3]uint32{r >> 8, g >> 8, b >> 8}
	}
	for _, tc := range []struct {
		db   float64
		want color.Color
	}{{-127, darkGreen}, {-15, green}, {-6, yellow}, {0, orange}, {3, orange}} {
		if got := levelGradient.at(tc.db); rgb(got) != rgb(tc.want) && tc.db != -127 {
			t.Errorf("at(%v) = %v, want %v", tc.db, got, tc.want)
		}
	}
	mid := rgb(levelGradient.at(-10.5))
	if mid[0] <= 51 || mid[0] >= 255 || mid[1] != 255 {
		t.Errorf("at(-10.5) = %v, want a blend of green and yellow", mid)
	}
}
