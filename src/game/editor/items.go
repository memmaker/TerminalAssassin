package editor

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

func (g *GameStateEditor) openItemMenu() {
	data := g.engine.GetData()
	allItems := data.Items()
	menuItems := make([]services.MenuItem, 0)
	menuItems = append(menuItems, services.MenuItem{
		Label: "Clear items",
		Handler: g.setBrushHandler(addItemsUI, 'X', func(pos geometry.Point) {
			g.removeItemAtPos(pos)
		}),
	})
	for _, i := range allItems {
		item := *i
		menuItems = append(menuItems, services.MenuItem{
			Label: item.Name,
			Icon:  item.DefinedIcon,
			Handler: g.setBrushHandler(addItemsUI, item.DefinedIcon, func(pos geometry.Point) {
				g.placeItemAtPos(item, pos)
			}),
		})
	}
	itemFactory := g.engine.GetItemFactory()
	complexItems := itemFactory.ComplexItems()
	for _, i := range complexItems {
		item := *i
		menuItems = append(menuItems, services.MenuItem{
			Label: item.Name,
			Icon:  item.DefinedIcon,
			Handler: g.setBrushHandler(addItemsUI, item.DefinedIcon, func(pos geometry.Point) {
				g.placeItemAtPos(item, pos)
			}),
		})
	}
	g.OpenMenuBarDropDown("Choose item", (2*3)-2, menuItems)
	return
}

func (g *GameStateEditor) removeItemAtPos(pos geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()
	if currentMap.IsActorAt(pos) {
		actorAt := currentMap.ActorAt(pos)
		actorAt.Inventory.Clear()
		actorAt.EquippedItem = nil
		g.PrintAsMessage("Cleared inventory of " + actorAt.Name)
	} else if currentMap.IsItemAt(pos) {
		itemAt := currentMap.ItemAt(pos)
		g.PrintAsMessage(fmt.Sprintf("Removed %s from map location %s", itemAt.Name, itemAt.Pos().String()))
		currentMap.RemoveItemAt(pos)
		g.SetDirty()
	}
}

func (g *GameStateEditor) removeSelectedItem() {
	if g.selectedItem == nil {
		return
	}
	holdingActor := g.selectedItem.HeldBy
	if holdingActor != nil {
		holdingActor.Inventory.RemoveItem(g.selectedItem)
		g.PrintAsMessage(fmt.Sprintf("Removed %s from %s's inventory", g.selectedItem.Name, holdingActor.Name))
	} else {
		currentMap := g.engine.GetGame().GetMap()
		currentMap.RemoveItem(g.selectedItem)
		g.PrintAsMessage(fmt.Sprintf("Removed %s from map location %s", g.selectedItem.Name, g.selectedItem.Pos().String()))
	}
	g.selectedItem = nil
	g.SetDirty()
	return
}
func (g *GameStateEditor) placeItemAtPos(item core.Item, pos geometry.Point) {
	game := g.engine.GetGame()
	currentMap := game.GetMap()
	placedItem := &item
	if !currentMap.IsTileWalkable(pos) || currentMap.IsObjectAt(pos) {
		return
	}
	if currentMap.IsActorAt(pos) {
		actorAt := currentMap.ActorAt(pos)
		actorAt.Inventory.AddItem(placedItem)
		placedItem.HeldBy = actorAt
		g.PrintAsMessage("Gave " + placedItem.Name + " to " + actorAt.Name)
	} else {
		currentMap.AddItem(placedItem, pos)
	}
}
func (g *GameStateEditor) selectItemAt(world geometry.Point) {
	itemString := "(No Item)"
	currentMap := g.engine.GetGame().GetMap()
	itemAt := currentMap.ItemAt(world)
	if itemAt == nil {
		return
	}
	g.selectedItem = itemAt
	itemString = services.EncodeItemAsString(itemAt)
	g.PrintAsMessage(fmt.Sprintf("Item: %s", itemString))
}
