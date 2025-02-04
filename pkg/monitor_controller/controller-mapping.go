package monitorcontroller

import (
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

type FocusriteId int

type Mapping interface {
	ValueString() string

	McuButtons() []gomcu.Switch
	FcIds() []FocusriteId

	IsMcuID(id gomcu.Switch) bool
	IsFcID(id FocusriteId) bool
}

// MappingInt struct
