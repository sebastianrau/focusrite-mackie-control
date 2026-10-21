package protocol

import "strconv"

// Element is a leaf of the device tree with an ID that set messages address.
type Element interface {
	ElementID() int
	SetValue(value string) error
}

type ElementString struct {
	ID    int `xml:"id,attr,omitempty"`
	Value string
}

func (e *ElementString) ElementID() int { return e.ID }

func (e *ElementString) SetValue(v string) error {
	e.Value = v
	return nil
}

type ElementInt struct {
	ID    int `xml:"id,attr,omitempty"`
	Value int
}

func (e *ElementInt) ElementID() int { return e.ID }

func (e *ElementInt) SetValue(v string) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return err
	}
	e.Value = i
	return nil
}

type ElementBool struct {
	ID    int `xml:"id,attr,omitempty"`
	Value bool
}

func (e *ElementBool) ElementID() int { return e.ID }

func (e *ElementBool) SetValue(v string) error {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	e.Value = b
	return nil
}

type ElementFloat struct {
	ID    int `xml:"id,attr,omitempty"`
	Value float64
}

func (e *ElementFloat) ElementID() int { return e.ID }

func (e *ElementFloat) SetValue(v string) error {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return err
	}
	e.Value = f
	return nil
}
