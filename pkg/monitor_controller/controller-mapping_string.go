package monitorcontroller

import (
	"slices"

	"github.com/sebastianrau/gomcu"
)

type MappingString struct {
	Value          string
	McuButtonsList []gomcu.Switch
	FcId           FocusriteId
}

func (m *MappingString) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingString) IsMcuID(id gomcu.Switch) bool {
	return slices.Contains(m.McuButtonsList, id)
}

func (m *MappingString) IsFcID(id FocusriteId) bool {
	return m.FcId == id
}

func (m *MappingString) ValueString() string {
	return m.Value
}

func (m *MappingString) GetFcID() FocusriteId {
	return m.FcId
}
