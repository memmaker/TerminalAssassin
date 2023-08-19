package editor

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

func (g *GameStateEditor) openClothesMenu() {
	data := g.engine.GetData()
	allItems := data.Clothing()
	menuItems := make([]services.MenuItem, len(allItems)+1)
	menuItems[0] = services.MenuItem{
		Label: "Clear clothes",
		Handler: g.setBrushHandler(addClothesUI, 'X', func(pos geometry.Point) {
			g.removeItemAtPos(pos)
		}),
	}
	for index, i := range allItems {
		item := i
		menuItems[index+1] = services.MenuItem{
			Label:               item.Name,
			Icon:                core.GlyphClothing,
			IconForegroundColor: item.FgColor,
			Handler:             g.setBrushHandler(addClothesUI, core.GlyphClothing, func(pos geometry.Point) { g.placeClothesAt(*item, pos) }),
		}
	}
	g.OpenMenuBarDropDown("Choose clothes", (2*5)-2, menuItems)
}

func (g *GameStateEditor) placeClothesAt(clothing core.Clothing, position geometry.Point) {
	game := g.engine.GetGame()
	currentMap := game.GetMap()
	if currentMap.IsObjectAt(position) || !currentMap.IsTileWalkable(position) {
		return
	}
	if currentMap.IsActorAt(position) {
		actorAt := currentMap.ActorAt(position)
		actorAt.Clothes = clothing
		g.PrintAsMessage(fmt.Sprintf("Gave %s to %s", clothing.Name, actorAt.Name))
		return
	}
	game.SpawnClothingItem(position, clothing)
}
