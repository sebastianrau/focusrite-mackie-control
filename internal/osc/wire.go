// Package osc connects the controller to Open Sound Control clients such as
// TouchOSC, Open Stage Control or a DAW. wire.go implements the subset of
// OSC 1.0 those clients use.
package osc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Message is one OSC message. Args holds int32, float32, string, bool,
// int64, float64 or nil values.
type Message struct {
	Address string
	Args    []any
}

func (m Message) String() string { return fmt.Sprintf("%s %v", m.Address, m.Args) }

// Encode serialises m. Unsupported argument types are an error.
func Encode(m Message) ([]byte, error) {
	var b bytes.Buffer
	writeString(&b, m.Address)
	tags := []byte{','}
	var data bytes.Buffer
	for _, a := range m.Args {
		switch v := a.(type) {
		case int32:
			tags = append(tags, 'i')
			_ = binary.Write(&data, binary.BigEndian, v)
		case int:
			tags = append(tags, 'i')
			_ = binary.Write(&data, binary.BigEndian, int32(v))
		case float32:
			tags = append(tags, 'f')
			_ = binary.Write(&data, binary.BigEndian, v)
		case float64:
			tags = append(tags, 'f')
			_ = binary.Write(&data, binary.BigEndian, float32(v))
		case string:
			tags = append(tags, 's')
			writeString(&data, v)
		case bool:
			if v {
				tags = append(tags, 'T')
			} else {
				tags = append(tags, 'F')
			}
		default:
			return nil, fmt.Errorf("osc: unsupported argument type %T", a)
		}
	}
	writeString(&b, string(tags))
	b.Write(data.Bytes())
	return b.Bytes(), nil
}

// Decode parses a packet, unpacking bundles into their messages.
func Decode(p []byte) ([]Message, error) {
	if bytes.HasPrefix(p, []byte("#bundle\x00")) {
		return decodeBundle(p)
	}
	m, err := decodeMessage(p)
	if err != nil {
		return nil, err
	}
	return []Message{m}, nil
}

func decodeBundle(p []byte) ([]Message, error) {
	p = p[16:] // "#bundle\0" and the time tag
	var msgs []Message
	for len(p) > 0 {
		if len(p) < 4 {
			return nil, errors.New("osc: truncated bundle")
		}
		n := int(binary.BigEndian.Uint32(p))
		p = p[4:]
		if n > len(p) || n%4 != 0 {
			return nil, errors.New("osc: invalid bundle element size")
		}
		inner, err := Decode(p[:n])
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, inner...)
		p = p[n:]
	}
	return msgs, nil
}

func decodeMessage(p []byte) (Message, error) {
	addr, p, err := readString(p)
	if err != nil {
		return Message{}, err
	}
	if len(addr) == 0 || addr[0] != '/' {
		return Message{}, fmt.Errorf("osc: invalid address %q", addr)
	}
	m := Message{Address: addr}
	if len(p) == 0 {
		return m, nil // very old clients omit the type tags
	}
	tags, p, err := readString(p)
	if err != nil {
		return Message{}, err
	}
	if len(tags) == 0 || tags[0] != ',' {
		return Message{}, fmt.Errorf("osc: invalid type tags %q", tags)
	}
	for _, t := range tags[1:] {
		switch t {
		case 'i', 'f':
			if len(p) < 4 {
				return Message{}, errTruncated
			}
			u := binary.BigEndian.Uint32(p)
			p = p[4:]
			if t == 'i' {
				m.Args = append(m.Args, int32(u))
			} else {
				m.Args = append(m.Args, math.Float32frombits(u))
			}
		case 'h', 'd', 't':
			if len(p) < 8 {
				return Message{}, errTruncated
			}
			u := binary.BigEndian.Uint64(p)
			p = p[8:]
			if t == 'd' {
				m.Args = append(m.Args, math.Float64frombits(u))
			} else {
				m.Args = append(m.Args, int64(u))
			}
		case 's', 'S':
			var s string
			if s, p, err = readString(p); err != nil {
				return Message{}, err
			}
			m.Args = append(m.Args, s)
		case 'b':
			if len(p) < 4 {
				return Message{}, errTruncated
			}
			n := int(binary.BigEndian.Uint32(p))
			if 4+pad(n) > len(p) {
				return Message{}, errTruncated
			}
			m.Args = append(m.Args, append([]byte(nil), p[4:4+n]...))
			p = p[4+pad(n):]
		case 'T':
			m.Args = append(m.Args, true)
		case 'F':
			m.Args = append(m.Args, false)
		case 'N', 'I':
			m.Args = append(m.Args, nil)
		default:
			return Message{}, fmt.Errorf("osc: unsupported type tag %q", t)
		}
	}
	return m, nil
}

var errTruncated = errors.New("osc: truncated message")

func pad(n int) int { return (n + 3) &^ 3 }

func writeString(b *bytes.Buffer, s string) {
	b.WriteString(s)
	for range pad(len(s)+1) - len(s) {
		b.WriteByte(0)
	}
}

func readString(p []byte) (string, []byte, error) {
	i := bytes.IndexByte(p, 0)
	if i < 0 {
		return "", nil, errTruncated
	}
	n := pad(i + 1)
	if n > len(p) {
		return "", nil, errTruncated
	}
	return string(p[:i]), p[n:], nil
}

// Float returns the first argument as a number. Bools count as 0 or 1.
func (m Message) Float() (float64, bool) {
	if len(m.Args) == 0 {
		return 0, false
	}
	switch v := m.Args[0].(type) {
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// On interprets the first argument as a switch: non-zero numbers and true
// are on. A message without arguments counts as on.
func (m Message) On() (on, ok bool) {
	if len(m.Args) == 0 {
		return true, true
	}
	f, ok := m.Float()
	return f != 0, ok
}
