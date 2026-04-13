package editor

import (
    "fmt"
    "github.com/hajimehoshi/ebiten/v2"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/gridmap"
    "github.com/memmaker/terminal-assassin/utils"
)

func (g *GameStateEditor) switchToGamePlay() {
    g.bottomMessageLabel = nil
    g.topStatusLineLabel = nil
    userInterface := g.engine.GetUI()
    userInterface.OpenMapsMenu(func(loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object]) {
        g.engine.GetGame().PushGameplayState()
    })
}

func (g *GameStateEditor) quitEditor() {
    g.bottomMessageLabel = nil
    g.topStatusLineLabel = nil
    g.engine.GetGame().PopAndInitPrevious()
}

func (g *GameStateEditor) PrintAsMessage(text string) {
    g.bottomMessageLabel.SetText(text)
}
func (g *GameStateEditor) changeUIStateFunc(state UIHandler) func() {
    return func() {
        g.changeUIStateTo(state)
    }
}

func (g *GameStateEditor) changeUIStateTo(state UIHandler) {
    g.handler = state
    g.menuBar.SetContextMenu(state.ContextMenu)
    //g.engine.LastMenuIndex = -1 // TODO: re-enable menus with memory
    g.PrintAsMessage("edit mode -> " + state.Name)
    g.gridIsDirty = true
    return
}

func (g *GameStateEditor) setBrushHandler(state UIHandler, placeIcon rune, placeFunc func(point geometry.Point)) func() {
    return func() {
        g.changeUIStateTo(state)
        g.placeThingIcon = placeIcon
        g.handler.CellsSelected = func() {
            g.callOnSelection(placeFunc)
        }
        //g.menuBar.SetDirty()
        g.updateStatusLine()
    }
}

func (g *GameStateEditor) callOnSelection(placeFunc func(worldPos geometry.Point)) {
    for _, worldPos := range g.selectedWorldPositions {
        placeFunc(worldPos)
    }
}
func (g *GameStateEditor) callOnSelectionFunc(placeFunc func(worldPos geometry.Point)) func() {
    return func() {
        g.callOnSelection(placeFunc)
    }
}

func (g *GameStateEditor) isState(uiState UIHandler) func() bool {
    return func() bool {
        return g.handler.Name == uiState.Name
    }
}

func (g *GameStateEditor) showTextInput(prompt string, prefilled string) {
    onComplete := func(string) {}
    if g.handler.TextReceived != nil {
        onComplete = func(s string) {
            g.gridIsDirty = true
            g.handler.TextReceived(s)
        }
    }
    onAbort := func() {
        g.gridIsDirty = true
    }
    userInterface := g.engine.GetUI()

    userInterface.ShowTextInput(prompt, prefilled, onComplete, onAbort)
}

func (g *GameStateEditor) selectAtMousePos() {
    currentMap := g.engine.GetGame().GetMap()
    pos := g.MousePositionInWorld
    if currentMap.IsActorAt(pos) {
        g.changeUIStateTo(editActorUI)
        g.selectActorAt(pos)
    } else if g.SelectedSchedule != nil && g.scheduleHasTaskAt(pos) {
        g.changeUIStateTo(editTaskUI)
        g.selectTaskAt(pos)
    } else if currentMap.IsObjectAt(pos) {
        g.changeUIStateTo(addObjectsUI)
        g.selectObjectAt(pos)
    } else if currentMap.IsItemAt(pos) {
        g.changeUIStateTo(addItemsUI)
        g.selectItemAt(pos)
    } else if currentMap.IsBakedLightSource(pos) {
        g.changeUIStateTo(editLightsUI)
        g.selectLightAt(pos)
    } else if currentMap.IsNamedLocationAt(pos) {
        g.changeUIStateTo(editNamedLocationUI)
        g.selectedNamedLocation = currentMap.GetNamedLocationByPos(pos)
        g.PrintAsMessage("edit named location -> " + g.selectedNamedLocation)
    }
    return
}

// scheduleHasTaskAt reports whether the currently selected schedule has a task at pos.
func (g *GameStateEditor) scheduleHasTaskAt(pos geometry.Point) bool {
    if g.SelectedSchedule == nil {
        return false
    }
    for _, task := range g.SelectedSchedule.Tasks {
        if task.Location == pos {
            return true
        }
    }
    return false
}

func (g *GameStateEditor) OpenMenuBarDropDown(title string, xOffset int, items []services.MenuItem) {
	userInterface := g.engine.GetUI()
	userInterface.OpenXOffsetAutoCloseMenuWithCallback(xOffset, items, func() {
		g.gridIsDirty = true
		g.menuBar.SetDirty()
		g.topStatusLineLabel.SetDirty()
	})
}

