package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebastianrau/gomcu"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite"
	"github.com/sebastianrau/focusrite-mackie-control/internal/mcu"
)

func tempPath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "sub", "monitor-controller.yaml")
}

func TestMissingFileGivesDefaults(t *testing.T) {
	path := tempPath(t)
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Get().State(); got != domain.DefaultState() {
		t.Errorf("state = %+v", got)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("defaults were not written: %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	path := tempPath(t)
	s, _ := Open(path)

	state := domain.DefaultState()
	state.Master.VolumeDB = -17
	state.Speakers[domain.SpeakerC] = domain.Speaker{Name: "Cans", Selected: true}
	fc := focusrite.Config{SerialNumber: "X1", MuteID: 7, Outputs: [domain.NumSpeakers]focusrite.Output{{NameID: 1, MuteID: 2, GainID: 3, MeterLeftID: 4, MeterRightID: 5}}}
	mc := mcu.DefaultConfig()
	mc.FaderChannel = gomcu.Master
	mc.LogarithmicFade = true

	s.Update(func(c *Config) {
		c.SetState(state)
		c.SetFocusrite(fc)
		c.SetMCU(mc)
	})
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c := s2.Get()
	if c.State() != state {
		t.Errorf("state = %+v", c.State())
	}
	if c.FocusriteConfig() != fc {
		t.Errorf("focusrite = %+v", c.FocusriteConfig())
	}
	if c.MCUConfig() != mc {
		t.Errorf("mcu = %+v", c.MCUConfig())
	}

	raw, _ := os.ReadFile(path)
	for _, want := range []string{"version: 2", "mute_key: Mute1", "fader_channel: Master", "- Trim"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("file lacks %q:\n%s", want, raw)
		}
	}
}

func TestMigratesLegacyFile(t *testing.T) {
	legacy, err := os.ReadFile("testdata/legacy-v1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := tempPath(t)
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, legacy, 0o600)

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c := s.Get()

	st := c.State()
	if st.Master.VolumeDB != -23 || !st.Master.Mute || st.Master.DimDB != 20 {
		t.Errorf("master = %+v", st.Master)
	}
	if st.Speakers[domain.SpeakerB].Name != "Nearfield" || !st.Speakers[domain.SpeakerD].Disabled || !st.Speakers[domain.Sub].Selected {
		t.Errorf("speakers = %+v", st.Speakers)
	}

	fc := c.FocusriteConfig()
	if fc.SerialNumber != "P9EAC6K250F325" || fc.MuteID != 1679 || fc.DimID != 1678 {
		t.Errorf("focusrite = %+v", fc)
	}
	if want := (focusrite.Output{NameID: 1536, MuteID: 1533, GainID: 1538, MeterLeftID: 1530, MeterRightID: 1540}); fc.Outputs[domain.Sub] != want {
		t.Errorf("sub output = %+v", fc.Outputs[domain.Sub])
	}

	mc := c.MCUConfig()
	want := mcu.DefaultConfig()
	want.InputPort, want.OutputPort = "IN", "OUT"
	if mc != want {
		t.Errorf("mcu = %+v\nwant  %+v", mc, want)
	}

	// The migrated file is written in the new format.
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(raw), "version: 2") {
		t.Errorf("not rewritten:\n%s", raw)
	}
}

func TestInvalidFileIsMovedAside(t *testing.T) {
	for name, content := range map[string]string{
		"syntax":        "controller: [",
		"unknown field": "version: 2\nbogus: 1\n",
		"unknown key":   "version: 2\nmidi:\n  mute_key: NotAKey\n",
		"future":        "version: 99\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := tempPath(t)
			os.MkdirAll(filepath.Dir(path), 0o755)
			os.WriteFile(path, []byte(content), 0o600)

			s, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if s.Get().State() != domain.DefaultState() {
				t.Error("expected defaults")
			}
			if b, err := os.ReadFile(path + ".broken"); err != nil || string(b) != content {
				t.Errorf("backup = %q, %v", b, err)
			}
		})
	}
}

func TestSaveOnlyWhenChanged(t *testing.T) {
	path := tempPath(t)
	s, _ := Open(path)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	os.Remove(path)

	s.Update(func(c *Config) { c.SetState(c.State()) }) // no change
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("unchanged settings were written")
	}

	s.Update(func(c *Config) { c.Controller.VolumeDB = -1 })
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("changed settings were not written")
	}
}

func TestGetReturnsACopy(t *testing.T) {
	s, _ := Open(tempPath(t))
	c := s.Get()
	c.Controller.Speakers[0].Name = "changed"
	if s.Get().Controller.Speakers[0].Name == "changed" {
		t.Error("Get exposed internal slices")
	}
}

func TestKeyAcceptsNumbers(t *testing.T) {
	path := tempPath(t)
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("version: 2\nmidi:\n  mute_key: 16\n  fader_channel: 8\n"), 0o600)
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m := s.Get().MCUConfig(); m.MuteKey != gomcu.Mute1 || m.FaderChannel != gomcu.Master {
		t.Errorf("mcu = %+v", m)
	}
}
