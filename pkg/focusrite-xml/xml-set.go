package focusritexml

import (
	"encoding/xml"
	"fmt"
)

type Set struct {
	XMLName xml.Name `xml:"set"`
	DevID   int      `xml:"devid,attr"`
	Items   []Item   `xml:"item"`
}

type Item struct {
	ID    int    `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

func NewSet(deviceId int) *Set {
	return &Set{
		DevID: deviceId,
		Items: make([]Item, 0),
	}
}

func (s *Set) AddItem(i Item) *Set {
	if i.ID != 0 {
		s.Items = append(s.Items, i)
	}
	return s
}

func (s *Set) AddItemBool(itemId int, value bool) {
	s.AddItem(Item{ID: itemId, Value: fmt.Sprintf("%t", value)})
}

func (s *Set) AddItemInt(itemId int, value int) {
	s.AddItem(Item{ID: itemId, Value: fmt.Sprintf("%d", value)})
}

func (s *Set) AddItemString(itemId int, value string) {
	s.AddItem(Item{ID: itemId, Value: value})
}

func (s *Set) AddItemsBool(itemIds []int, value bool) {
	for _, i := range itemIds {
		s.AddItemBool(i, value)
	}
}

func (s *Set) AddItemsInt(itemIds []int, value int) {
	for _, i := range itemIds {
		s.AddItemInt(i, value)

	}
}

func (s *Set) AddItemsString(itemIds []int, value string) {
	for _, i := range itemIds {
		s.AddItemString(i, value)

	}
}
