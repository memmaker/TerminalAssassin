package editor

import (
    "fmt"

    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/objects"
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
    g.PrintAsMessage(fmt.Sprintf("Zone '%s' type: %s", g.SelectedZone.Name, g.SelectedZone.Type.ToString()))
}

// autoZone puts the editor into a one-shot click mode.
// The clicked tile becomes the seed for the public-space flood fill.
func (g *GameStateEditor) autoZone() {
    g.changeUIStateTo(editMapUI)
    g.PrintAsMessage("Auto-Zone: click to set starting coordinates for public space")
    g.handler.CellsSelected = func() {
        if len(g.selectedWorldPositions) == 0 {
            return
        }
        startPos := g.selectedWorldPositions[0]
        g.runAutoZone(startPos)
        g.changeUIStateTo(editMapUI)
    }
}

// runAutoZone clears all existing zones and rebuilds them automatically:
//   - One public zone, flood-filled from startPos (stopping at walls, doors and windows).
//   - One "private_NN" zone per connected region of walkable tiles not in the public zone.
func (g *GameStateEditor) runAutoZone(startPos geometry.Point) {
    currentMap := g.engine.GetGame().GetMap()

    // Clear all existing zones and zone assignments.
    currentMap.ListOfZones = make([]*gridmap.ZoneInfo, 0)
    for i := range currentMap.ZoneMap {
        currentMap.ZoneMap[i] = nil
    }

    // --- Public zone ---
    publicZone := gridmap.NewPublicZone(gridmap.PublicZoneName)
    currentMap.AddZone(publicZone)
    for _, pos := range g.bfsPublicZone(currentMap, startPos) {
        currentMap.SetZone(pos, publicZone)
    }

    // --- Private zones ---
    // Iterate every tile; when we find a walkable, still-unassigned tile, grow
    // a new private zone from it.
    privateCount := 0
    for y := 0; y < currentMap.MapHeight; y++ {
        for x := 0; x < currentMap.MapWidth; x++ {
            pos := geometry.Point{X: x, Y: y}
            if !currentMap.IsTileWalkable(pos) || currentMap.ZoneAt(pos) != nil {
                continue
            }
            privateCount++
            zoneName := fmt.Sprintf("private_%02d", privateCount)
            privateZone := gridmap.NewZone(zoneName)
            privateZone.Type = gridmap.ZoneTypePrivate
            currentMap.AddZone(privateZone)
            for _, p := range g.bfsPrivateZone(currentMap, pos) {
                currentMap.SetZone(p, privateZone)
            }
        }
    }

    g.gridIsDirty = true
    g.PrintAsMessage(fmt.Sprintf("Auto-Zone complete: 1 public zone, %d private zone(s)", privateCount))
}

// bfsPublicZone returns all tiles reachable from start without crossing
// walls, door objects, or window objects.
func (g *GameStateEditor) bfsPublicZone(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], start geometry.Point) []geometry.Point {
    if !currentMap.Contains(start) || !currentMap.IsTileWalkable(start) {
        return nil
    }
    visited := make(map[geometry.Point]bool)
    queue := []geometry.Point{start}
    visited[start] = true
    var result []geometry.Point

    for len(queue) > 0 {
        pos := queue[0]
        queue = queue[1:]
        result = append(result, pos)

        for _, nb := range currentMap.GetAllCardinalNeighbors(pos) {
            if visited[nb] {
                continue
            }
            visited[nb] = true
            if !currentMap.IsTileWalkable(nb) {
                continue
            }
            // Treat any tile that holds a door or window as a boundary.
            if currentMap.IsObjectAt(nb) && g.isDoorOrWindow(currentMap.ObjectAt(nb)) {
                continue
            }
            queue = append(queue, nb)
        }
    }
    return result
}

// bfsPrivateZone returns all tiles that belong to this private zone:
// walkable unassigned tiles reachable from start, plus any unassigned
// non-walkable (wall) tiles that directly border them. Wall tiles are
// collected but never enqueued, so the BFS never expands through walls.
func (g *GameStateEditor) bfsPrivateZone(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], start geometry.Point) []geometry.Point {
    visited := make(map[geometry.Point]bool)
    queue := []geometry.Point{start}
    visited[start] = true
    var result []geometry.Point

    for len(queue) > 0 {
        pos := queue[0]
        queue = queue[1:]
        result = append(result, pos)

        for _, nb := range currentMap.GetAllCardinalNeighbors(pos) {
            if visited[nb] {
                continue
            }
            visited[nb] = true
            // Stop at tiles already assigned to any zone (includes public space).
            if currentMap.ZoneAt(nb) != nil {
                continue
            }
            if !currentMap.IsTileWalkable(nb) {
                // Wall tile: claim it for this zone but don't expand through it.
                result = append(result, nb)
            }
            queue = append(queue, nb)
        }
    }
    return result
}

// isDoorOrWindow reports whether the given map object is a Door or Window.
func (g *GameStateEditor) isDoorOrWindow(obj services.Object) bool {
    _, isDoor := obj.(*objects.Door)
    _, isWindow := obj.(*objects.Window)
    return isDoor || isWindow
}
