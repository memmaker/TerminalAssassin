package core

import "strings"

type InventoryComponent struct {
	Items []*Item
}

func (i *InventoryComponent) AddItem(item *Item) {
	if item == nil {
		return
	}
	i.Items = append(i.Items, item)
}

func (i *InventoryComponent) RemoveItem(item *Item) {
	itemIndex := -1
	for j, it := range i.Items {
		if it == item {
			itemIndex = j
			break
		}
	}
	if itemIndex == -1 {
		return
	}
	item.HeldBy = nil
	i.Items = append(i.Items[:itemIndex], i.Items[itemIndex+1:]...)
}

func (i *InventoryComponent) Clear() {
	i.Items = make([]*Item, 0)
}

func (i *InventoryComponent) AsRunes() string {
	runes := make([]string, 0)
	for _, item := range i.Items {
		runes = append(runes, string(item.Icon()))
	}
	return strings.Join(runes, " ")
}

func (i *InventoryComponent) IsEmpty() bool {
	return len(i.Items) == 0
}
