package monitorcontroller

import (
	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

type MappingString struct {
	Value          string
	McuButtonsList []gomcu.Switch
	FcIdsList      []FocusriteId
}

/*
	func (m *MappingString) SetString(val string) error {
		m.Value = val
		return nil
	}

	func (m *MappingString) SetBool(val bool) error {
		return fmt.Errorf("wrong datatype")
	}

	func (m *MappingString) SetInt(val int) error {
		return fmt.Errorf("wrong datatype")
	}
*/

func (m *MappingString) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingString) FcIds() []FocusriteId {
	return m.FcIdsList
}

func (m *MappingString) IsMcuID(id gomcu.Switch) bool {
	for _, v := range m.McuButtonsList {
		if v == id {
			return true
		}
	}
	return false
}

func (m *MappingString) IsFcID(id FocusriteId) bool {
	for _, v := range m.FcIdsList {
		if v == id {
			return true
		}
	}
	return false
}

func (m *MappingString) ValueString() string {
	return m.Value
}
