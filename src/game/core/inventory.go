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

// FindItemByType returns the first item of the given type, or nil.
func (i *InventoryComponent) FindItemByType(itemType ItemType) *Item {
	for _, it := range i.Items {
		if it.Type == itemType {
			return it
		}
	}
	return nil
}

// Contains reports whether item is currently held in this inventory.
func (i *InventoryComponent) Contains(item *Item) bool {
	for _, it := range i.Items {
		if it == item {
			return true
		}
	}
	return false
}

