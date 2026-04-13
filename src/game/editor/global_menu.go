package editor

import (
    "fmt"
    "os"
    "path"
    "path/filepath"
    "strings"

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
            Label:   "Ambient Light",
            Handler: g.changeAmbientLight,
        },
        {
            Label:   "Set default style",
            Handler: g.setDefaultStyleFromCurrent,
        },
        {
            Label:   "Apply default style",
            Handler: g.applyDefaultStyleToWholeMap,
        },
        {
            Label:   "Auto-Zone",
            Handler: g.autoZone,
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
        g.currentForegroundColor = loadedMap.DefaultStyle.Foreground
        g.currentBackgroundColor = loadedMap.DefaultStyle.Background
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
                currentMap.MetaData.FileName = mapFilePath
                g.PrintAsMessage("Map saved to " + mapFilePath)
            } else {
                g.PrintAsMessage("ERR: Failed to save map to " + mapFilePath + " (" + err.Error() + ")")
            }
        }
    }}
    prefilledName := ""
    if existingFileName := currentMap.MapFileName(); existingFileName != "" {
        prefilledName = strings.TrimSuffix(filepath.Base(existingFileName), ".map")
    }
    g.showTextInput("Map name: ", prefilledName)
    return
}

// setDefaultStyleFromCurrent records the editor's current fg/bg palette as the
// map's DefaultStyle without touching any existing tile or object colours.
func (g *GameStateEditor) setDefaultStyleFromCurrent() {
    currentMap := g.engine.GetGame().GetMap()
    currentMap.DefaultStyle = common.Style{
        Foreground: g.currentForegroundColor,
        Background: g.currentBackgroundColor,
    }
    g.PrintAsMessage(fmt.Sprintf("Map default style set (fg: %v  bg: %v)", g.currentForegroundColor, g.currentBackgroundColor))
}

// applyDefaultStyleToWholeMap re-colours every tile and object on the map using
// the map's DefaultStyle. Non-walkable (wall) tiles receive the reversed style.
func (g *GameStateEditor) applyDefaultStyleToWholeMap() {
    currentMap := g.engine.GetGame().GetMap()
    defaultStyle := currentMap.DefaultStyle

    currentMap.Apply(func(cell gridmap.MapCell[*core.Actor, *core.Item, services.Object]) gridmap.MapCell[*core.Actor, *core.Item, services.Object] {
        cell.TileType = cell.TileType.WithBGColor(defaultStyle.Background).WithFGColor(defaultStyle.Foreground)
        return cell
    })

    for _, obj := range currentMap.Objects() {
        obj.SetStyle(defaultStyle)
    }

    g.gridIsDirty = true
    g.PrintAsMessage("Applied default style to whole map")
}
