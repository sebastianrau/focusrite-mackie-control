package osc

import (
	"math"
	"strings"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// The address space. Every state address works both ways: clients send it to
// change the value and the app sends it back with the current value. See
// docs/osc.md.
const (
	AddrVolume           = "/monitor/volume"            // float dB -127..0
	AddrVolumeNormalized = "/monitor/volume/normalized" // float 0..1
	AddrMute             = "/monitor/mute"              // int 0/1
	AddrMuteToggle       = "/monitor/mute/toggle"
	AddrDim              = "/monitor/dim" // int 0/1
	AddrDimToggle        = "/monitor/dim/toggle"

	AddrMeterLeft            = "/monitor/meter/left"             // float dBFS, out only
	AddrMeterRight           = "/monitor/meter/right"            // float dBFS, out only
	AddrMeterLeftNormalized  = "/monitor/meter/left/normalized"  // float 0..1 (-60..0 dB), out only
	AddrMeterRightNormalized = "/monitor/meter/right/normalized" // float 0..1 (-60..0 dB), out only

	AddrDeviceConnected = "/monitor/device/connected" // int 0/1, out only
	AddrDeviceModel     = "/monitor/device/model"     // string, out only

	AddrSync        = "/monitor/sync"        // send the full state now
	AddrSubscribe   = "/monitor/subscribe"   // [int port]: receive updates for 60 s
	AddrUnsubscribe = "/monitor/unsubscribe" // [int port]
)

// Speaker addresses use these names: /monitor/speaker/<name> (int 0/1),
// /monitor/speaker/<name>/toggle, and the outgoing /name (string) and
// /enabled (int 0/1).
var speakerNames = [domain.NumSpeakers]string{"a", "b", "c", "d", "sub"}

func SpeakerAddr(id domain.SpeakerID) string     { return "/monitor/speaker/" + speakerNames[id] }
func SpeakerToggle(id domain.SpeakerID) string   { return SpeakerAddr(id) + "/toggle" }
func SpeakerNameAddr(id domain.SpeakerID) string { return SpeakerAddr(id) + "/name" }
func SpeakerEnabled(id domain.SpeakerID) string  { return SpeakerAddr(id) + "/enabled" }

// parseSpeaker splits "/monitor/speaker/<name>[/rest]".
func parseSpeaker(addr string) (id domain.SpeakerID, rest string, ok bool) {
	tail, found := strings.CutPrefix(addr, "/monitor/speaker/")
	if !found {
		return 0, "", false
	}
	name, rest, _ := strings.Cut(tail, "/")
	for i, n := range speakerNames {
		if n == name {
			return domain.SpeakerID(i), rest, true
		}
	}
	return 0, "", false
}

const meterFloorDB = -60.0

func normalizeVolume(db int) float32 {
	return float32(db-domain.MinVolumeDB) / float32(domain.MaxVolumeDB-domain.MinVolumeDB)
}

func denormalizeVolume(v float64) int {
	v = min(max(v, 0), 1)
	return domain.MinVolumeDB + int(math.Round(v*float64(domain.MaxVolumeDB-domain.MinVolumeDB)))
}

func normalizeMeter(db float64) float32 {
	return float32((min(max(db, meterFloorDB), 0) - meterFloorDB) / -meterFloorDB)
}

func boolInt(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// stateMessages describes the state of s, without the meter.
func stateMessages(s domain.Snapshot) []Message {
	st := s.State
	msgs := []Message{
		{AddrVolume, []any{float32(st.Master.VolumeDB)}},
		{AddrVolumeNormalized, []any{normalizeVolume(st.Master.VolumeDB)}},
		{AddrMute, []any{boolInt(st.Master.Mute)}},
		{AddrDim, []any{boolInt(st.Master.Dim)}},
		{AddrDeviceConnected, []any{boolInt(s.Device.ServerConnected && s.Device.Present)}},
		{AddrDeviceModel, []any{s.Device.Model}},
	}
	for _, id := range domain.Speakers {
		spk := st.Speakers[id]
		msgs = append(msgs,
			Message{SpeakerAddr(id), []any{boolInt(spk.Selected)}},
			Message{SpeakerNameAddr(id), []any{spk.Name}},
			Message{SpeakerEnabled(id), []any{boolInt(!spk.Disabled)}},
		)
	}
	return msgs
}

func meterMessages(l domain.Levels) []Message {
	return []Message{
		{AddrMeterLeft, []any{float32(l.LeftDB)}},
		{AddrMeterRight, []any{float32(l.RightDB)}},
		{AddrMeterLeftNormalized, []any{normalizeMeter(l.LeftDB)}},
		{AddrMeterRightNormalized, []any{normalizeMeter(l.RightDB)}},
	}
}

// command turns an incoming message into a controller command. Toggles and
// selections ignore "off" values: momentary buttons send 1 on press and 0 on
// release, and only the press should count.
func command(m Message) (domain.Command, bool) {
	switch m.Address {
	case AddrVolume:
		if v, ok := m.Float(); ok {
			return domain.SetVolume{DB: int(math.Round(v))}, true
		}
	case AddrVolumeNormalized:
		if v, ok := m.Float(); ok {
			return domain.SetVolume{DB: denormalizeVolume(v)}, true
		}
	case AddrMute:
		if on, ok := m.On(); ok && len(m.Args) > 0 {
			return domain.SetMute{On: on}, true
		}
	case AddrDim:
		if on, ok := m.On(); ok && len(m.Args) > 0 {
			return domain.SetDim{On: on}, true
		}
	case AddrMuteToggle:
		if on, ok := m.On(); ok && on {
			return domain.ToggleMute{}, true
		}
	case AddrDimToggle:
		if on, ok := m.On(); ok && on {
			return domain.ToggleDim{}, true
		}
	default:
		id, rest, ok := parseSpeaker(m.Address)
		if !ok {
			return nil, false
		}
		on, valid := m.On()
		switch {
		case rest == "" && valid && len(m.Args) > 0:
			return domain.SelectSpeaker{ID: id, On: on}, true
		case rest == "toggle" && valid && on:
			return domain.ToggleSpeaker{ID: id}, true
		}
	}
	return nil, false
}

// Known reports whether the app sends or understands addr. Layout generators
// use it to stay in sync with the address space.
func Known(addr string) bool {
	if _, ok := command(Message{Address: addr, Args: []any{int32(1)}}); ok {
		return true
	}
	switch addr {
	case AddrSync, AddrSubscribe, AddrUnsubscribe:
		return true
	}
	for _, m := range append(stateMessages(domain.Snapshot{}), meterMessages(domain.Silence)...) {
		if m.Address == addr {
			return true
		}
	}
	return false
}
