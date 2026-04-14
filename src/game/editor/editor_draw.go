package editor

import (
    "github.com/memmaker/terminal-assassin/game/services"
    "strconv"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/console"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/gridmap"
)

func (g *GameStateEditor) Draw(con console.CellInterface) {
    if g.clearHalfWidth {
        con.HalfWidthTransparent()
        g.clearHalfWidth = false
    }

    if !g.gridIsDirty {
        return
    }

    if g.clearHUD {
        g.clearHUDArea(con)
        g.clearHUD = false
    }
    m := g.engine.GetGame()
    currentMap := m.GetMap()

    currentMap.IterWindow(m.GetCamera().ViewPort, func(worldPos geometry.Point, c gridmap.MapCell[*core.Actor, *core.Item, services.Object]) {
        screenPos := m.GetCamera().WorldToScreen(worldPos)
        icon, style := m.DrawWorldAtPosition(worldPos, c)
        style = m.ApplyLighting(worldPos, c, style)

        // In the editor, buried items are rendered with a dirt-brown background
        // so the designer can see and select them. This is done inline so no
        // intermediate rendering step can produce an unexpected colour first.
        if currentMap.IsItemAt(worldPos) {
            if item := currentMap.ItemAt(worldPos); item.Buried {
                icon = item.Icon()
                if g.selectedItem == item {
                    style = style.WithBg(core.CurrentTheme.MarkedBackground)
                } else {
                    style = style.WithBg(core.CurrentTheme.EditorBuriedItemBackground)
                }
            }
        }

        if g.LastSelectedPos == worldPos {
            style = style.WithBg(core.CurrentTheme.MarkedBackground)
        }
        if g.SelectedZone != nil {
            zoneAt := currentMap.ZoneAt(worldPos)
            if zoneAt != nil && zoneAt.Name == g.SelectedZone.Name {
                var zoneColor common.Color
                if zoneAt.IsDropOff() {
                    zoneColor = core.CurrentTheme.ZoneDropOffBackground
                } else if zoneAt.IsHighSecurity() {
                    zoneColor = core.CurrentTheme.ZoneHighSecurityBackground
                } else if zoneAt.IsPublic() {
                    zoneColor = core.CurrentTheme.ZonePublicBackground
                } else if zoneAt.IsPrivate() {
                    zoneColor = core.CurrentTheme.ZonePrivateBackground
                }
                style = style.WithBg(zoneColor)
            }
        }

        con.SetSquare(screenPos, common.Cell{Rune: icon, Style: style})
    })
    currentMap.IterAllLights(func(p geometry.Point, l *gridmap.LightSource) {
        if !m.GetCamera().ViewPort.Contains(p) {
            return
        }
        screenPos := m.GetCamera().WorldToScreen(p)
        cellAt := con.AtSquare(screenPos)
        style := cellAt.Style
        if g.SelectedLightSource == l {
            style = style.Reversed()
        }
        con.SetSquare(screenPos, common.Cell{Rune: '*', Style: style})
    })

    if m.GetCamera().ViewPort.Contains(currentMap.PlayerSpawn) {
        con.SetSquare(m.GetCamera().WorldToScreen(currentMap.PlayerSpawn), common.Cell{Rune: '@', Style: common.Style{Foreground: common.Black, Background: core.CurrentTheme.EditorSpawnBackground}})
    }

    if g.SelectedActor != nil {
        g.SelectedActor.VisionCone(func(worldPos geometry.Point) {
            if !m.GetCamera().ViewPort.Contains(worldPos) {
                return
            }
            screenPos := m.GetCamera().WorldToScreen(worldPos)
            cellAt := con.AtSquare(screenPos)
            con.SetSquare(screenPos, cellAt.WithBackgroundColor(core.CurrentTheme.EditorVisionConeBackground))
        })
    }

    // Draw a green vision-cone preview for the currently selected task whenever
    // it has a look direction assigned (including live updates during mouse-aim).
    // The FOV (shadow-casting) was already computed in recomputeTaskPreviewFov
    // at selection time; here we only apply the cheap cone-angle filter.
    if g.taskPreviewFov != nil &&
        g.SelectedSchedule != nil &&
        g.SelectedTaskIndex >= 0 &&
        g.SelectedTaskIndex < len(g.SelectedSchedule.Tasks) {
        task := g.SelectedSchedule.Tasks[g.SelectedTaskIndex]
        const fovDegrees = 90.0
        visionRange := currentMap.MaxVisionRange
        rangeSquared := visionRange * visionRange
        drawCone := func(dir float64) {
            left, right := geometry.GetLeftAndRightBorderOfVisionCone(task.Location, dir, fovDegrees)
            g.taskPreviewFov.IterSSC(func(p geometry.Point) {
                if geometry.DistanceSquared(task.Location, p) > rangeSquared {
                    return
                }
                if !geometry.InVisionCone(task.Location, p, left, right) {
                    return
                }
                if !m.GetCamera().ViewPort.Contains(p) {
                    return
                }
                screenPos := m.GetCamera().WorldToScreen(p)
                cellAt := con.AtSquare(screenPos)
                con.SetSquare(screenPos, cellAt.WithBackgroundColor(core.CurrentTheme.EditorVisionConeBackground))
            })
        }
        for _, dir := range task.LookDirections {
            drawCone(dir)
        }
        if g.pendingLookDir >= 0 {
            drawCone(g.pendingLookDir)
        }
    }

    if g.SelectedSchedule != nil && len(g.SelectedSchedule.Tasks) > 0 {
        g.drawTasks(con, g.SelectedSchedule)
    }
    if len(g.selectedWorldPositions) > 0 {
        for _, p := range g.selectedWorldPositions {
            if !m.GetCamera().ViewPort.Contains(p) {
                continue
            }
            screenPos := m.GetCamera().WorldToScreen(p)
            cellAt := con.AtSquare(screenPos)
            con.SetSquare(screenPos, cellAt.WithBackgroundColor(core.CurrentTheme.MarkedBackground))
        }
    }
    for name, location := range currentMap.NamedLocations {
        if !m.GetCamera().ViewPort.Contains(location) {
            continue
        }
        screenPos := m.GetCamera().WorldToScreen(location)
        cell := con.AtSquare(screenPos)
        colorForLocation := core.CurrentTheme.MapBackgroundLight
        if g.selectedNamedLocation != "" && g.selectedNamedLocation == name {
            colorForLocation = core.CurrentTheme.MarkedBackground
        }
        con.SetSquare(screenPos, cell.WithBackgroundColor(colorForLocation))
    }

    if g.currentPrefab != nil {
        g.currentPrefab.Draw(con, g.MousePositionOnScreen)
    }

    cellAtMouse := con.AtSquare(g.MousePositionOnScreen)
    con.SetSquare(g.MousePositionOnScreen, cellAtMouse.WithStyle(cellAtMouse.Style.Reversed()))

    g.gridIsDirty = false
}

