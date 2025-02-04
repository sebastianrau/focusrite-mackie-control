package monitorcontroller

import (
	"fmt"

	"github.com/sebastianrau/focusrite-mackie-control/pkg/gomcu"
)

// MappingBool struct
type MappingBool struct {
	Value          bool
	McuButtonsList []gomcu.Switch
	FcIdsList      []FocusriteId
}

/*
func (m *MappingBool) SetBool(val bool) error {
	m.Value = val
	return nil
}

func (m *MappingBool) SetInt(val int) error {
	return fmt.Errorf("wrong datatype")
}

func (m *MappingBool) SetString(val string) error {
	return fmt.Errorf("wrong datatype")
}
*/

func (m *MappingBool) McuButtons() []gomcu.Switch {
	return m.McuButtonsList
}

func (m *MappingBool) FcIds() []FocusriteId {
	return m.FcIdsList
}

func (m *MappingBool) IsMcuID(id gomcu.Switch) bool {
	for _, v := range m.McuButtonsList {
		if v == id {
			return true
		}
	}
	return false
}

func (m *MappingBool) IsFcID(id FocusriteId) bool {
	for _, v := range m.FcIdsList {
		if v == id {
			return true
		}
	}
	return false
}

func (m *MappingBool) ValueString() string {
	return fmt.Sprintf("%t", m.Value)
}
