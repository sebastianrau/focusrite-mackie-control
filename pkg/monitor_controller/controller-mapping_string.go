package monitorcontroller

import (
	"slices"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

type MappingString struct {
	Value          string
	McuButtonsList []gomcu.Switch
	FcIdsList      []FocusriteId
}

func (m *MappingString) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingString) FcIds() []FocusriteId {
	return m.FcIdsList
}

func (m *MappingString) IsMcuID(id gomcu.Switch) bool {
	return slices.Contains(m.McuButtonsList, id)
}

func (m *MappingString) IsFcID(id FocusriteId) bool {
	return slices.Contains(m.FcIdsList, id)
}

func (m *MappingString) ValueString() string {
	return m.Value
}
