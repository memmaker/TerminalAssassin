package editor

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

func (g *GameStateEditor) addZone() {
	// let the user enter a name for the zone
	currentMap := g.engine.GetGame().GetMap()
	g.handler = UIHandler{Name: "name zone", TextReceived: func(name string) {
		if name == "" {
			return
		}
		newZone := gridmap.NewZone(name)
		currentMap.AddZone(newZone)
		g.SelectedZone = newZone
		//g.handler = addZonesUI
		g.setBrushHandler(addZonesUI, 'ç', g.setZoneAtPos)()
	}}
	g.showTextInput("Zone name: ", "")
}

func (g *GameStateEditor) openZoneMenu() {
	currentMap := g.engine.GetGame().GetMap()
	menuItems := make([]services.MenuItem, len(currentMap.ListOfZones))
	index := 0
	for _, z := range currentMap.ListOfZones {
		zone := z
		menuItems[index] = services.MenuItem{
			Label: zone.Name,
			Handler: func() {
				//g.handler = addZonesUI
				g.SelectedZone = zone
				g.setBrushHandler(addZonesUI, 'ç', g.setZoneAtPos)()
				g.PrintAsMessage(fmt.Sprintf("Selected zone %s", zone.Name))
			},
		}
		index++
	}
	g.OpenMenuBarDropDown("Choose zone", (2*6)-2, menuItems)
	return
}

func (g *GameStateEditor) openClothesMenuForZone() {
	data := g.engine.GetData()
	allItems := data.Clothing()
	menuItems := make([]services.MenuItem, 0)
	for _, i := range allItems {
		if g.SelectedZone.AllowedClothing.Contains(i.Name) {
			continue
		}
		item := i
		menuItem := services.MenuItem{
			Label: item.Name,
			Handler: func() {
				g.SelectedZone.AllowedClothing.Add(item.Name)
			},
		}
		menuItems = append(menuItems, menuItem)
	}
	g.OpenMenuBarDropDown("Add allowed clothes", (2*6)-2, menuItems)
}

func (g *GameStateEditor) setZoneAtPos(pos geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()
	if g.SelectedZone == nil {
		return
	}
	currentMap.SetZone(pos, g.SelectedZone)
}

func (g *GameStateEditor) toggleZoneType() {
	if g.SelectedZone == nil {
		return
	}
	g.SelectedZone.Type = (g.SelectedZone.Type + 1) % 4
	g.gridIsDirty = true
}
