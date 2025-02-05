package monitorcontroller

import (
	"fmt"
	"slices"

	"github.com/sebastianrau/gomcu"
)

// MappingBool struct
type MappingBool struct {
	Value          bool
	McuButtonsList []gomcu.Switch
	FcId           FocusriteId
}

func (m *MappingBool) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingBool) IsMcuID(id gomcu.Switch) bool {
	return slices.Contains(m.McuButtonsList, id)
}

func (m *MappingBool) IsFcID(id FocusriteId) bool {
	return m.FcId == id
}

func (m *MappingBool) ValueString() string {
	return fmt.Sprintf("%t", m.Value)
}

func (m *MappingBool) GetFcID() FocusriteId {
	return m.FcId
}
