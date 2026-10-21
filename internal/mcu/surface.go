// Package mcu connects the controller to a Mackie Control (MCU) compatible
// surface such as a fader controller or a Stream Deck with a MIDI plugin.
package mcu

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sebastianrau/gomcu"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // registers the MIDI driver

	"github.com/sebastianrau/focusrite-mackie-control/internal/logging"
)

var log = logging.New("mcu")

const (
	reconnectDelay = 3 * time.Second
	portCheck      = time.Second
	displayText    = "Monitor Control"
)

// Event is input from the surface.
type Event interface{ isEvent() }

type Connected struct{ Connected bool }
type KeyPressed struct{ Key gomcu.Switch }
type FaderMoved struct {
	Channel  gomcu.Channel
	Position uint16
}

func (Connected) isEvent()  {}
func (KeyPressed) isEvent() {}
func (FaderMoved) isEvent() {}

// Command is output to the surface.
type Command interface{ message() midi.Message }

type SetLED struct {
	Key gomcu.Switch
	On  bool
}
type SetFader struct {
	Channel  gomcu.Channel
	Position uint16
}
type SetMeter struct {
	Channel gomcu.Channel
	Level   gomcu.MeterLevel
}

func (c SetLED) message() midi.Message {
	if c.On {
		return gomcu.SetLED(c.Key, gomcu.StateOn)
	}
	return gomcu.SetLED(c.Key, gomcu.StateOff)
}
func (c SetFader) message() midi.Message { return gomcu.SetFaderPos(c.Channel, c.Position) }
func (c SetMeter) message() midi.Message { return gomcu.SetMeter(c.Channel, c.Level) }

// decode turns an incoming MIDI message into a surface event.
func decode(msg midi.Message) (Event, bool) {
	var ch, key, vel uint8
	var rel int16
	var abs uint16
	switch {
	case msg.GetNoteOn(&ch, &key, &vel):
		if vel == 0 { // note-on with velocity 0 is a release
			return nil, false
		}
		return KeyPressed{Key: gomcu.Switch(key)}, true
	case msg.GetPitchBend(&ch, &rel, &abs):
		return FaderMoved{Channel: gomcu.Channel(ch), Position: abs}, true
	}
	return nil, false
}

// MidiSurface talks to the surface over a pair of MIDI ports and reconnects
// when they disappear.
type MidiSurface struct {
	inPort, outPort string
	events          chan Event
	cmds            chan Command
}

func NewMidiSurface(inPort, outPort string) *MidiSurface {
	return &MidiSurface{
		inPort:  inPort,
		outPort: outPort,
		events:  make(chan Event, 256),
		cmds:    make(chan Command, 256),
	}
}

func (s *MidiSurface) Events() <-chan Event { return s.events }

// Send queues a command; it is dropped while the surface is disconnected.
func (s *MidiSurface) Send(c Command) {
	select {
	case s.cmds <- c:
	default:
		log.Warn("command queue full, dropping", "command", fmt.Sprintf("%T", c))
	}
}

// Run keeps the surface connected until ctx is cancelled.
func (s *MidiSurface) Run(ctx context.Context) {
	defer midi.CloseDriver()
	failing := false
	for {
		err := s.session(ctx)
		if ctx.Err() != nil {
			return
		}
		if !failing {
			log.Info("MIDI surface unavailable, retrying", "in", s.inPort, "out", s.outPort, "err", err)
			failing = true
		}
		t := time.NewTimer(reconnectDelay)
	wait:
		for {
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				break wait
			case <-s.cmds:
			}
		}
		if err == nil {
			failing = false
		}
	}
}

// session runs one connection to the surface.
func (s *MidiSurface) session(ctx context.Context) error {
	in, err := midi.FindInPort(s.inPort)
	if err != nil {
		return fmt.Errorf("input port %q: %w", s.inPort, err)
	}
	out, err := midi.FindOutPort(s.outPort)
	if err != nil {
		return fmt.Errorf("output port %q: %w", s.outPort, err)
	}
	if err := out.Open(); err != nil {
		return fmt.Errorf("open output: %w", err)
	}
	defer out.Close()
	send, err := midi.SendTo(out)
	if err != nil {
		return err
	}
	if err := gomcu.Reset(out); err != nil {
		return fmt.Errorf("reset surface: %w", err)
	}
	for _, m := range gomcu.SetTimeDisplay(displayText) {
		if err := send(m); err != nil {
			return err
		}
	}

	stop, err := midi.ListenTo(in, func(msg midi.Message, _ int32) {
		if e, ok := decode(msg); ok {
			select {
			case s.events <- e:
			default:
				log.Warn("event queue full, dropping input")
			}
		}
	})
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer func() {
		stop()
		_ = in.Close()
	}()

	log.Info("MIDI surface connected", "in", s.inPort, "out", s.outPort)
	s.emit(ctx, Connected{Connected: true})
	defer s.emit(ctx, Connected{Connected: false})

	check := time.NewTicker(portCheck)
	defer check.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case c := <-s.cmds:
			if err := send(c.message()); err != nil {
				return fmt.Errorf("send: %w", err)
			}
		case <-check.C:
			if !in.IsOpen() || !out.IsOpen() || !portExists(s.inPort, s.outPort) {
				return errors.New("MIDI port closed")
			}
		}
	}
}

func (s *MidiSurface) emit(ctx context.Context, e Event) {
	select {
	case s.events <- e:
	case <-ctx.Done():
	}
}

func portExists(in, out string) bool {
	return hasPort(midi.GetInPorts(), in) && hasPort(midi.GetOutPorts(), out)
}

func hasPort[P drivers.Port](ports []P, name string) bool {
	for _, p := range ports {
		if p.String() == name {
			return true
		}
	}
	return false
}

// InPorts and OutPorts list the available MIDI port names.
func InPorts() []string  { return portNames(midi.GetInPorts()) }
func OutPorts() []string { return portNames(midi.GetOutPorts()) }

func portNames[P drivers.Port](ports []P) []string {
	names := make([]string, 0, len(ports))
	for _, p := range ports {
		names = append(names, p.String())
	}
	return names
}
