package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"testing/iotest"
)

func frame(t *testing.T, body []byte) []byte {
	t.Helper()
	return fmt.Appendf(nil, "Length=%06X %s", len(body), body)
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The expected strings were captured from the previous implementation, which
// is known to work against Focusrite Control.
func TestEncodeWireFormat(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{
			Set{DevID: 3, Items: []Item{BoolItem(1679, true), IntItem(1458, -20), StringItem(1456, "Main")}},
			`Length=00006E <set devid="3"><item id="1679" value="true"/><item id="1458" value="-20"/><item id="1456" value="Main"/></set>`,
		},
		{KeepAlive{}, `Length=00000D <keep-alive/>`},
		{
			ClientDetails{Hostname: "Monitor Controller", ClientKey: "123456789"},
			`Length=000046 <client-details hostname="Monitor Controller" client-key="123456789"/>`,
		},
		{SubscribeMessage{DeviceID: 3, Subscribe: true}, `Length=00002E <device-subscribe devid="3" subscribe="true"/>`},
	}
	for _, tt := range tests {
		got, err := Encode(tt.in)
		if err != nil {
			t.Fatalf("Encode(%T): %v", tt.in, err)
		}
		if string(got) != tt.want {
			t.Errorf("Encode(%T)\n got %s\nwant %s", tt.in, got, tt.want)
		}
	}
}

func TestReaderHandlesStreamFraming(t *testing.T) {
	arrival := fixture(t, "device-arrival-18i20.xml")
	var stream bytes.Buffer
	stream.Write(frame(t, []byte(`<keep-alive/>`)))
	stream.WriteString("\n")
	stream.Write(frame(t, arrival)) // larger than any single socket read
	stream.Write(frame(t, []byte(`<device-removal id="3"/>`)))

	// One byte per Read is the worst case for frame reassembly.
	r := NewReader(iotest.OneByteReader(&stream))

	if m, err := r.Next(); err != nil {
		t.Fatal(err)
	} else if _, ok := m.(KeepAlive); !ok {
		t.Fatalf("first message = %#v, %v", m, err)
	}
	m, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	da, ok := m.(DeviceArrival)
	if !ok {
		t.Fatalf("second message is %T", m)
	}
	if da.Device.Model != "Scarlett 18i20 (3rd Gen)" || da.Device.ID != 3 {
		t.Errorf("device = %q id %d", da.Device.Model, da.Device.ID)
	}
	if m, err := r.Next(); err != nil || m.(DeviceRemoval).ID != 3 {
		t.Fatalf("third message = %#v, %v", m, err)
	}
	if _, err := r.Next(); !errors.Is(err, io.EOF) {
		t.Errorf("after last frame: err = %v, want EOF", err)
	}
}

func TestReaderRejectsGarbage(t *testing.T) {
	r := NewReader(bytes.NewBufferString("HELLO=000010 <keep-alive/>"))
	if _, err := r.Next(); err == nil {
		t.Error("expected an error for an invalid header")
	}
}

func TestReaderTruncatedFrame(t *testing.T) {
	r := NewReader(bytes.NewBufferString("Length=0000FF <set"))
	if _, err := r.Next(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("err = %v, want ErrUnexpectedEOF", err)
	}
}

func TestDecodeUnknownMessage(t *testing.T) {
	if _, err := Decode([]byte(`<bogus/>`)); !errors.Is(err, ErrUnknownMessage) {
		t.Errorf("err = %v, want ErrUnknownMessage", err)
	}
}

func TestDecodeFrameIgnoresTrailingBytes(t *testing.T) {
	buf := make([]byte, 1024) // a UDP read buffer
	copy(buf, frame(t, []byte(`<server-announcement app='SAFFIRE-CONTROL' version='4' hostname='mac.local' port='55145'/>`)))
	m, err := DecodeFrame(buf)
	if err != nil {
		t.Fatal(err)
	}
	if a := m.(ServerAnnouncement); a.Port != 55145 {
		t.Errorf("port = %d", a.Port)
	}
}

func TestDecodeApprovalAndSet(t *testing.T) {
	m, err := Decode([]byte(`<approval hostname="Monitor Controller" id="1" type="response" authorised="true"/>`))
	if err != nil || !m.(Approval).Authorised {
		t.Errorf("approval = %#v, %v", m, err)
	}
	m, err = Decode(fixture(t, "set-4i4.xml"))
	if err != nil {
		t.Fatal(err)
	}
	set := m.(Set)
	if set.DevID != 1 || len(set.Items) == 0 || set.Items[0] != (Item{ID: 342, Value: "false"}) {
		t.Errorf("set = devid %d, %d items, first %+v", set.DevID, len(set.Items), set.Items[0])
	}
}
