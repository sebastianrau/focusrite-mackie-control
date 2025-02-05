package monitorcontroller

import (
	"github.com/sebastianrau/gomcu"
)

type FocusriteId int

type Mapping interface {
	ValueString() string
	McuButtons() []gomcu.Switch
	GetFcID() FocusriteId

	IsMcuID(id gomcu.Switch) bool
	IsFcID(id FocusriteId) bool
}

// MappingInt struct
