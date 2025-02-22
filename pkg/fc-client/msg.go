package focusriteclient

import (
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-xml"
)

type ApprovalMessasge bool
type ConnectionStatusMessage bool

type DeviceArrivalMessage focusritexml.Device
type DeviceRemovalMessage int

type DeviceUpdateMessage focusritexml.Device
type RawUpdateMessage focusritexml.Set

/* Example implementation
go func() {
		for msg := range fc.FromFocusrite {
			switch msg.(type) {
			case focusriteclient.ApprovalMessasge:
			case focusriteclient.ConnectionStatusMessage:
			case focusriteclient.DeviceArrivalMessage:
			case focusriteclient.DeviceRemovalMessage:
			case focusriteclient.DeviceUpdateMessage:
			case focusriteclient.RawUpdateMessage:
			}
		}
	}()
*/
