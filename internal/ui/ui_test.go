package ui

import (
	"image/color"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
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

	if a.speakers[domain.SpeakerA].enabled || a.mute.enabled || a.scenes[domain.SceneA].enabled || !a.fader.Disabled() {
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
	if a.volume.Text != "-30" || a.fader.Value != -30 {
		t.Errorf("volume = %q fader %v", a.volume.Text, a.fader.Value)
	}
	if a.mute.label.Color != colTextDim || a.dim.bg.StrokeColor != colTextDim {
		t.Error("Mute and Dim must be grey when off")
	}
	if a.status.Text != "Scarlett 18i20 (3rd Gen)" || a.dot.FillColor != colGreen {
		t.Errorf("status = %q", a.status.Text)
	}

	s.State.Master.Dim = true
	a.render(s)
	if a.dim.bg.StrokeColor != colYellow || a.dim.label.Color != colYellow {
		t.Errorf("dim: outline %v", a.dim.bg.StrokeColor)
	}
	s.State.Master.Mute = true
	a.render(s)
	if a.mute.bg.StrokeColor != colRedText || a.mute.label.Color != colRedText {
		t.Errorf("mute: outline %v", a.mute.bg.StrokeColor)
	}
}

func TestInputBecomesCommands(t *testing.T) {
	a, cmds := newTestApp(t)
	a.render(domain.Snapshot{State: domain.DefaultState(), Device: connected()})

	test.Tap(a.speakers[domain.SpeakerB])
	test.Tap(a.speakers[domain.SpeakerD]) // disabled: ignored
	test.Tap(a.mute)
	test.Tap(a.dim)
	test.Tap(a.scenes[domain.SceneB])
	a.fader.SetValue(-12)

	want := []domain.Command{
		domain.ToggleSpeaker{ID: domain.SpeakerB},
		domain.ToggleMute{},
		domain.ToggleDim{},
		domain.SelectScene{ID: domain.SceneB},
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
	m.left, m.right = -3, m.floor
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

// A lower floor spreads the 30 segments over a wider range, and the state
// sets it on meter and scale.
func TestMeterFloor(t *testing.T) {
	a, _ := newTestApp(t)
	s := domain.Snapshot{State: domain.DefaultState(), Device: connected()}
	s.State.Master.MeterFloorDB = -96
	s.Levels = domain.Levels{LeftDB: -93, RightDB: -127}
	a.render(s)

	m := a.meter
	if m.floor != -96 || a.scale.floor != -96 {
		t.Fatalf("floor meter %v, scale %v; want -96", m.floor, a.scale.floor)
	}
	if n := m.litSegments(m.left); n != 1 {
		t.Errorf("-93 dB lights %d segments at -96 dB range, want 1", n)
	}
	if n := m.litSegments(m.right); n != 0 {
		t.Errorf("silence lights %d segments", n)
	}
	s.State.Master.MeterFloorDB = -60
	a.render(s)
	if m.left != -60 {
		t.Errorf("level %v not clamped to the new floor", m.left)
	}
}

// The scale labels of every range start at 0, end at the floor and do not
// overlap in the smallest window.
func TestMeterScales(t *testing.T) {
	a, _ := newTestApp(t)
	a.render(domain.Snapshot{State: domain.DefaultState(), Device: connected()})
	a.window.Resize(a.window.Content().MinSize())
	height := a.scale.Size().Height
	for _, floor := range domain.MeterFloors {
		sc := newMeterScale()
		sc.SetFloor(float64(floor))
		r := test.TempWidgetRenderer(t, sc).(*scaleRenderer)
		r.Layout(fyne.NewSize(18, height))
		labels := meterScale(float64(floor))
		if labels[0] != 0 || labels[len(labels)-1] != float64(floor) {
			t.Errorf("%d dB: scale %v", floor, labels)
		}
		if len(r.labels) != len(labels) {
			t.Fatalf("%d dB: %d labels for %d values", floor, len(r.labels), len(labels))
		}
		for i := 1; i < len(r.labels); i++ {
			prev, cur := r.labels[i-1], r.labels[i]
			if prev.Position().Y+prev.MinSize().Height > cur.Position().Y {
				t.Errorf("%d dB: labels %s and %s overlap at %.0f px", floor, prev.Text, cur.Text, height)
			}
		}
	}
}

func TestTrayIconFollowsState(t *testing.T) {
	if trayIconFor(false, false) != trayNormal || trayIconFor(false, true) != trayDim || trayIconFor(true, true) != trayMute {
		t.Error("tray icon must show normal, dim, and mute (which wins over dim)")
	}
}

// The meter only redraws when a segment changes, and not at all while paused.
func TestMeterRedrawsOnlyOnChange(t *testing.T) {
	test.NewTempApp(t)
	m := NewMeter()
	test.WidgetRenderer(m) // registers the renderer, as showing it would

	m.SetLevels(-10, -10)
	if m.redraws != 1 {
		t.Fatalf("redraws = %d after the first level", m.redraws)
	}
	m.left, m.right = -10.5, -10.5 // within the same segment (-11 dB)
	m.redrawIfChanged()
	if m.redraws != 1 {
		t.Errorf("redrew without a visible change")
	}
	m.left, m.right = -12, -12
	m.redrawIfChanged()
	if m.redraws != 2 {
		t.Errorf("did not redraw when a segment went out")
	}

	m.Pause()
	m.SetLevels(-1, -1)
	if m.redraws != 2 {
		t.Error("paused meter redrew")
	}
	m.Resume()
	m.SetLevels(-1, -1)
	if m.litL != m.litSegments(-1) {
		t.Errorf("resumed meter shows %d segments, want %d", m.litL, m.litSegments(-1))
	}

	if m.litSegments(0) != meterSegments || m.litSegments(m.floor) != 0 || m.litSegments(-31) != 15 {
		t.Errorf("litSegments: 0 dB %d, -60 dB %d, -31 dB %d", m.litSegments(0), m.litSegments(m.floor), m.litSegments(-31))
	}
}

func TestLongNamesAreShortened(t *testing.T) {
	test.NewTempApp(t)
	b := newBlockButton("A very long speaker name · disabled", fyne.TextAlignLeading, nil).withKey("D")
	r := test.WidgetRenderer(b)
	b.Resize(fyne.NewSize(160, 44))
	r.Layout(b.Size())
	if got := b.label.Text; !strings.HasSuffix(got, "…") || b.label.MinSize().Width > 160-40 {
		t.Errorf("label = %q (%.0f px wide)", got, b.label.MinSize().Width)
	}
	if b.Text() != "A very long speaker name · disabled" {
		t.Error("Text must return the full name")
	}
	b.Resize(fyne.NewSize(600, 44))
	r.Layout(b.Size())
	if b.label.Text != b.Text() {
		t.Errorf("label shortened although it fits: %q", b.label.Text)
	}
}

func TestPrePostSwitch(t *testing.T) {
	a, cmds := newTestApp(t)
	s := domain.Snapshot{State: domain.DefaultState(), Device: connected()}
	s.State.Master.Mute, s.State.Master.VolumeDB = false, -30
	s.Levels = domain.Levels{LeftDB: -6, RightDB: -6}
	a.render(s)
	if a.prePost.Text() != "PRE" || a.meter.left != -6 {
		t.Errorf("pre: switch %q, meter %v", a.prePost.Text(), a.meter.left)
	}

	test.Tap(a.prePost)
	if len(*cmds) != 1 || (*cmds)[0] != (domain.ToggleMeterPost{}) {
		t.Fatalf("commands = %#v", *cmds)
	}

	a.meter.left, a.meter.right = a.meter.floor, a.meter.floor
	s.State.Master.MeterPost = true
	a.render(s)
	if a.prePost.Text() != "POST" || a.meter.left != -36 {
		t.Errorf("post: switch %q, meter %v; want POST and -36", a.prePost.Text(), a.meter.left)
	}
	if a.prePost.bg.StrokeColor != colGreen || a.prePost.label.Color != colGreen {
		t.Error("the switch must have a green outline and green text")
	}
}

// The tags must show their full label, never "P…".
func TestTagLabelsFit(t *testing.T) {
	test.NewTempApp(t)
	b := newTagButton("PRE", nil)
	r := test.WidgetRenderer(b)
	for _, text := range []string{"PRE", "POST"} {
		b.setText(text)
		b.Resize(b.MinSize())
		r.Layout(b.Size())
		if b.label.Text != text {
			t.Errorf("label shows %q, want %q", b.label.Text, text)
		}
	}
	for _, text := range []string{"MUTE", "DIM"} {
		b := newTagButton(text, nil)
		test.WidgetRenderer(b).Layout(b.MinSize())
		if b.label.Text != text {
			t.Errorf("label shows %q, want %q", b.label.Text, text)
		}
	}
}

// Mute, Dim and PRE/POST are the same size and share one row in the strip.
func TestTagRowFitsStrip(t *testing.T) {
	test.NewTempApp(t)
	tag := newTagButton("PRE", nil).MinSize()
	if newTagButton("MUTE", nil).MinSize() != tag || newTagButton("DIM", nil).MinSize() != tag {
		t.Error("all tags must be the same size")
	}
	if w := 3*tag.Width + 2*3; w > 128-20 {
		t.Errorf("tag row is %.0f px, the strip has 108", w)
	}
}

// A scene button shows the volume, Mute and Dim, and the speakers at fixed
// places: green when on, hidden when disabled.
func TestSceneButtons(t *testing.T) {
	a, _ := newTestApp(t)
	s := domain.Snapshot{State: domain.DefaultState(), Device: connected()}
	s.State.Scenes[domain.SceneB] = domain.Scene{Dim: true, VolumeDB: -12,
		Selected: [domain.NumSpeakers]bool{domain.SpeakerC: true}}
	s.State.Master.Mute, s.State.Master.VolumeDB = false, -24
	s.State.Scenes[domain.SceneA] = s.State.CurrentScene()
	a.render(s)

	sa, sb := a.scenes[domain.SceneA], a.scenes[domain.SceneB]
	if sa.bg.StrokeColor != colGreen || sb.bg.StrokeColor != nil {
		t.Error("only the active scene must be outlined")
	}
	if sa.volume.Text != "-24" || sb.volume.Text != "-12" {
		t.Errorf("volumes %q %q", sa.volume.Text, sb.volume.Text)
	}
	if sa.mute.Visible() || sa.dim.Visible() || sb.mute.Visible() || !sb.dim.Visible() {
		t.Error("M and D must show Mute and Dim of their scene")
	}
	for i, l := range sb.letters {
		on := domain.SpeakerID(i) == domain.SpeakerC
		if (l.Color == colGreen) != on {
			t.Errorf("scene B letter %s green %t", l.Text, l.Color == colGreen)
		}
	}
	if sa.letters[domain.SpeakerA].Color != colGreen || sa.letters[domain.Sub].Color != colGreen {
		t.Error("scene A must show A and S on")
	}
	if sa.letters[domain.SpeakerD].Visible() || !sa.letters[domain.SpeakerC].Visible() {
		t.Error("only the disabled speaker D must be hidden")
	}
	r := test.WidgetRenderer(sa)
	r.Layout(fyne.NewSize(100, 44))
	x := sa.letters[domain.Sub].Position().X
	s.State.Scenes[domain.SceneA].VolumeDB = -127
	s.State.Master.VolumeDB = -127
	a.render(s)
	r.Layout(fyne.NewSize(100, 44))
	if got := sa.letters[domain.Sub].Position().X; got != x {
		t.Errorf("speaker letters moved with the volume: %v -> %v", x, got)
	}
}

// Holding the inactive scene's button stores the live state instead of
// switching, and its release is not a tap.
func TestSceneLongPressStores(t *testing.T) {
	a, cmds := newTestApp(t)
	a.render(domain.Snapshot{State: domain.DefaultState(), Device: connected()})
	b := a.scenes[domain.SceneB]
	b.Resize(fyne.NewSize(100, 44))

	// The test driver runs animations to their end at once: the press is a
	// complete hold.
	b.MouseDown(&desktop.MouseEvent{})
	b.MouseUp(&desktop.MouseEvent{})
	test.Tap(b)
	if len(*cmds) != 1 || (*cmds)[0] != (domain.StoreScene{ID: domain.SceneB}) {
		t.Fatalf("commands = %#v", *cmds)
	}
	if b.bar.Visible() {
		t.Error("progress bar left visible")
	}

	b.pressed = time.Now().Add(-2 * time.Second) // released before the hold ended
	test.Tap(b)
	if len(*cmds) != 1 {
		t.Errorf("an aborted hold switched: %#v", *cmds)
	}

	test.Tap(b) // a short press is a tap
	if len(*cmds) != 2 || (*cmds)[1] != (domain.SelectScene{ID: domain.SceneB}) {
		t.Errorf("commands = %#v", *cmds)
	}

	b.showProgress(0.5)
	if !b.bar.Visible() || b.bar.Size().Width != 42 {
		t.Errorf("progress bar %v visible %t", b.bar.Size(), b.bar.Visible())
	}
}

// Holding the active scene's button stores nothing.
func TestActiveSceneHoldDoesNothing(t *testing.T) {
	a, cmds := newTestApp(t)
	a.render(domain.Snapshot{State: domain.DefaultState(), Device: connected()})
	b := a.scenes[domain.SceneA]
	b.MouseDown(&desktop.MouseEvent{})
	b.MouseUp(&desktop.MouseEvent{})
	if len(*cmds) != 0 || b.bar.Visible() {
		t.Errorf("commands = %#v, bar %t", *cmds, b.bar.Visible())
	}
}
