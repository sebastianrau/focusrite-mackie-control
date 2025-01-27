package focusrite

import (
	"log"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
)

type DeviceList map[int]*focusritexml.Device

func (dl DeviceList) GetDevice(id int) (*focusritexml.Device, bool) {
	d, ok := dl[id]
	return d, ok
}

func (dl DeviceList) AddDevice(d *focusritexml.Device) {
	for _, v := range dl {
		if v.SerialNumber == d.SerialNumber {
			delete(dl, v.ID)
			log.Printf("removed device with same serial from device list. Old Id  %d", v.ID)
		}
	}
	dl[d.ID] = d
	d.UpdateMap()
}

func (dl DeviceList) Remove(id int) {
	delete(dl, id)
	log.Printf("removed device with ID: %d device list.", id)
}

func (dl DeviceList) UpdateSet(set focusritexml.Set) {
	d, ok := dl[set.DevID]
	if ok {
		d.UpdateSet(set)
	}
}
