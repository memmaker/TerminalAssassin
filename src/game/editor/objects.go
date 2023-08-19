package editor

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

func (g *GameStateEditor) openObjectsMenu() {
	currentMap := g.engine.GetGame().GetMap()
	data := g.engine.GetData()
	defaultFloor := data.GroundTile()
	objectFactory := g.engine.GetObjectFactory()
	menuItems := []services.MenuItem{
		{
			Label: "clear object",
			Handler: g.setBrushHandlerWithLightUpdate(addObjectsUI, 'X', func(pos geometry.Point) {
				currentMap.RemoveObjectAt(pos)
			}),
		},
	}
	for _, o := range objectFactory.SimpleObjects() {
		objectCreator := o
		identifier := objectCreator.Name
		menuItems = append(menuItems, services.MenuItem{
			Label: identifier,
			Icon:  objectCreator.Icon,
			Handler: g.setBrushHandlerWithLightUpdate(addObjectsUI, 'O', func(pos geometry.Point) {
				newObject := objectCreator.Create(identifier)
				newObject.SetStyle(common.Style{Foreground: g.currentForegroundColor, Background: g.currentBackgroundColor})
				currentMap.AddObject(newObject, pos)
				currentMap.SetTile(pos, defaultFloor.WithBGColor(g.currentBackgroundColor).WithFGColor(g.currentForegroundColor))
			}),
		})
	}

	g.OpenMenuBarDropDown("Choose object", (2*4)-2, menuItems)
	return
}
func (g *GameStateEditor) selectObjectAt(pos geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()
	objectAt := currentMap.ObjectAt(pos)
	if objectAt != nil {
		infoText := objectAt.Description()
		if keyed, ok := objectAt.(services.KeyBound); ok {
			infoText = fmt.Sprintf("%s (%s)", objectAt.Description(), keyed.GetKey())
		}
		g.selectedObject = objectAt
		g.PrintAsMessage(fmt.Sprintf("Object: %s", infoText))
	} else {
		g.PrintAsMessage("No object at " + pos.String())
	}
}

func (g *GameStateEditor) removeSelectedObject() {
	if g.selectedObject == nil {
		return
	}
	currentMap := g.engine.GetGame().GetMap()
	currentMap.RemoveObjectAt(g.selectedObject.Pos())
	g.selectedObject = nil
	g.PrintAsMessage(fmt.Sprintf("Removed object %s at %s", g.selectedObject.Description(), g.selectedObject.Pos().String()))
	g.SetDirty()
}
