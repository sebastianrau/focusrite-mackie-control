package monitorcontroller

import (
	"fmt"
	"slices"

	"github.com/sebastianrau/gomcu"
)

type MappingInt struct {
	Value          int
	McuButtonsList []gomcu.Switch
	FcIdsList      []FocusriteId
}

func (m *MappingInt) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingInt) FcIds() []FocusriteId {
	return m.FcIdsList
}

func (m *MappingInt) IsMcuID(id gomcu.Switch) bool {
	return slices.Contains(m.McuButtonsList, id)
}

func (m *MappingInt) IsFcID(id FocusriteId) bool {
	return slices.Contains(m.FcIdsList, id)
}

func (m *MappingInt) ValueString() string {
	return fmt.Sprintf("%d", m.Value)
}