// OpenTilePickerDropDown opens a 2-D icon picker modal for the given items.
// It is the preferred way to present tile / item / object palettes in the editor.
func (g *GameStateEditor) OpenTilePickerDropDown(title string, items []services.MenuItem) {
	userInterface := g.engine.GetUI()
	userInterface.OpenTilePicker(title, items,
		func(label string) {
			g.PrintAsMessage(label)
		},
		func() {
			g.PrintAsMessage("")
			g.gridIsDirty = true
			g.menuBar.SetDirty()
			g.topStatusLineLabel.SetDirty()
		},
	)
}

func (g *GameStateEditor) openContextMenuAtMousePos(items []services.MenuItem) {
    userInterface := g.engine.GetUI()
    userInterface.OpenAtPosAutoCloseMenuWithCallback(g.MousePositionOnScreen, items, func() {
        g.gridIsDirty = true
        g.menuBar.SetDirty()
        g.topStatusLineLabel.SetDirty()
    })
}

func (g *GameStateEditor) setSelectionCompletedHandler(f func()) func() {
    return func() {
        g.handler.CellsSelected = func() {
            f()
            return
        }
        return
    }
}

func (g *GameStateEditor) moveCameraOnMap(delta geometry.Point) {
    g.engine.GetGame().GetCamera().MoveBy(delta, g.engine.GetGame().GetMap().MapWidth, g.engine.GetGame().GetMap().MapHeight)
}

func (g *GameStateEditor) updateStatusLine() {
    statusLine := core.Text(fmt.Sprintf("%s | %s %s | fg:@f @N bg:@b @N", g.handler.Name, string(g.selectionTool.Icon()), string(g.placeThingIcon))).WithMarkups(map[rune]common.Style{
        'f': common.Style{Foreground: common.White, Background: g.currentForegroundColor},
        'b': common.Style{Foreground: common.White, Background: g.currentBackgroundColor},
    })
    g.topStatusLineLabel.SetStyledText(statusLine)
    g.topStatusLineLabel.SetDirty()
}

func (g *GameStateEditor) Update(input services.InputInterface) {
    for _, cmd := range input.PollEditorCommands() {
        if !g.menuBar.TryHandle(cmd) {
            switch typedCmd := cmd.(type) {
            case core.PointerCommand:
                g.handlePointerCommand(typedCmd)
                g.topStatusLineLabel.SetDirty()
            case core.KeyCommand:
                g.handleKeyCommand(typedCmd)
            }
        }
    }
}
func (g *GameStateEditor) handleKeyCommand(cmd core.KeyCommand) {
    if uiAction, isGlobal := globalKeyPresses[cmd.Key]; isGlobal {
        uiAction()
        return
    }
    if g.handler.KeyPressed == nil {
        return
    }
    if uiAction, ok := g.handler.KeyPressed[cmd.Key]; ok {
        uiAction()
        return
    }
}

func (g *GameStateEditor) handlePointerCommand(cmd core.PointerCommand) {
    g.MousePositionOnScreen = cmd.Pos
    g.MousePositionInWorld = g.engine.GetGame().GetCamera().ScreenToWorld(g.MousePositionOnScreen)

    switch cmd.Action {
    case core.MouseRight:
        g.selectAtMousePos()
        if len(g.handler.ContextMenu) > 0 {
            g.openContextMenuAtMousePos(g.handler.ContextMenu)
        }
    case core.MouseWheelUp:
        g.moveCameraOnMap(geometry.Point{X: 0, Y: -1})
        g.gridIsDirty = true
    case core.MouseWheelDown:
        g.moveCameraOnMap(geometry.Point{X: 0, Y: 1})
        g.gridIsDirty = true
    case core.MouseWheelLeft:
        g.moveCameraOnMap(geometry.Point{X: -1, Y: 0})
        g.gridIsDirty = true
    case core.MouseWheelRight:
        g.moveCameraOnMap(geometry.Point{X: 1, Y: 0})
        g.gridIsDirty = true
    case core.MouseLeftReleased:
        g.MouseDown = false
        if g.selectionTool != nil {
            g.selectedWorldPositions = g.selectionTool.StopDrawing(g.MousePositionInWorld)
            if ebiten.IsKeyPressed(ebiten.KeyShift) {
                g.shiftClickAt(g.MousePositionInWorld)
            } else if g.handler.CellsSelected != nil {
                g.handler.CellsSelected()
            }
            g.selectedWorldPositions = []geometry.Point{}
            g.gridIsDirty = true
            return
        }
    case core.MouseLeft:
        //println("Mouse down")
        g.MouseDown = true
        if g.selectionTool != nil {
            g.selectionTool.StartDrawing(g.MousePositionInWorld)
            return
        }
    case core.MouseMoved:
        g.gridIsDirty = true
        if g.MouseDown && g.selectionTool != nil {
            g.selectedWorldPositions = g.selectionTool.DraggedOver(g.MousePositionInWorld)
            return
        } else if g.handler.MouseMoved != nil {
            g.handler.MouseMoved()
            return
        }
        g.updateMouseOver()
    }
}
func (g *GameStateEditor) updateMouseOver() {
    model := g.engine.GetGame()
    currentMap := model.GetMap()
    userInterface := g.engine.GetUI()
    shiftBeingHeld := ebiten.IsKeyPressed(ebiten.KeyShift)
    if !currentMap.Contains(g.MousePositionInWorld) || !shiftBeingHeld {
        userInterface.ClearTooltip()
        return
    }
    toolTipText := g.ToolTipAt(g.MousePositionInWorld)
    if toolTipText == "" {
        userInterface.ClearTooltip()
        return
    }
    userInterface.ShowTooltipAt(g.MousePositionOnScreen, core.Text(toolTipText))
    g.gridIsDirty = true
}

