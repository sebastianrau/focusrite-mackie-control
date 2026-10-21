package touchosc

import (
	"bytes"
	"compress/zlib"
	"encoding/xml"
	"io"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/osc"
)

type xmlNode struct {
	Type       string        `xml:"type,attr"`
	Properties []xmlProperty `xml:"properties>property"`
	Messages   []xmlOSC      `xml:"messages>osc"`
	Children   []xmlNode     `xml:"children>node"`
}

type xmlProperty struct {
	Type  string `xml:"type,attr"`
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type xmlOSC struct {
	Send    int    `xml:"send"`
	Receive int    `xml:"receive"`
	Path    string `xml:"path>partial>value"`
	Conv    string `xml:"arguments>partial>conversion"`
}

func load(t *testing.T) xmlNode {
	t.Helper()
	b, err := Layout()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("not zlib: %v", err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		XMLName xml.Name
		Version string  `xml:"version,attr"`
		Root    xmlNode `xml:"node"`
	}
	if err := xml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}
	if doc.XMLName.Local != "lexml" || doc.Version != "3" {
		t.Fatalf("root = %s v%s", doc.XMLName.Local, doc.Version)
	}
	return doc.Root
}

func walk(n xmlNode, f func(xmlNode)) {
	f(n)
	for _, c := range n.Children {
		walk(c, f)
	}
}

// Every address in the layout must be one the app sends or understands.
func TestAddressesAreKnown(t *testing.T) {
	paths := 0
	walk(load(t), func(n xmlNode) {
		for _, m := range n.Messages {
			paths++
			if !osc.Known(m.Path) {
				t.Errorf("%s uses unknown address %q", n.Type, m.Path)
			}
			if m.Receive != 1 {
				t.Errorf("%s %s does not receive", n.Type, m.Path)
			}
		}
	})
	// device and volume labels, volume, 2 meters, 5 speakers with names, mute, dim
	if paths != 17 {
		t.Errorf("found %d OSC bindings, want 17", paths)
	}
}

func TestEveryControlHasTheBasics(t *testing.T) {
	walk(load(t), func(n xmlNode) {
		keys := map[string]int{}
		for _, p := range n.Properties {
			keys[p.Key]++
		}
		for _, k := range []string{"frame", "name", "color", "visible", "interactive"} {
			if keys[k] != 1 {
				t.Errorf("%s has %d %q properties", n.Type, keys[k], k)
			}
		}
	})
}

// The root script turns /monitor/speaker/<x>/enabled into a locked, grey
// button; it must find every speaker button and name by the names it builds.
func TestDisabledSpeakerScript(t *testing.T) {
	root := load(t)
	names := map[string]bool{}
	for _, c := range root.Children {
		for _, p := range c.Properties {
			if p.Key == "name" {
				names[p.Value] = true
			}
		}
	}
	for _, id := range domain.Speakers {
		slug := path.Base(osc.SpeakerAddr(id))
		for _, n := range []string{"speaker_" + slug, "speakerName_" + slug} {
			if !names[n] {
				t.Errorf("no control named %q for the script", n)
			}
		}
		if !osc.Known(osc.SpeakerEnabled(id)) {
			t.Errorf("app does not send %s", osc.SpeakerEnabled(id))
		}
	}
	var script string
	for _, p := range root.Properties {
		if p.Key == "script" {
			script = p.Value
		}
	}
	if !strings.Contains(script, "/monitor/speaker/(%w+)/enabled") || !strings.Contains(script, "button.interactive = on") {
		t.Errorf("root script missing or changed:\n%s", script)
	}
}

func TestPropertiesSorted(t *testing.T) {
	walk(load(t), func(n xmlNode) {
		keys := make([]string, len(n.Properties))
		for i, p := range n.Properties {
			keys[i] = p.Key
		}
		if !slices.IsSorted(keys) {
			t.Errorf("%s properties not sorted: %v", n.Type, keys)
		}
	})
}
