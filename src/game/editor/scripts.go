package editor

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/geometry"
)

// goals:
// 1. Make it easier to add new locations
// 2. Make it easier to manage them
// 3. Allow to promote them to other kinds of locations (eg. starting point with clothes)

func (g *GameStateEditor) newNamedLocation(point geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()
	locationName := fmt.Sprintf("Location #%d (%d, %d)", len(currentMap.NamedLocations)+1, point.X, point.Y)
	currentMap.SetNamedLocation(locationName, point)
	g.changeUIStateTo(editNamedLocationUI)
	g.gridIsDirty = true
}
func (g *GameStateEditor) selectNamedLocation(point geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()
	g.selectedNamedLocation = currentMap.GetNamedLocationByPos(point)
	g.gridIsDirty = true
}
func (g *GameStateEditor) renameSelectedNamedLocation() {
	if g.selectedNamedLocation == "" {
		return
	}
	g.handler = UIHandler{Name: "Rename Location", TextReceived: func(content string) {
		currentMap := g.engine.GetGame().GetMap()
		currentMap.RenameLocation(g.selectedNamedLocation, content)
		g.changeUIStateTo(editNamedLocationUI)
	}}
	g.showTextInput("New name: ", "")
}

func (g *GameStateEditor) setNamedLocation(point geometry.Point) {
	if g.selectedNamedLocation == "" {
		return
	}
	currentMap := g.engine.GetGame().GetMap()
	currentMap.SetNamedLocation(g.selectedNamedLocation, point)
	g.changeUIStateTo(editNamedLocationUI)
	g.gridIsDirty = true
}

func (g *GameStateEditor) moveSelectedNamedLocation() {
	if g.selectedNamedLocation == "" {
		return
	}
	g.setBrushHandler(editNamedLocationUI, 'ç', g.setNamedLocation)()
}
func (g *GameStateEditor) deleteSelectedNamedLocation() {
	if g.selectedNamedLocation == "" {
		return
	}
	currentMap := g.engine.GetGame().GetMap()
	currentMap.RemoveNamedLocation(g.selectedNamedLocation)
	g.changeUIStateTo(editNamedLocationUI)
	g.gridIsDirty = true
}
