package focusriteclient

import (
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-xml"
)

type DeviceList map[int]*focusritexml.Device

func (dl DeviceList) GetDevice(id int) (*focusritexml.Device, bool) {
	d, ok := dl[id]
	return d, ok
}

func (dl DeviceList) GetDeviceBySerialnumber(serial string) (*focusritexml.Device, bool) {
	for _, v := range dl {
		if v.SerialNumber == serial {
			return v, true
		}
	}
	return nil, false
}

func (dl DeviceList) AddDevice(d *focusritexml.Device) *focusritexml.Device {
	for _, v := range dl {
		if v.SerialNumber == d.SerialNumber {
			delete(dl, v.ID)
			log.Debugf("removed device with same serial from device list. Old Id  %d", v.ID)
		}
	}
	dl[d.ID] = d
	return dl[d.ID]
}

func (dl DeviceList) Remove(id int) {
	delete(dl, id)
	log.Debugf("removed device with ID: %d device list.", id)
}

func (dl DeviceList) UpdateSet(set focusritexml.Set) {
	d, ok := dl[set.DevID]
	if ok {
		d.UpdateSet(set)
	}
}
