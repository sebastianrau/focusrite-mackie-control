package monitorcontroller

import (
	"fmt"
	"slices"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

// MappingBool struct
type MappingBool struct {
	Value          bool
	McuButtonsList []gomcu.Switch
	FcIdsList      []FocusriteId
}

func (m *MappingBool) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingBool) FcIds() []FocusriteId {
	return m.FcIdsList
}

func (m *MappingBool) IsMcuID(id gomcu.Switch) bool {
	return slices.Contains(m.McuButtonsList, id)
}

func (m *MappingBool) IsFcID(id FocusriteId) bool {
	return slices.Contains(m.FcIdsList, id)
}

func (m *MappingBool) ValueString() string {
	return fmt.Sprintf("%t", m.Value)
}
