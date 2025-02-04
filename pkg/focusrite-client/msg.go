package focusriteclient

import (
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
)

type ApprovalMessasge bool
type ConnectionStatusMessage bool

type DeviceArrivalMessage focusritexml.Device
type DeviceRemovalMessage int

type DeviceUpdateMessage focusritexml.Device
type RawUpdateMessage focusritexml.Set
