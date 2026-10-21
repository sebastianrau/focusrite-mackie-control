package protocol

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ECUST-XX/xml"
)

// Device is the element tree of one interface as sent in a device-arrival.
type Device struct {
	XMLName xml.Name `xml:"device"`

	ID           int    `xml:"id,attr"`
	Protocol     string `xml:"protocol,attr"`
	Model        string `xml:"model,attr"`
	Class        string `xml:"class,attr"`
	BusID        string `xml:"bus-id,attr"`
	SerialNumber string `xml:"serial-number,attr"`
	Version      string `xml:"version,attr"`

	Nickname        ElementString `xml:"nickname"`
	SealBroken      ElementBool   `xml:"seal-broken"`
	Snapshot        ElementString `xml:"snapshot"`
	SaveSnapshot    ElementString `xml:"save-snapshot"`
	ResetDevice     ElementString `xml:"reset-device"`
	RecordOutputs   ElementString `xml:"record-outputs"`
	Dante           ElementString `xml:"dante"`
	State           ElementString `xml:"state"`
	PairableDevices ElementString `xml:"pairable-devices"`
	Preset          ElementString `xml:"preset"`

	Firmware     Firmware     `xml:"firmware"`
	Mixer        Mixer        `xml:"mixer"`
	Inputs       Inputs       `xml:"inputs"`
	Outputs      Outputs      `xml:"outputs"`
	Monitoring   Monitoring   `xml:"monitoring"`
	Clocking     Clocking     `xml:"clocking"`
	Settings     Settings     `xml:"settings"`
	QuickStart   QuickStart   `xml:"quick-start"`
	HaloSettings HaloSettings `xml:"halo-settings"`

	elements map[int]Element
}

// ApplySet writes the values of a set message into the device tree and
// returns the IDs it did not know. Keep a *Device: the lookup built on first
// use points into this struct, so a copy would update the original.
func (d *Device) ApplySet(set Set) (unknown []int, err error) {
	if d.elements == nil {
		d.elements = make(map[int]Element)
		indexElements(reflect.ValueOf(d).Elem(), d.elements)
	}
	var errs []error
	for _, it := range set.Items {
		if it.ID == 0 {
			continue // Focusrite Control sends some items without an element
		}
		e, ok := d.elements[it.ID]
		if !ok {
			unknown = append(unknown, it.ID)
			continue
		}
		if err := e.SetValue(it.Value); err != nil {
			errs = append(errs, fmt.Errorf("item %d=%q: %w", it.ID, it.Value, err))
		}
	}
	return unknown, errors.Join(errs...)
}

func indexElements(v reflect.Value, m map[int]Element) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			indexElements(v.Elem(), m)
		}
	case reflect.Struct:
		if e, ok := v.Addr().Interface().(Element); ok {
			if id := e.ElementID(); id != 0 {
				m[id] = e
			}
			return
		}
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				indexElements(v.Field(i), m)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			indexElements(v.Index(i), m)
		}
	}
}
