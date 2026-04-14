package editor

import (
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/gridmap"
)

func (g *GameStateEditor) openTileMenu() {
    data := g.engine.GetData()
    menuItems := make([]services.MenuItem, len(data.Tiles()))
    for i, t := range data.Tiles() {
        tile := *t
        icon := tile.DefinedIcon
        if tile.Special == gridmap.SpecialTilePlayerSpawn {
            icon = '@'
        }
        menuItems[i] = services.MenuItem{
            Label: tile.Description(),
            Icon:  icon,
            Handler: g.setBrushHandlerWithLightUpdate(editMapUI, icon, func(pos geometry.Point) {
                g.placeTileAtPos(tile, pos)
            }),
        }
    }
    g.OpenTilePickerDropDown("Choose tile", menuItems)
    return
}

func (g *GameStateEditor) placeTileAtPos(tile gridmap.Tile, pos geometry.Point) {
    currentMap := g.engine.GetGame().GetMap()
    if currentMap.IsObjectAt(pos) || (!tile.IsWalkable && (currentMap.IsActorAt(pos) || currentMap.IsItemAt(pos))) {
        return
    }
    if tile.Special == gridmap.SpecialTilePlayerSpawn {
        currentMap.SetPlayerSpawn(pos)
        return
    }

    currentMap.SetTile(pos, tile)
}