func (g *GameStateEditor) clearHUDArea(con console.CellInterface) int {
    mapHeight := g.engine.MapWindowHeight()
    gridWidth, gridHeight := g.engine.ScreenGridWidth(), g.engine.ScreenGridHeight()
    con.SquareFill(geometry.NewRect(0, mapHeight, gridWidth, gridHeight), common.Cell{Rune: ' ', Style: common.DefaultStyle})
    return mapHeight
}

func (g *GameStateEditor) drawTasks(grid console.CellInterface, schedule *gridmap.Schedule) {
    camera := g.engine.GetGame().GetCamera()
    mapHeight := g.engine.MapWindowHeight()
    var selectedPos geometry.Point
    if g.SelectedTaskIndex > -1 && g.SelectedTaskIndex < len(schedule.Tasks) {
        selectedPos = schedule.Tasks[g.SelectedTaskIndex].Location
    }
    for i, task := range schedule.Tasks {
        itoa := strconv.Itoa(i + 1)
        taskPosOnScreen := camera.WorldToScreen(task.Location)
        if taskPosOnScreen.Y >= 0 && taskPosOnScreen.Y < mapHeight {
            grid.SetSquare(taskPosOnScreen, common.Cell{Rune: []rune(itoa)[0], Style: common.Style{Foreground: core.CurrentTheme.EditorTaskNumberForeground, Background: core.CurrentTheme.SelectionBackground}})
        }
        for _, pos := range task.KnownPath {
            if !camera.ViewPort.Contains(pos) {
                continue
            }
            screenPos := camera.WorldToScreen(pos)
            cellAtPos := grid.AtSquare(screenPos)
            if selectedPos == pos {
                cellAtPos.Style.Background = core.CurrentTheme.MarkedBackground
            } else {
                cellAtPos.Style.Background = core.CurrentTheme.HUDWarningBackground
            }

            grid.SetSquare(screenPos, cellAtPos)
        }
    }
}
