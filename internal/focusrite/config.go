package focusrite

import (
	"cmp"
	"slices"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite/protocol"
)

// Output maps a speaker to a stereo output pair by Focusrite element IDs.
// Zero IDs are not used.
type Output struct {
	NameID       int
	MuteID       int
	GainID       int
	MeterLeftID  int
	MeterRightID int
}

// Config selects the interface and the elements the controller drives.
type Config struct {
	// SerialNumber of the interface to control. Empty adopts the first one
	// that appears and maps its outputs automatically.
	SerialNumber string
	// MuteID and DimID are the interface's monitor mute and dim controls.
	MuteID  int
	DimID   int
	Outputs [domain.NumSpeakers]Output
}

// DeviceInfo describes an attached interface for the settings UI.
type DeviceInfo struct {
	Model        string
	SerialNumber string
	MuteID       int
	DimID        int
	Outputs      []NamedOutput
}

// NamedOutput is a stereo output pair as Focusrite Control names it.
type NamedOutput struct {
	Name string
	Output
}

// Describe extracts the stereo outputs and monitor controls of a device.
// A pair is announced by its left channel, which carries the stereo name; the
// right channel follows it.
func Describe(d *protocol.Device) DeviceInfo {
	hw := d.Monitoring.HardwareControls.Controls
	info := DeviceInfo{
		Model:        d.Model,
		SerialNumber: d.SerialNumber,
		MuteID:       hw.Mute.ID,
		DimID:        hw.Dim.ID,
	}
	outs := d.Outputs.Analogues
	for i := 0; i+1 < len(outs); i++ {
		l := outs[i]
		if l.StereoName == "" || l.Mute.ID == 0 {
			continue
		}
		info.Outputs = append(info.Outputs, NamedOutput{
			Name: l.StereoName,
			Output: Output{
				NameID:       l.Nickname.ID,
				MuteID:       l.Mute.ID,
				GainID:       l.Gain.ID,
				MeterLeftID:  l.Meter.ID,
				MeterRightID: outs[i+1].Meter.ID,
			},
		})
		i++
	}
	return info
}

// AutoConfig assigns the device's stereo outputs to the speakers in order.
func AutoConfig(info DeviceInfo) Config {
	cfg := Config{SerialNumber: info.SerialNumber, MuteID: info.MuteID, DimID: info.DimID}
	for i, out := range info.Outputs {
		if i >= domain.NumSpeakers {
			break
		}
		cfg.Outputs[i] = out.Output
	}
	return cfg
}

func sortDevices(list []DeviceInfo) {
	slices.SortFunc(list, func(a, b DeviceInfo) int {
		return cmp.Or(cmp.Compare(a.Model, b.Model), cmp.Compare(a.SerialNumber, b.SerialNumber))
	})
}

// DesiredItems lists the element values that represent state on the device.
func DesiredItems(cfg Config, s domain.State) []protocol.Item {
	var items []protocol.Item
	add := func(it protocol.Item) {
		if it.ID != 0 {
			items = append(items, it)
		}
	}
	add(protocol.BoolItem(cfg.MuteID, s.Master.Mute))
	add(protocol.BoolItem(cfg.DimID, s.Master.Dim))
	gain := s.OutputGainDB()
	for _, id := range domain.Speakers {
		out, spk := cfg.Outputs[id], s.Speakers[id]
		add(protocol.StringItem(out.NameID, spk.Name))
		if spk.Disabled {
			continue
		}
		add(protocol.BoolItem(out.MuteID, s.OutputMuted(id)))
		add(protocol.IntItem(out.GainID, gain))
	}
	return items
}
