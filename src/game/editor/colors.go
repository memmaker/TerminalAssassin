package editor

import (
    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
)

func (g *GameStateEditor) openColorMenu() {
    menuItems := []services.MenuItem{
        {
            Label:   "Foreground Color",
            Icon:    core.GlyphPalette,
            Handler: g.changeForegroundColor,
        },
        {
            Label:   "Background Color",
            Icon:    core.GlyphPalette,
            Handler: g.changeBackgroundColor,
        },
    }
    g.OpenMenuBarDropDown("Color", (2*12)-2, menuItems)
}

// ...existing code...

func (g *GameStateEditor) setBackgroundColorOfTileAt(point geometry.Point, color common.Color) {
    currentMap := g.engine.GetGame().GetMap()
    if !currentMap.Contains(point) {
        return
    }
    cellAt := currentMap.GetCell(point)
    cellAt.TileType = cellAt.TileType.WithBGColor(color)
    currentMap.SetCell(point, cellAt)

    if currentMap.IsObjectAt(point) {
        obj := currentMap.ObjectAt(point)
        newObjStyle := obj.Style(common.Style{
            Foreground: g.currentForegroundColor,
            Background: color,
        }).WithBg(color)
        obj.SetStyle(newObjStyle)
    }
}

func (g *GameStateEditor) setForegroundColorOfTileAt(point geometry.Point, color common.Color) {
    currentMap := g.engine.GetGame().GetMap()
    if !currentMap.Contains(point) {
        return
    }
    cellAt := currentMap.GetCell(point)
    cellAt.TileType = cellAt.TileType.WithFGColor(color)
    currentMap.SetCell(point, cellAt)

    if currentMap.IsObjectAt(point) {
        obj := currentMap.ObjectAt(point)
        newObjStyle := obj.Style(common.Style{
            Foreground: color,
            Background: g.currentBackgroundColor,
        }).WithFg(color)
        obj.SetStyle(newObjStyle)
    }
}

func (g *GameStateEditor) changeBackgroundColor() {
    g.bottomMessageLabel.Clear()

    userInterface := g.engine.GetUI()
    userInterface.HideWidget(g.menuBar)
    userInterface.HideWidget(g.topStatusLineLabel)
    userInterface.HideWidget(g.bottomMessageLabel)
    changed := func(color common.Color) {
        g.currentBackgroundColor = color
        g.gridIsDirty = true
    }
    closed := func(color common.Color) {
        changed(color)
        userInterface.ShowWidget(g.menuBar)
        userInterface.ShowWidget(g.topStatusLineLabel)
        userInterface.ShowWidget(g.bottomMessageLabel)
        g.updateStatusLine()
        g.clearHUD = true
    }
    userInterface.OpenColorPicker(g.currentBackgroundColor, changed, closed)
}

func (g *GameStateEditor) changeForegroundColor() {
    g.bottomMessageLabel.Clear()

    userInterface := g.engine.GetUI()
    userInterface.HideWidget(g.menuBar)
    userInterface.HideWidget(g.topStatusLineLabel)
    userInterface.HideWidget(g.bottomMessageLabel)
    changed := func(color common.Color) {
        g.currentForegroundColor = color
        g.gridIsDirty = true
    }
    closed := func(color common.Color) {
        changed(color)
        userInterface.ShowWidget(g.menuBar)
        userInterface.ShowWidget(g.topStatusLineLabel)
        userInterface.ShowWidget(g.bottomMessageLabel)
        g.updateStatusLine()
        g.clearHUD = true
    }
    userInterface.OpenColorPicker(g.currentForegroundColor, changed, closed)
}
