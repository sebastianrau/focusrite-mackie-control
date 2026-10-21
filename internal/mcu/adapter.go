package mcu

import (
	"context"
	"time"

	"github.com/sebastianrau/gomcu"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// meterRefresh is how often the level meter is resent. MCU meters decay on
// their own, so a steady signal must be refreshed.
const meterRefresh = 100 * time.Millisecond

// Config maps controller functions to surface controls.
type Config struct {
	InputPort       string
	OutputPort      string
	FaderChannel    gomcu.Channel
	MuteKey         gomcu.Switch
	DimKey          gomcu.Switch
	SpeakerKeys     [domain.NumSpeakers]gomcu.Switch
	LogarithmicFade bool
}

// DefaultConfig matches the Stream Deck profile shipped in streamdeck/.
func DefaultConfig() Config {
	return Config{
		InputPort:    "IAC StreamDeckToController",
		OutputPort:   "IAC ControllerToStreamDeck",
		FaderChannel: gomcu.Channel1,
		MuteKey:      gomcu.Mute1,
		DimKey:       gomcu.Solo1,
		SpeakerKeys:  [domain.NumSpeakers]gomcu.Switch{gomcu.Trim, gomcu.Touch, gomcu.Write, gomcu.F4, gomcu.Read},
	}
}

// Surface is the connection to the control surface. *MidiSurface implements it.
type Surface interface {
	Events() <-chan Event
	Send(Command)
}

// Adapter mirrors the controller state on the surface and turns surface
// input into commands. All fields are owned by the Run goroutine.
type Adapter struct {
	cfg     Config
	surface Surface
	submit  func(domain.Command)
	snaps   <-chan domain.Snapshot

	connected bool
	snap      domain.Snapshot
	ready     bool
	// shown is what the surface currently displays; nil after (re)connect.
	shown *domain.State
	// faderDB is the volume the fader is at. Moving the fader updates it, so
	// the resulting volume change is not sent back to a fader in motion.
	faderDB int
}

func NewAdapter(cfg Config, surface Surface, submit func(domain.Command), snaps <-chan domain.Snapshot) *Adapter {
	return &Adapter{cfg: cfg, surface: surface, submit: submit, snaps: snaps}
}

func (a *Adapter) Run(ctx context.Context) {
	meter := time.NewTicker(meterRefresh)
	defer meter.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-a.surface.Events():
			a.handle(e)
		case s := <-a.snaps:
			a.snap, a.ready = s, true
			a.render()
		case <-meter.C:
			a.sendMeter()
		}
	}
}

func (a *Adapter) handle(e Event) {
	switch e := e.(type) {
	case Connected:
		a.connected = e.Connected
		a.shown = nil
		a.render()

	case KeyPressed:
		a.keyPressed(e.Key)

	case FaderMoved:
		if e.Channel != a.cfg.FaderChannel {
			return
		}
		db := faderToDB(e.Position, a.cfg.LogarithmicFade)
		a.faderDB = db
		a.submit(domain.SetVolume{DB: db})
	}
}

func (a *Adapter) keyPressed(key gomcu.Switch) {
	switch key {
	case a.cfg.MuteKey:
		a.submit(domain.ToggleMute{})
		return
	case a.cfg.DimKey:
		a.submit(domain.ToggleDim{})
		return
	case selectKey(a.cfg.FaderChannel):
		// Pressing the fader's select button pulls it to the current volume.
		a.sendFader()
		return
	}
	for _, id := range domain.Speakers {
		if a.cfg.SpeakerKeys[id] == key {
			a.submit(domain.ToggleSpeaker{ID: id})
			return
		}
	}
	log.Debug("unassigned key", "key", key, "name", keyName(key))
}

// render sends the parts of the state that differ from what the surface shows.
func (a *Adapter) render() {
	if !a.connected || !a.ready {
		return
	}
	s := a.snap.State
	full := a.shown == nil
	if full {
		a.surface.Send(SetLED{Key: selectKey(a.cfg.FaderChannel), On: true})
	}
	led := func(key gomcu.Switch, on, was bool) {
		if full || on != was {
			a.surface.Send(SetLED{Key: key, On: on})
		}
	}
	var prev domain.State
	if !full {
		prev = *a.shown
	}
	led(a.cfg.MuteKey, s.Master.Mute, prev.Master.Mute)
	led(a.cfg.DimKey, s.Master.Dim, prev.Master.Dim)
	for _, id := range domain.Speakers {
		led(a.cfg.SpeakerKeys[id], s.Speakers[id].Selected, prev.Speakers[id].Selected)
	}
	if full || s.Master.VolumeDB != a.faderDB {
		a.sendFader()
	}
	a.shown = &s
}

func (a *Adapter) sendFader() {
	if !a.connected || !a.ready {
		return
	}
	a.faderDB = a.snap.State.Master.VolumeDB
	a.surface.Send(SetFader{Channel: a.cfg.FaderChannel, Position: dbToFader(a.faderDB, a.cfg.LogarithmicFade)})
}

func (a *Adapter) sendMeter() {
	if !a.connected || !a.ready || a.cfg.FaderChannel == gomcu.Master {
		return // the master strip has no meter
	}
	level := meterLevel(max(a.snap.Levels.LeftDB, a.snap.Levels.RightDB))
	if level != gomcu.LessThan60 {
		a.surface.Send(SetMeter{Channel: a.cfg.FaderChannel, Level: level})
	}
}

func selectKey(ch gomcu.Channel) gomcu.Switch {
	if ch > gomcu.Channel8 {
		return gomcu.Switch(0xFF) // the master strip has no select button
	}
	return gomcu.Select1 + gomcu.Switch(ch)
}

func keyName(k gomcu.Switch) string {
	if int(k) < len(gomcu.Names) {
		return gomcu.Names[k]
	}
	return ""
}
