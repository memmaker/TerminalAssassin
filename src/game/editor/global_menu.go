package editor

import (
	"fmt"
	"os"
	"path"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/gridmap"
)

func (g *GameStateEditor) openGlobalMenu() {
	menuItems := []services.MenuItem{
		{
			Label: "New Map",
			Handler: func() {
				g.ResizeAndClearMap(g.engine.MapWindowWidth(), g.engine.MapWindowHeight())
			},
		},
		{
			Label: "CImage from selection",
			Handler: func() {
				g.changeUIStateTo(editMapUI)
				g.placeThingIcon = 'i'
				g.selectionTool = NewFilledRectangleBrush()
				g.handler.CellsSelected = g.imageFromSelection
			},
		},
		{
			Label:   "Resize Map",
			Handler: g.resizeMap,
		},
		{
			Label:    "Load Map",
			Handler:  g.loadMap,
			QuickKey: "l",
		},
		{
			Label:    "Save Map",
			Handler:  g.saveMap,
			QuickKey: "s",
		},
		{
			Label:    "Quit Editor",
			Handler:  g.quitEditor,
			QuickKey: "q",
		},
	}
	g.OpenMenuBarDropDown("Global", (2*10)-2, menuItems)
}

func (g *GameStateEditor) resizeMap() {
	g.engine.GetUI().ShowTextInput("Size for new map ('w h' eg. '64 36'): ", "64 36", func(text string) {
		width, height := 64, 36
		fmt.Sscanf(text, "%d %d", &width, &height)
		g.ResizeAndClearMap(width, height)
		g.PrintAsMessage(fmt.Sprintf("New map created: %dx%d", width, height))
	}, func() {
		g.PrintAsMessage("cancelled")
	})
}

func (g *GameStateEditor) loadMap() {
	g.engine.GetUI().OpenMapsMenu(func(loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object]) {
		loadedMap.Apply(func(cell gridmap.MapCell[*core.Actor, *core.Item, services.Object]) gridmap.MapCell[*core.Actor, *core.Item, services.Object] {
			cell.IsExplored = true
			return cell
		})
		loadedMap.SetAmbientLight(common.GetAmbientLightFromDayTime(loadedMap.TimeOfDay).ToRGB())
		g.gridIsDirty = true
	})
}
func (g *GameStateEditor) saveMap() {
	currentMap := g.engine.GetGame().GetMap()
	career := g.engine.GetCareer()
	config := g.engine.GetGame().GetConfig()

	campaignFolderName := config.CampaignDirectory
	g.handler = UIHandler{Name: "enter map name", TextReceived: func(text string) {
		g.changeUIStateTo(editMapUI)
		if text != "" {
			folderPath := path.Join(campaignFolderName, career.CurrentCampaignFolder)
			mapFilePath := path.Join(folderPath, text+".map")
			os.MkdirAll(mapFilePath, 0755)
			//err := currentMap.SaveToDisk(mapFilePath)
			err := g.engine.SaveMap(currentMap, mapFilePath)
			if err == nil {
				g.PrintAsMessage("Map saved to " + mapFilePath)
			} else {
				g.PrintAsMessage("ERR: Failed to save map to " + mapFilePath + " (" + err.Error() + ")")
			}
		}
	}}
	g.showTextInput("Map name: ", "")
	return
}
