package monitorcontroller

import (
	"fmt"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

type MappingInt struct {
	Value          int
	McuButtonsList []gomcu.Switch
	FcIdsList      []FocusriteId
}

/*
	func (m *MappingInt) SetInt(val int) error {
		m.Value = val
		return nil
	}

	func (m *MappingInt) SetBool(val bool) error {
		return fmt.Errorf("wrong datatype")
	}

	func (m *MappingInt) SetString(val string) error {
		return fmt.Errorf("wrong datatype")
	}
*/
func (m *MappingInt) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingInt) FcIds() []FocusriteId {
	return m.FcIdsList
}

func (m *MappingInt) IsMcuID(id gomcu.Switch) bool {
	for _, v := range m.McuButtonsList {
		if v == id {
			return true
		}
	}
	return false
}

func (m *MappingInt) IsFcID(id FocusriteId) bool {
	for _, v := range m.FcIdsList {
		if v == id {
			return true
		}
	}
	return false
}

func (m *MappingInt) ValueString() string {
	return fmt.Sprintf("%d", m.Value)
}
