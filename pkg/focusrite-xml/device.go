package focusritexml

import (
	"fmt"
	"io"
	"log"

	"github.com/ECUST-XX/xml"
)

type Device struct {
	XMLName      xml.Name `xml:"device"`
	ID           int      `xml:"id,attr"`
	Protocol     string   `xml:"protocol,attr"`
	Model        string   `xml:"model,attr"`
	Class        string   `xml:"class,attr"`
	BusID        string   `xml:"bus-id,attr"`
	SerialNumber string   `xml:"serial-number,attr"`
	Version      string   `xml:"version,attr"`

	Nickname        ElementString `xml:"nickname"`
	SealBroken      ElementBool   `xml:"seal-broken"`
	Snapshot        ElementString `xml:"snapshot"`
	SaveSnapshot    ElementString `xml:"save-snapshot"`
	ResetDevice     ElementString `xml:"reset-device"`
	RecordOutputs   ElementString `xml:"record-outputs"`
	Dante           ElementString `xml:"dante"`
	State           ElementString `xml:"state"`
	PairableDevices ElementString `xml:"pairable-devices"`

	Preset       Preset       `xml:"preset"`
	Firmware     Firmware     `xml:"firmware"`
	Mixer        Mixer        `xml:"mixer"`
	Inputs       Inputs       `xml:"inputs"`
	Outputs      Outputs      `xml:"outputs"`
	Monitoring   Monitoring   `xml:"monitoring"`
	Clocking     Clocking     `xml:"clocking"`
	Settings     Settings     `xml:"settings"`
	QuickStart   QuickStart   `xml:"quick-start"`
	HaloSettings HaloSettings `xml:"halo-settings"`

	elementsMap map[int]Elements
}

func (d *Device) UpdateMap() {
	d.elementsMap = make(map[int]Elements)
	d.elementsMap[d.Nickname.ID] = &d.Nickname
	d.elementsMap[d.SealBroken.ID] = &d.SealBroken
	d.elementsMap[d.Snapshot.ID] = &d.Snapshot
	d.elementsMap[d.SaveSnapshot.ID] = &d.SaveSnapshot
	d.elementsMap[d.ResetDevice.ID] = &d.ResetDevice
	d.elementsMap[d.RecordOutputs.ID] = &d.RecordOutputs
	d.elementsMap[d.Dante.ID] = &d.Dante
	d.elementsMap[d.State.ID] = &d.State
	d.elementsMap[d.PairableDevices.ID] = &d.PairableDevices

	d.Firmware.UpdateMap(d.elementsMap)
	d.Firmware.UpdateMap(d.elementsMap)
	d.Mixer.UpdateMap(d.elementsMap)
	d.Inputs.UpdateMap(d.elementsMap)
	d.Outputs.UpdateMap(d.elementsMap)
	d.Clocking.UpdateMap(d.elementsMap)
	d.Settings.UpdateMap(d.elementsMap)
	d.QuickStart.UpdateMap(d.elementsMap)
	d.HaloSettings.UpdateMap(d.elementsMap)

	log.Printf("Updated Device Map with %d items", len(d.elementsMap))
}

func (d *Device) UpdateSet(set Set) {
	for _, v := range set.Items {
		value, ok := d.elementsMap[v.ID]
		if ok {
			value.Set(v.ID, v.Value)
		} else {
			log.Printf("unknown ID to update: %d with name %s\n", v.ID, v.Value)
		}
	}
}

func (d *Device) PrintMap(w io.Writer) {
	for k, e := range d.elementsMap {
		fmt.Printf("id: %d  - %v\n", k, e)
	}
}
