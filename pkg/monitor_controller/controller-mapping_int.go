package monitorcontroller

import (
	"fmt"
	"slices"

	"github.com/sebastianrau/gomcu"
)

type MappingInt struct {
	Value          int
	McuButtonsList []gomcu.Switch
	FcId           FocusriteId
}

func (m *MappingInt) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingInt) IsMcuID(id gomcu.Switch) bool {
	return slices.Contains(m.McuButtonsList, id)
}

func (m *MappingInt) IsFcID(id FocusriteId) bool {
	return m.FcId == id
}

func (m *MappingInt) ValueString() string {
	return fmt.Sprintf("%d", m.Value)
}

func (m *MappingInt) GetFcID() FocusriteId {
	return m.FcId
}
