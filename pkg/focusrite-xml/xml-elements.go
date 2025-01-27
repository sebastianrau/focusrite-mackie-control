package focusritexml

import (
	"strconv"
)

type Elements interface {
	Set(id int, value string) error
}

// ElementString
type ElementString struct {
	ID    int `xml:"id,attr,omitempty"`
	Value string
}

func (e *ElementString) Set(id int, value string) error {
	e.Value = value
	return nil
}

// ElementInt
type ElementInt struct {
	ID    int `xml:"id,attr,omitempty"`
	Value int
}

func (e *ElementInt) Set(id int, value string) error {
	i, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	e.Value = i
	return nil
}

// ElementBool
type ElementBool struct {
	ID    int `xml:"id,attr,omitempty"`
	Value bool
}

func (e *ElementBool) Set(id int, value string) error {
	i, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	e.Value = i
	return nil
}
