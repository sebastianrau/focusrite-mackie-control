// Package protocol implements the XML protocol Focusrite Control speaks with
// remote clients (the one its iOS app uses).
//
// Every message is framed as "Length=XXXXXX <xml>", where XXXXXX is the byte
// length of the XML as six hex digits.
package protocol

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/ECUST-XX/xml"
)

const (
	framePrefix = "Length="
	headerLen   = len(framePrefix) + 6 + 1 // prefix, six hex digits, space
)

// ErrUnknownMessage is returned by Decode for well-formed XML with an
// unsupported root element. Callers usually log it and carry on.
var ErrUnknownMessage = errors.New("unknown message")

// Encode marshals v and adds the length header. Empty elements are written in
// short form (<keep-alive/>), which Focusrite Control expects.
func Encode(v any) ([]byte, error) {
	body, err := xml.MarshalIndentShortForm(v, "", "")
	if err != nil {
		return nil, fmt.Errorf("encode %T: %w", v, err)
	}
	body = bytes.ReplaceAll(body, []byte(" />"), []byte("/>"))
	return fmt.Appendf(nil, "%s%06X %s", framePrefix, len(body), body), nil
}

// Reader reads framed messages from a stream.
type Reader struct {
	r *bufio.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: bufio.NewReaderSize(r, 64*1024)}
}

// ReadFrame returns the XML body of the next frame.
func (r *Reader) ReadFrame() ([]byte, error) {
	if err := r.skipSpace(); err != nil {
		return nil, err
	}
	header := make([]byte, headerLen)
	if _, err := io.ReadFull(r.r, header); err != nil {
		return nil, err
	}
	n, err := parseHeader(header)
	if err != nil {
		return nil, err
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r.r, body); err != nil {
		return nil, fmt.Errorf("read %d byte frame: %w", n, err)
	}
	return body, nil
}

// Next reads and decodes the next message.
func (r *Reader) Next() (any, error) {
	body, err := r.ReadFrame()
	if err != nil {
		return nil, err
	}
	return Decode(body)
}

func (r *Reader) skipSpace() error {
	for {
		b, err := r.r.ReadByte()
		if err != nil {
			return err
		}
		if b != ' ' && b != '\n' && b != '\r' && b != '\t' {
			return r.r.UnreadByte()
		}
	}
}

func parseHeader(h []byte) (int, error) {
	if !bytes.HasPrefix(h, []byte(framePrefix)) || h[len(h)-1] != ' ' {
		return 0, fmt.Errorf("invalid frame header %q", h)
	}
	n, err := strconv.ParseUint(string(h[len(framePrefix):len(h)-1]), 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid frame length %q: %w", h, err)
	}
	return int(n), nil
}

// DecodeFrame decodes a single frame held entirely in buf, such as a UDP
// datagram. Bytes after the frame are ignored.
func DecodeFrame(buf []byte) (any, error) {
	if len(buf) < headerLen {
		return nil, fmt.Errorf("frame too short (%d bytes)", len(buf))
	}
	n, err := parseHeader(buf[:headerLen])
	if err != nil {
		return nil, err
	}
	if len(buf)-headerLen < n {
		return nil, fmt.Errorf("frame truncated: want %d bytes, have %d", n, len(buf)-headerLen)
	}
	return Decode(buf[headerLen : headerLen+n])
}

// Decode unmarshals an XML message body into its typed value.
func Decode(body []byte) (any, error) {
	var root struct{ XMLName xml.Name }
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	switch name := root.XMLName.Local; name {
	case "client-details":
		return decodeAs[ClientDetails](name, body)
	case "set":
		return decodeAs[Set](name, body)
	case "device-arrival":
		return decodeAs[DeviceArrival](name, body)
	case "device-removal":
		return decodeAs[DeviceRemoval](name, body)
	case "keep-alive":
		return decodeAs[KeepAlive](name, body)
	case "approval":
		return decodeAs[Approval](name, body)
	case "server-announcement":
		return decodeAs[ServerAnnouncement](name, body)
	case "device-subscribe":
		return decodeAs[SubscribeMessage](name, body)
	case "client-discovery":
		return decodeAs[DiscoveryRequest](name, body)
	default:
		return nil, fmt.Errorf("%w <%s>", ErrUnknownMessage, name)
	}
}

func decodeAs[T any](name string, body []byte) (any, error) {
	var v T
	if err := xml.Unmarshal(body, &v); err != nil {
		return nil, fmt.Errorf("decode <%s>: %w", name, err)
	}
	return v, nil
}
