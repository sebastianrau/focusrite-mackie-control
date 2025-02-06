package monitorcontroller

import (
	"fmt"
	"slices"
	"strconv"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
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

func (m *MappingInt) ParseItem(item focusritexml.Item) {
	if m.FcId != FocusriteId(item.ID) {
		return
	}
	intValue, err := strconv.Atoi(item.Value)
	if err != nil {
		log.Error(err.Error())
		return
	}
	m.Value = int(intValue)
}
