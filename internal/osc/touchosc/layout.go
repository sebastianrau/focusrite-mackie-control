// Package touchosc generates a TouchOSC layout (.tosc) for the monitor
// controller's OSC address space.
//
// A .tosc file is zlib-compressed XML. The element structure follows files
// saved by TouchOSC itself.
package touchosc

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/osc"
)

type rgba struct{ r, g, b, a float64 }

var (
	background = rgba{0.10, 0.10, 0.10, 1}
	grey       = rgba{0.50, 0.50, 0.50, 1}
	white      = rgba{1, 1, 1, 1}
	green      = rgba{0.20, 1.00, 0.20, 1} // the app's speaker colour
	red        = rgba{1.00, 0.20, 0.00, 1} // the app's mute colour
	yellow     = rgba{1.00, 1.00, 0.20, 1} // the app's dim colour
	fader      = rgba{0.30, 0.55, 0.85, 1}
)

type frame struct{ x, y, w, h int }

// node is one control in the layout tree.
type node struct {
	kind     string // GROUP, BUTTON, FADER, LABEL
	name     string
	frame    frame
	color    rgba
	props    []string // extra or overriding <property> elements
	values   []string // <value> elements
	messages []string // <osc> elements
	children []node
}

// Layout returns the .tosc file.
func Layout() ([]byte, error) {
	var xml strings.Builder
	xml.WriteString("<?xml version='1.0' encoding='UTF-8'?><lexml version='3'>")
	root().write(&xml)
	xml.WriteString("</lexml>")

	var out bytes.Buffer
	zw := zlib.NewWriter(&out)
	if _, err := zw.Write([]byte(xml.String())); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func root() node {
	const top, height = 130, 700
	g := node{kind: "GROUP", name: "Monitor Controller", frame: frame{0, 0, 600, 900}, color: background,
		values: []string{value("touch", "false")}}

	g.children = append(g.children,
		label("title", frame{20, 10, 560, 40}, white, 24, "Monitor Controller", ""),
		label("device", frame{20, 50, 560, 30}, grey, 16, "", osc.AddrDeviceModel),
		label("volumeDB", frame{20, 95, 140, 30}, white, 16, "", osc.AddrVolume),
		slider("volume", frame{20, top, 140, height}, fader, true, osc.AddrVolumeNormalized),
		slider("meterLeft", frame{175, top, 30, height}, green, false, osc.AddrMeterLeftNormalized),
		slider("meterRight", frame{210, top, 30, height}, green, false, osc.AddrMeterRightNormalized),
	)
	for i, id := range domain.Speakers {
		f := frame{260, top + i*115, 320, 100}
		g.children = append(g.children,
			toggle("speaker_"+id.String(), f, green, osc.SpeakerAddr(id)),
			label("speakerName_"+id.String(), f, white, 20, id.String(), osc.SpeakerNameAddr(id)),
		)
	}
	mute, dim := frame{260, 720, 155, 110}, frame{425, 720, 155, 110}
	g.children = append(g.children,
		toggle("mute", mute, red, osc.AddrMute),
		label("muteText", mute, white, 20, "Mute", ""),
		toggle("dim", dim, yellow, osc.AddrDim),
		label("dimText", dim, white, 20, "Dim", ""),
	)
	return g
}

// toggle is a button that switches on press and mirrors the app's state.
func toggle(name string, f frame, c rgba, addr string) node {
	return node{kind: "BUTTON", name: name, frame: f, color: c,
		props: []string{
			prop("i", "buttonType", "2"), // toggle on press
			prop("b", "press", "1"),
			prop("b", "release", "1"),
			prop("b", "valuePosition", "0"),
		},
		values:   []string{value("touch", "false"), value("x", "0")},
		messages: []string{oscMessage(addr, "x", "INTEGER", true)},
	}
}

// slider is a vertical fader; a non-interactive one serves as a meter.
func slider(name string, f frame, c rgba, interactive bool, addr string) node {
	return node{kind: "FADER", name: name, frame: f, color: c,
		props: []string{
			prop("b", "interactive", boolStr(interactive)),
			prop("b", "bar", "1"),
			prop("i", "barDisplay", "0"),
			prop("b", "cursor", boolStr(interactive)),
			prop("i", "cursorDisplay", "0"),
			prop("b", "grid", boolStr(interactive)),
			prop("i", "gridSteps", "10"),
			prop("i", "response", "0"),
			prop("i", "responseFactor", "100"),
		},
		values:   []string{value("touch", "false"), value("x", "0")},
		messages: []string{oscMessage(addr, "x", "FLOAT", interactive)},
	}
}

// label shows fixed text, or the string the app sends to addr.
func label(name string, f frame, textColor rgba, size int, text, addr string) node {
	n := node{kind: "LABEL", name: name, frame: f, color: rgba{0, 0, 0, 0},
		props: []string{
			prop("b", "interactive", "0"),
			prop("b", "background", "0"),
			prop("b", "outline", "0"),
			prop("i", "font", "0"),
			prop("i", "textAlignH", "2"),
			prop("i", "textAlignV", "2"),
			prop("b", "textClip", "1"),
			colorProp("textColor", textColor),
			prop("i", "textLength", "0"),
			prop("i", "textSize", fmt.Sprint(size)),
		},
		values: []string{value("text", text), value("touch", "false")},
	}
	if addr != "" {
		n.messages = []string{oscMessage(addr, "text", "STRING", false)}
	}
	return n
}

// common properties every control has, as TouchOSC writes them.
func (n node) properties() []string {
	base := map[string]string{
		"background":      prop("b", "background", "1"),
		"color":           colorProp("color", n.color),
		"cornerRadius":    prop("f", "cornerRadius", "1"),
		"frame":           fmt.Sprintf("<property type='r'><key><![CDATA[frame]]></key><value><x>%d</x><y>%d</y><w>%d</w><h>%d</h></value></property>", n.frame.x, n.frame.y, n.frame.w, n.frame.h),
		"grabFocus":       prop("b", "grabFocus", boolStr(n.kind != "GROUP")),
		"interactive":     prop("b", "interactive", boolStr(n.kind != "GROUP")),
		"locked":          prop("b", "locked", "0"),
		"name":            fmt.Sprintf("<property type='s'><key><![CDATA[name]]></key><value><![CDATA[%s]]></value></property>", n.name),
		"orientation":     prop("i", "orientation", "0"),
		"outline":         prop("b", "outline", "1"),
		"outlineStyle":    prop("i", "outlineStyle", "1"),
		"pointerPriority": prop("i", "pointerPriority", "0"),
		"shape":           prop("i", "shape", "1"),
		"visible":         prop("b", "visible", "1"),
	}
	order := []string{"background", "color", "cornerRadius", "frame", "grabFocus", "interactive", "locked",
		"name", "orientation", "outline", "outlineStyle", "pointerPriority", "shape", "visible"}
	var out []string
	for _, p := range n.props {
		key := between(p, "<key><![CDATA[", "]]>")
		if _, ok := base[key]; ok {
			base[key] = p // override
			continue
		}
		out = append(out, p)
	}
	for _, k := range order {
		out = append(out, base[k])
	}
	return out
}

func (n node) write(b *strings.Builder) {
	fmt.Fprintf(b, "<node ID='%s' type='%s'><properties>", uuid(), n.kind)
	b.WriteString(strings.Join(n.properties(), ""))
	b.WriteString("</properties><values>")
	b.WriteString(strings.Join(n.values, ""))
	b.WriteString("</values>")
	if len(n.messages) > 0 {
		b.WriteString("<messages>" + strings.Join(n.messages, "") + "</messages>")
	}
	if len(n.children) > 0 {
		b.WriteString("<children>")
		for _, c := range n.children {
			c.write(b)
		}
		b.WriteString("</children>")
	}
	b.WriteString("</node>")
}

func prop(typ, key, val string) string {
	return fmt.Sprintf("<property type='%s'><key><![CDATA[%s]]></key><value>%s</value></property>", typ, key, val)
}

func colorProp(key string, c rgba) string {
	return fmt.Sprintf("<property type='c'><key><![CDATA[%s]]></key><value><r>%g</r><g>%g</g><b>%g</b><a>%g</a></value></property>", key, c.r, c.g, c.b, c.a)
}

func value(key, def string) string {
	return fmt.Sprintf("<value><key><![CDATA[%s]]></key><locked>0</locked><lockedDefaultCurrent>0</lockedDefaultCurrent><default><![CDATA[%s]]></default><defaultPull>0</defaultPull></value>", key, def)
}

// oscMessage binds a control value to an address. Every message receives;
// send is set for controls the user operates.
func oscMessage(addr, variable, conversion string, send bool) string {
	return fmt.Sprintf("<osc><enabled>1</enabled><send>%s</send><receive>1</receive><feedback>0</feedback>"+
		"<connections>00001</connections><triggers><trigger><var><![CDATA[%s]]></var><condition>ANY</condition></trigger></triggers>"+
		"<path><partial><type>CONSTANT</type><conversion>STRING</conversion><value><![CDATA[%s]]></value><scaleMin>0</scaleMin><scaleMax>1</scaleMax></partial></path>"+
		"<arguments><partial><type>VALUE</type><conversion>%s</conversion><value><![CDATA[%s]]></value><scaleMin>0</scaleMin><scaleMax>1</scaleMax></partial></arguments></osc>",
		boolStr(send), variable, addr, conversion, variable)
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func between(s, start, end string) string {
	_, rest, ok := strings.Cut(s, start)
	if !ok {
		return ""
	}
	v, _, _ := strings.Cut(rest, end)
	return v
}

func uuid() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
