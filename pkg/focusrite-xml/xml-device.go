package focusritexml

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/ECUST-XX/xml"
)

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

	elementsMap map[int]Elements
}

func (d *Device) UpdateMap() {
	if d.elementsMap == nil {
		d.elementsMap = make(map[int]Elements)
	}
	UpdateAllMaps(d, d.elementsMap, 0)
	log.Printf("Updated Device Map with %d items", len(d.elementsMap))
}

func (d *Device) UpdateSet(set Set) int {
	updateCount := 0
	for _, v := range set.Items {
		value, ok := d.elementsMap[v.ID]
		if ok {
			updateCount++
			value.Set(v.ID, v.Value)
		} else {
			log.Printf("unknown ID to update: %d with name %s\n", v.ID, v.Value)
		}
	}
	return updateCount
}

func UpdateAllMaps(v interface{}, elementsMap map[int]Elements, level int) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch e := v.(type) {
	case ElementBool:
		elementsMap[e.Id()] = &e
		return

	case ElementInt:
		elementsMap[e.Id()] = &e
		return

	case ElementString:
		elementsMap[e.Id()] = &e
		return
	}

	// Wenn es kein Element ist, durch alle Felder der Struktur iterieren
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() == reflect.Struct || (field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct) {
			UpdateAllMaps(field.Interface(), elementsMap, level+1)
		}

		if field.Kind() == reflect.Array || field.Kind() == reflect.Slice {
			for j := 0; j < field.Len(); j++ {
				UpdateAllMaps(field.Index(j).Interface(), elementsMap, level+1)
			}
		}
	}
}

func IndentPrintf(indent int, format string, a ...interface{}) string {
	return strings.Repeat(" ", indent*4) + fmt.Sprintf(format, a...)
}

/*

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
	d.elementsMap[d.Preset.ID] = &d.Preset

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
*/
