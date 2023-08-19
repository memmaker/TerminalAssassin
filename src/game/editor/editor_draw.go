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

		if g.LastSelectedPos == worldPos {
			style = style.WithBg(core.ColorFromCode(core.ColorMarked))
		}
		if g.SelectedZone != nil {
			zoneAt := currentMap.ZoneAt(worldPos)
			if zoneAt != nil && zoneAt.Name == g.SelectedZone.Name {
				zoneColor := core.ColorFromCode(core.ColorEnforcer)
				if zoneAt.IsDropOff() {
					zoneColor = core.ColorFromCode(core.ColorWater)
				} else if zoneAt.IsHighSecurity() {
					zoneColor = core.ColorFromCode(core.ColorBrightRed)
				} else if zoneAt.IsPublic() {
					zoneColor = core.ColorFromCode(core.ColorMapGreen)
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
		con.SetSquare(m.GetCamera().WorldToScreen(currentMap.PlayerSpawn), common.Cell{Rune: '@', Style: common.Style{Foreground: common.Black, Background: common.Green}})
	}

	if g.SelectedActor != nil {
		g.SelectedActor.VisionCone(func(worldPos geometry.Point) {
			if !m.GetCamera().ViewPort.Contains(worldPos) {
				return
			}
			screenPos := m.GetCamera().WorldToScreen(worldPos)
			cellAt := con.AtSquare(screenPos)
			con.SetSquare(screenPos, cellAt.WithBackgroundColor(common.Green))
		})
		if len(g.SelectedActor.AI.Schedule.Tasks) > 0 {
			g.drawTasks(con, g.SelectedActor)
		}
	}
	if len(g.selectedWorldPositions) > 0 {
		for _, p := range g.selectedWorldPositions {
			if !m.GetCamera().ViewPort.Contains(p) {
				continue
			}
			screenPos := m.GetCamera().WorldToScreen(p)
			cellAt := con.AtSquare(screenPos)
			con.SetSquare(screenPos, cellAt.WithBackgroundColor(core.ColorFromCode(core.ColorMarked)))
		}
	}
	for name, location := range currentMap.NamedLocations {
		if !m.GetCamera().ViewPort.Contains(location) {
			continue
		}
		screenPos := m.GetCamera().WorldToScreen(location)
		cell := con.AtSquare(screenPos)
		colorForLocation := core.ColorFromCode(core.ColorEnforcer)
		if g.selectedNamedLocation != "" && g.selectedNamedLocation == name {
			colorForLocation = core.ColorFromCode(core.ColorGood)
		}
		con.SetSquare(screenPos, cell.WithBackgroundColor(colorForLocation))
	}
	if g.currentPrefab != nil {
		g.currentPrefab.Draw(con, g.MousePositionOnScreen)
	}

	cellAtMouse := con.AtSquare(g.MousePositionOnScreen)
	con.SetSquare(g.MousePositionOnScreen, cellAtMouse.WithStyle(cellAtMouse.Style.Reversed()))

	//con.HalfWidthFill(geometry.NewRect(0, 0, g.engine.ScreenGridWidth()*2, g.engine.ScreenGridHeight()), common.Cell{Rune: ' ', Style: common.Style{Foreground: common.White, Background: common.Transparent}})
	g.gridIsDirty = false
}

func (g *GameStateEditor) clearHUDArea(con console.CellInterface) int {
	mapHeight := g.engine.MapWindowHeight()
	gridWidth, gridHeight := g.engine.ScreenGridWidth(), g.engine.ScreenGridHeight()
	con.SquareFill(geometry.NewRect(0, mapHeight, gridWidth, gridHeight), common.Cell{Rune: ' ', Style: common.DefaultStyle})
	return mapHeight
}

func (g *GameStateEditor) drawTasks(grid console.CellInterface, selectedActor *core.Actor) {
	camera := g.engine.GetGame().GetCamera()
	mapHeight := g.engine.MapWindowHeight()
	var selectedPos geometry.Point
	if g.SelectedTaskIndex > -1 {
		selectedPos = selectedActor.AI.Schedule.Tasks[g.SelectedTaskIndex].Location
	}
	for i, task := range selectedActor.AI.Schedule.Tasks {
		itoa := strconv.Itoa(i + 1)
		taskPosOnScreen := camera.WorldToScreen(task.Location)
		if taskPosOnScreen.Y >= 0 && taskPosOnScreen.Y < mapHeight {
			grid.SetSquare(taskPosOnScreen, common.Cell{Rune: []rune(itoa)[0], Style: common.Style{Foreground: core.ColorFromCode(core.ColorBlood), Background: core.ColorFromCode(core.ColorGray)}})
		}
		for _, pos := range task.KnownPath {
			if !camera.ViewPort.Contains(pos) {
				continue
			}
			screenPos := camera.WorldToScreen(pos)
			cellAtPos := grid.AtSquare(screenPos)
			if selectedPos == pos {
				cellAtPos.Style.Background = core.ColorFromCode(core.ColorMarked)
			} else {
				cellAtPos.Style.Background = core.ColorFromCode(core.ColorWarning)
			}

			grid.SetSquare(screenPos, cellAtPos)
		}
	}
}
