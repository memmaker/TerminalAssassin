package editor

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

func (g *GameStateEditor) prefabFromSelection() {
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
	bounds := geometry.NewRect(minX, minY, maxX+1, maxY+1)

	// create prefab
	prefab := gridmap.NewPrefabFromMap[*core.Actor, *core.Item, services.Object](g.engine.GetGame().GetMap(), bounds)

	g.currentPrefab = prefab
	g.changeUIStateTo(placePrefabUI)
}

func (g *GameStateEditor) placePrefab() {
	if g.selectedWorldPositions == nil || len(g.selectedWorldPositions) == 0 {
		return
	}
	pos := g.selectedWorldPositions[0]
	g.currentPrefab.SetMapRegion(g.engine.GetGame().GetMap(), pos)
}
