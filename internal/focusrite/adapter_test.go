package focusrite

import (
	"os"
	"slices"
	"testing"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite/protocol"
)

// legacy18i20 is the hard-coded default of earlier versions, which was made
// for this interface.
var legacy18i20 = Config{
	SerialNumber: "P9EAC6K250F325",
	MuteID:       1679,
	DimID:        1678,
	Outputs: [domain.NumSpeakers]Output{
		{NameID: 1456, MuteID: 1453, GainID: 1458, MeterLeftID: 1450, MeterRightID: 1460},
		{NameID: 1476, MuteID: 1473, GainID: 1478, MeterLeftID: 1470, MeterRightID: 1480},
		{NameID: 1496, MuteID: 1493, GainID: 1498, MeterLeftID: 1490, MeterRightID: 1500},
		{NameID: 1516, MuteID: 1513, GainID: 1518, MeterLeftID: 1510, MeterRightID: 1520},
		{NameID: 1536, MuteID: 1533, GainID: 1538, MeterLeftID: 1530, MeterRightID: 1540},
	},
}

func loadDevice(t *testing.T, name string) *protocol.Device {
	t.Helper()
	b, err := os.ReadFile("protocol/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	m, err := protocol.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	d := m.(protocol.DeviceArrival).Device
	return &d
}

func TestAutoConfigMatchesLegacyDefaults(t *testing.T) {
	info := Describe(loadDevice(t, "device-arrival-18i20.xml"))
	if got := AutoConfig(info); got != legacy18i20 {
		t.Errorf("AutoConfig =\n%+v\nwant\n%+v", got, legacy18i20)
	}
	if info.Outputs[0].Name != "Monitor Outputs 1-2" {
		t.Errorf("first output name = %q", info.Outputs[0].Name)
	}
}

func TestDescribeSmallInterface(t *testing.T) {
	info := Describe(loadDevice(t, "device-arrival-4i4.xml"))
	if len(info.Outputs) != 2 {
		t.Fatalf("4i4 outputs = %d (%+v), want 2", len(info.Outputs), info.Outputs)
	}
	cfg := AutoConfig(info)
	if cfg.Outputs[domain.SpeakerC] != (Output{}) {
		t.Error("speakers beyond the available outputs must stay unassigned")
	}
}

func TestDesiredItems(t *testing.T) {
	s := domain.DefaultState()
	s.Master.Mute = false
	s.Master.VolumeDB = -10
	s.Master.Dim = true
	s.Master.DimDB = 15

	items := DesiredItems(legacy18i20, s)
	want := map[int]string{
		1679: "false", 1678: "true",
		1456: "Speaker A", 1453: "false", 1458: "-25", // selected
		1476: "Speaker B", 1473: "true", 1478: "-25", // not selected
		1516: "Speaker D", // disabled: name only
		1536: "Sub", 1533: "false", 1538: "-25",
	}
	got := map[int]string{}
	for _, it := range items {
		got[it.ID] = it.Value
	}
	for id, v := range want {
		if got[id] != v {
			t.Errorf("item %d = %q, want %q", id, got[id], v)
		}
	}
	if _, ok := got[1513]; ok {
		t.Error("disabled speaker D must not be muted or unmuted")
	}
}

type fakeSender struct{ sets []protocol.Set }

func (f *fakeSender) Send(s protocol.Set) { f.sets = append(f.sets, s) }

func (f *fakeSender) take() []protocol.Item {
	var items []protocol.Item
	for _, s := range f.sets {
		items = append(items, s.Items...)
	}
	f.sets = nil
	return items
}

type harness struct {
	a    *Adapter
	out  *fakeSender
	cmds []domain.Command
}

func newHarness(t *testing.T, cfg Config) *harness {
	t.Helper()
	h := &harness{out: &fakeSender{}}
	h.a = NewAdapter(cfg, nil, h.out, func(c domain.Command) { h.cmds = append(h.cmds, c) }, nil)
	h.state(domain.Snapshot{State: domain.DefaultState(), Levels: domain.Silence})
	h.a.handle(ServerConnected{Connected: true})
	return h
}

// state applies a snapshot as the controller would publish it.
func (h *harness) state(s domain.Snapshot) {
	h.a.snap, h.a.ready = s, true
	h.a.push()
}

func (h *harness) takeCmds() []domain.Command {
	c := h.cmds
	h.cmds = nil
	return c
}

// arrive announces the 18i20 and delivers its initial value dump.
func (h *harness) arrive(t *testing.T, dump ...protocol.Item) {
	t.Helper()
	h.a.handle(DeviceArrived{Device: loadDevice(t, "device-arrival-18i20.xml")})
	if len(h.out.sets) != 0 {
		t.Fatal("sent state before the device reported its own")
	}
	h.a.handle(DeviceUpdated{Set: protocol.Set{DevID: 3, Items: dump}})
}

func TestArrivalPushesFullStateOnce(t *testing.T) {
	h := newHarness(t, legacy18i20)
	h.takeCmds()
	h.arrive(t)

	cmds := h.takeCmds()
	if len(cmds) == 0 {
		t.Fatal("no commands")
	}
	dev := cmds[0].(domain.SetDevice).Device
	if !dev.Present || !dev.ServerConnected || dev.Model != "Scarlett 18i20 (3rd Gen)" {
		t.Errorf("device = %+v", dev)
	}
	if got, want := len(h.out.take()), len(DesiredItems(legacy18i20, h.a.snap.State)); got != want {
		t.Errorf("pushed %d items, want %d", got, want)
	}

	h.state(h.a.snap) // unchanged
	if items := h.out.take(); len(items) != 0 {
		t.Errorf("unchanged state sent %v", items)
	}

	next := h.a.snap
	next.State = domain.Apply(next, domain.ToggleMute{}).State
	h.state(next)
	got := h.out.take()
	want := []protocol.Item{protocol.BoolItem(1679, false), protocol.BoolItem(1453, false), protocol.BoolItem(1533, false)}
	if !slices.Equal(got, want) {
		t.Errorf("unmute sent %v, want %v", got, want)
	}
}

// At startup the device's own values for its controls and names win; the
// app's speaker selection and volume are pushed. Nothing may flip back.
func TestStartupReconciliation(t *testing.T) {
	h := newHarness(t, legacy18i20) // default state: muted
	h.takeCmds()
	h.arrive(t,
		protocol.BoolItem(1679, false),    // hardware mute is off
		protocol.StringItem(1456, "Main"), // nickname set in Focusrite Control
		protocol.IntItem(1458, -80),       // output gain: the app's volume wins
	)

	cmds := h.takeCmds()
	if !slices.Contains(cmds, domain.Command(domain.SetMute{On: false})) ||
		!slices.Contains(cmds, domain.Command(domain.RenameSpeaker{ID: domain.SpeakerA, Name: "Main"})) {
		t.Errorf("commands = %#v", cmds)
	}
	sent := map[int]string{}
	for _, it := range h.out.take() {
		sent[it.ID] = it.Value
	}
	if _, ok := sent[1679]; ok {
		t.Errorf("overwrote the hardware mute with %q", sent[1679])
	}
	if _, ok := sent[1456]; ok {
		t.Errorf("overwrote the nickname with %q", sent[1456])
	}
	if sent[1458] != "-127" || sent[1453] != "false" {
		t.Errorf("speaker A gain/mute = %q/%q, want -127/false (unmuted now)", sent[1458], sent[1453])
	}
}

func TestEchoesAreNotCommands(t *testing.T) {
	h := newHarness(t, legacy18i20)
	h.arrive(t)
	h.out.take()
	h.takeCmds()

	// The user toggles mute twice quickly; the device echoes both late.
	s := h.a.snap
	s.State.Master.Mute = false
	h.state(s)
	s.State.Master.Mute = true
	h.state(s)
	h.a.handle(DeviceUpdated{Set: protocol.Set{DevID: 3, Items: []protocol.Item{protocol.BoolItem(1679, false)}}})
	h.a.handle(DeviceUpdated{Set: protocol.Set{DevID: 3, Items: []protocol.Item{protocol.BoolItem(1679, true)}}})

	for _, c := range h.takeCmds() {
		if _, ok := c.(domain.SetMute); ok {
			t.Errorf("echo turned into %#v", c)
		}
	}
}

func TestDeviceChangesBecomeCommands(t *testing.T) {
	h := newHarness(t, legacy18i20)
	h.arrive(t)
	h.takeCmds()

	h.a.handle(DeviceUpdated{Set: protocol.Set{DevID: 3, Items: []protocol.Item{
		protocol.BoolItem(1678, true),          // hardware dim pressed
		protocol.StringItem(1476, "Nearfield"), // renamed in Focusrite Control
		protocol.StringItem(1450, "-12.5"),     // speaker A meter left
		protocol.StringItem(1460, "-14"),       // speaker A meter right
		protocol.StringItem(1470, "-3"),        // speaker B meter: not the meter source
	}}})

	cmds := h.takeCmds()
	want := []domain.Command{
		domain.SetDim{On: true},
		domain.RenameSpeaker{ID: domain.SpeakerB, Name: "Nearfield"},
		domain.SetLevels{Levels: domain.Levels{LeftDB: -12.5, RightDB: -14}},
	}
	if !slices.Contains(cmds, want[0]) || !slices.Contains(cmds, want[1]) || !slices.Contains(cmds, want[2]) {
		t.Errorf("commands = %#v\nwant to contain %#v", cmds, want)
	}
	levels := 0
	for _, c := range cmds {
		if _, ok := c.(domain.SetLevels); ok {
			levels++
		}
	}
	if levels != 1 {
		t.Errorf("got %d SetLevels for one set, want 1", levels)
	}
}

func TestOtherDevicesAreIgnored(t *testing.T) {
	cfg := legacy18i20
	cfg.SerialNumber = "SOMETHING-ELSE"
	h := newHarness(t, cfg)
	h.arrive(t)
	if items := h.out.take(); len(items) != 0 {
		t.Errorf("sent %d items to an unconfigured device", len(items))
	}
	select {
	case list := <-h.a.Devices():
		if len(list) != 1 || list[0].SerialNumber != "P9EAC6K250F325" {
			t.Errorf("catalog = %+v", list)
		}
	default:
		t.Error("catalog not published")
	}
}

func TestAdoptsFirstDeviceWithoutConfig(t *testing.T) {
	h := newHarness(t, Config{})
	var adopted Config
	h.a.OnAdopt = func(c Config) { adopted = c }
	h.arrive(t)
	if adopted != legacy18i20 {
		t.Errorf("adopted %+v", adopted)
	}
	if len(h.out.take()) == 0 {
		t.Error("adopted device received no state")
	}
}

func TestRemovalAndDisconnect(t *testing.T) {
	h := newHarness(t, legacy18i20)
	h.arrive(t)
	h.takeCmds()
	h.out.take()

	h.a.handle(DeviceRemoved{ID: 3})
	if dev := h.takeCmds()[0].(domain.SetDevice).Device; dev.Present || !dev.ServerConnected {
		t.Errorf("after removal: %+v", dev)
	}
	h.state(domain.Snapshot{State: domain.Apply(h.a.snap, domain.ToggleMute{}).State})
	if items := h.out.take(); len(items) != 0 {
		t.Errorf("sent %v without a device", items)
	}

	h.arrive(t)
	h.a.handle(ServerConnected{Connected: false})
	cmds := h.takeCmds()
	if dev := cmds[len(cmds)-1].(domain.SetDevice).Device; dev.Present || dev.ServerConnected {
		t.Errorf("after disconnect: %+v", dev)
	}
}
