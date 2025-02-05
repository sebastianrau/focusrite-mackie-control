package monitorcontroller

import (
	"slices"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
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

func (m *MappingString) ParseItem(item focusritexml.Item) {
	if m.FcId != FocusriteId(item.ID) {
		return
	}
	m.Value = item.Value
}
