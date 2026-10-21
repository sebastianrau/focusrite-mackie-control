package protocol

import "testing"

func arrival(t *testing.T, name string) *Device {
	t.Helper()
	m, err := Decode(fixture(t, name))
	if err != nil {
		t.Fatal(err)
	}
	d := m.(DeviceArrival).Device
	return &d
}

func TestDecodeDeviceArrival18i20(t *testing.T) {
	d := arrival(t, "device-arrival-18i20.xml")
	if d.SerialNumber != "P9EAC6K250F325" {
		t.Errorf("serial = %q", d.SerialNumber)
	}
	hw := d.Monitoring.HardwareControls.Controls
	if hw.Mute.ID != 1679 || hw.Dim.ID != 1678 {
		t.Errorf("hardware mute/dim = %d/%d, want 1679/1678", hw.Mute.ID, hw.Dim.ID)
	}
	out := d.Outputs.Analogues[0]
	if out.StereoName != "Monitor Outputs 1-2" || out.Nickname.ID != 1456 || out.Mute.ID != 1453 || out.Gain.ID != 1458 || out.Meter.ID != 1450 {
		t.Errorf("first output = %+v", out)
	}
}

func TestApplySetUpdatesDeviceInPlace(t *testing.T) {
	d := arrival(t, "device-arrival-18i20.xml")
	unknown, err := d.ApplySet(Set{DevID: 3, Items: []Item{
		StringItem(1686, "48000"), // clocking/sample-rate
		StringItem(1456, "Mains"), // outputs/analogue[0]/nickname
		BoolItem(1679, true),      // monitoring/.../mute
		StringItem(999999, "x"),
		{ID: 0, Value: "0"},    // sent by Focusrite Control, belongs to nothing
		StringItem(76, "-6.5"), // mixer input gain is fractional
	}})
	if err != nil {
		t.Fatal(err)
	}
	if d.Clocking.SampleRate.Value != "48000" {
		t.Errorf("sample rate = %q", d.Clocking.SampleRate.Value)
	}
	if d.Outputs.Analogues[0].Nickname.Value != "Mains" {
		t.Errorf("nickname = %q", d.Outputs.Analogues[0].Nickname.Value)
	}
	if !d.Monitoring.HardwareControls.Controls.Mute.Value {
		t.Error("hardware mute not set")
	}
	if len(unknown) != 1 || unknown[0] != 999999 {
		t.Errorf("unknown = %v", unknown)
	}
}

func TestApplySetReportsBadValues(t *testing.T) {
	d := arrival(t, "device-arrival-18i20.xml")
	if _, err := d.ApplySet(Set{Items: []Item{StringItem(1679, "maybe")}}); err == nil {
		t.Error("expected an error for a non-bool value")
	}
}

func TestDecodeDeviceArrival4i4(t *testing.T) {
	d := arrival(t, "device-arrival-4i4.xml")
	if d.Model != "Scarlett 4i4 (3rd Gen)" || d.ID != 1 {
		t.Errorf("device = %q id %d", d.Model, d.ID)
	}
	if _, err := d.ApplySet(Set{Items: []Item{StringItem(d.Outputs.Analogues[0].Nickname.ID, "Main")}}); err != nil {
		t.Fatal(err)
	}
}
