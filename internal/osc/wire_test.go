package osc

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Reference encoding from the OSC 1.0 specification examples.
func TestEncodeSpecExample(t *testing.T) {
	got, err := Encode(Message{Address: "/oscillator/4/frequency", Args: []any{float32(440.0)}})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("2f6f7363696c6c61746f722f342f6672657175656e6379002c66000043dc0000")
	if !bytes.Equal(got, want) {
		t.Errorf("got  %x\nwant %x", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	in := Message{Address: "/monitor/speaker/a/name", Args: []any{int32(-3), float32(0.5), "KRK", true, false}}
	b, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(b)%4 != 0 {
		t.Errorf("packet length %d not aligned", len(b))
	}
	out, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].String() != in.String() {
		t.Errorf("round trip = %v, want %v", out, in)
	}
}

func TestDecodeBundle(t *testing.T) {
	m1, _ := Encode(Message{Address: "/a", Args: []any{int32(1)}})
	m2, _ := Encode(Message{Address: "/b", Args: []any{"x"}})
	var b bytes.Buffer
	b.WriteString("#bundle\x00")
	b.Write(make([]byte, 8)) // time tag
	for _, m := range [][]byte{m1, m2} {
		b.Write([]byte{0, 0, 0, byte(len(m))})
		b.Write(m)
	}
	msgs, err := Decode(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Address != "/a" || msgs[1].Args[0] != "x" {
		t.Errorf("bundle = %v", msgs)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	for _, p := range [][]byte{
		[]byte("no-slash\x00\x00\x00\x00"),
		[]byte("/a\x00\x00,i\x00\x00\x00\x01"), // truncated int
		[]byte("/a\x00\x00,q\x00\x00"),         // unknown tag
		[]byte("/abc"),                         // no terminator
	} {
		if _, err := Decode(p); err == nil {
			t.Errorf("Decode(%q) succeeded", p)
		}
	}
}

func TestDecodeWithoutTypeTags(t *testing.T) {
	msgs, err := Decode([]byte("/monitor/sync\x00\x00\x00"))
	if err != nil || len(msgs) != 1 || len(msgs[0].Args) != 0 {
		t.Errorf("got %v, %v", msgs, err)
	}
}

func TestOn(t *testing.T) {
	for _, tc := range []struct {
		args []any
		want bool
	}{{nil, true}, {[]any{int32(0)}, false}, {[]any{float32(1)}, true}, {[]any{true}, true}, {[]any{false}, false}} {
		if on, _ := (Message{Args: tc.args}).On(); on != tc.want {
			t.Errorf("On(%v) = %t", tc.args, on)
		}
	}
	if _, ok := (Message{Args: []any{"x"}}).On(); ok {
		t.Error("a string is not a switch value")
	}
}