func (g *GameStateEditor) shiftClickAt(pos geometry.Point) {
    currentMap := g.engine.GetGame().GetMap()

    if currentMap.IsObjectAt(pos) {
        objectAt := currentMap.ObjectAt(pos)
        if textable, ok := objectAt.(services.Textable); ok {
            g.handler = UIHandler{
                Name: "edit text",
                TextReceived: func(text string) {
                    textable.SetText(text)
                    g.PrintAsMessage("Text updated.")
                    g.SetDirty()
                },
            }
            g.showTextInput("Text: ", textable.GetText())
            return
        }
    }

    if currentMap.IsItemAt(pos) {
        itemAt := currentMap.ItemAt(pos)
        var prompt string
        switch itemAt.Type {
        case core.ItemTypeMessage:
            prompt = "Text file: "
        case core.ItemTypeKey, core.ItemTypeKeyCard:
            prompt = "Key: "
        default:
            return
        }
        g.handler = UIHandler{
            Name: "edit item key",
            TextReceived: func(text string) {
                itemAt.SetKey(text)
                g.PrintAsMessage(fmt.Sprintf("Key of %s set to: %s", itemAt.Name, text))
                g.SetDirty()
            },
        }
        g.showTextInput(prompt, itemAt.GetKey())
    }
}

func (g *GameStateEditor) ToolTipAt(world geometry.Point) string {
    model := g.engine.GetGame()
    currentMap := model.GetMap()
    if !currentMap.Contains(world) {
        return ""
    }
    actor, isActorAt := model.GetMap().TryGetActorAt(world)
    if isActorAt {
        return actor.TooltipText()
    }
    object, isObjectAt := model.GetMap().TryGetObjectAt(world)
    if isObjectAt {
        if keyed, ok := object.(services.KeyBound); ok && keyed.GetKey() != "" {
            return fmt.Sprintf("%s (%s)", object.Description(), keyed.GetKey())
        } else {
            return object.Description()
        }
    }
    item, isItemAt := model.GetMap().TryGetItemAt(world)
    if isItemAt {
        label := item.Name
        if item.GetKey() != "" {
            label = fmt.Sprintf("%s (%s)", item.Name, item.GetKey())
        }
        if item.Buried {
            label += " [buried]"
        }
        return label
    }
    namedLocation := currentMap.GetNamedLocationByPos(world)
    cell := model.GetMap().CellAt(world)
    zoneAt := model.GetMap().ZoneAt(world)
    cellInfo := fmt.Sprintf("%s / Zone: %s", cell.TileType.ToString(), zoneAt.ToString())
    if namedLocation != "" {
        cellInfo += fmt.Sprintf(" / %s", namedLocation)
    }
    return cellInfo
}

func (g *GameStateEditor) SetDirty() {
    g.gridIsDirty = true
}

func (g *GameStateEditor) imageFromSelection() {
    if len(g.selectedWorldPositions) == 0 {
        return
    }
    // find the min and the max points
    minX, minY := g.selectedWorldPositions[0].X, g.selectedWorldPositions[0].Y
    maxX, maxY := g.selectedWorldPositions[0].X, g.selectedWorldPositions[0].Y
    for _, pos := range g.selectedWorldPositions {
        if pos.X < minX {
            minX = pos.X
        }
        if pos.Y < minY {
            minY = pos.Y
        }
        if pos.X > maxX {
            maxX = pos.X
        }
        if pos.Y > maxY {
            maxY = pos.Y
        }
    }
    // create bounds
    bounds := geometry.NewRect(minX, minY, maxX+1, maxY+1) // screen space

    // create image
    image := utils.NewCellImage(uint64(bounds.Size().X), uint64(bounds.Size().Y), g.MapToImage(bounds))
    err := image.SaveToDisk("selectedWorldPositions.cmg")
    if err != nil {
        println(err.Error())
    }

    g.changeUIStateTo(placePrefabUI)
}

func (g *GameStateEditor) MapToImage(bounds geometry.Rect) []common.Cell {
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    result := make([]common.Cell, bounds.Size().X*bounds.Size().Y)
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            worldPos := geometry.Point{X: x, Y: y}
            icon, style := game.DrawWorldAtPosition(worldPos, currentMap.CellAt(worldPos))
            index := (y-bounds.Min.Y)*bounds.Size().X + (x - bounds.Min.X)
            result[index] = common.Cell{Rune: icon, Style: style}
        }
    }
    return result
}
