package monitorcontroller

import (
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/gomcu"
)

type FocusriteId int

type Mapping interface {
	ValueString() string
	McuButtons() []gomcu.Switch
	GetFcID() FocusriteId

	IsMcuID(id gomcu.Switch) bool
	IsFcID(id FocusriteId) bool
	ParseItem(item focusritexml.Item)
}

// MappingInt struct
