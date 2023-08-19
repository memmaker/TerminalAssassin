package editor

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/game/services"
	"strconv"

	"github.com/memmaker/terminal-assassin/game/ai"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

func (g *GameStateEditor) selectActorAt(pos geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()
	aic := g.engine.GetAI()
	currentActor := currentMap.ActorAt(pos)
	g.SelectedActor = currentActor
	taskCount := len(currentActor.AI.Schedule.Tasks)
	aic.CalculateAllTaskPaths(currentActor)
	g.SelectedTaskIndex = -1
	g.LastSelectedPos = g.MousePositionInWorld
	itemString := "(No Items)"
	if currentActor.Inventory != nil && len(currentActor.Inventory.Items) > 0 {
		itemString = fmt.Sprintf("items: %s", currentActor.Inventory.AsRunes())
	}
	g.PrintAsMessage(fmt.Sprintf("(S) "+currentActor.Name+" ("+strconv.Itoa(taskCount)+" tasks) - %s", itemString))
}

func (g *GameStateEditor) deleteActor() {
	if g.SelectedActor == nil {
		return
	}
	currentMap := g.engine.GetGame().GetMap()
	currentMap.RemoveActor(g.SelectedActor)
	g.SelectedActor = nil
	g.SetDirty()
	return
}

func (g *GameStateEditor) renameSelectedActor() {
	g.handler = UIHandler{Name: "rename actor", TextReceived: func(content string) {
		g.SelectedActor.Name = content
		g.changeUIStateTo(editActorUI)
	}}
	g.showTextInput("New name: ", "")
}

func (g *GameStateEditor) moveSelectedActor() {
	if g.SelectedActor == nil {
		g.PrintAsMessage("ERR: select an Actor first")
		return
	}
	g.setBrushHandler(editActorUI, 'A', func(point geometry.Point) {
		currentMap := g.engine.GetGame().GetMap()
		if !currentMap.IsCurrentlyPassable(point) {
			g.PrintAsMessage("ERR: cannot move Actor to non-passable terrain")
			return
		}
		currentMap.MoveActor(g.SelectedActor, point)
		aic := g.engine.GetAI()
		aic.CalculateAllTaskPaths(g.SelectedActor)
		currentMap.UpdateFieldOfView(g.SelectedActor)
		g.changeUIStateTo(editActorUI)
		g.gridIsDirty = true
	})()
}

func (g *GameStateEditor) selectLeaderForActor() {
	if g.SelectedActor == nil {
		g.PrintAsMessage("ERR: select an Actor first")
		return
	}
	g.PrintAsMessage("Select a Leader for " + g.SelectedActor.Name)
	g.handler = UIHandler{Name: "select Leader", CellsSelected: g.setLeaderForActor(g.SelectedActor)}
}

func (g *GameStateEditor) setLeaderForActor(follower *core.Actor) func() {
	currentMap := g.engine.GetGame().GetMap()
	return func() {
		leader := currentMap.ActorAt(g.MousePositionInWorld)
		if leader == nil {
			g.PrintAsMessage("ERR: no Actor at mouse position")
			return
		}
		if leader == follower {
			g.PrintAsMessage("ERR: cannot set self as Leader")
			return
		}
		offset := follower.Pos().Sub(leader.Pos())
		follower.Status = core.ActorStatusFollowing
		follower.AI.SetState(&ai.FollowerMovement{LeaderStartsAt: leader.Pos(), PosOffset: offset})
		follower.AI.Schedule.Clear()
		g.PrintAsMessage(fmt.Sprintf("OK: %s is now following %s", follower.Name, leader.Name))
		g.changeUIStateTo(editActorUI)
	}
}

func (g *GameStateEditor) toggleActorType() {
	if g.SelectedActor == nil {
		return
	}
	currentType := g.SelectedActor.Type
	switch currentType {
	case core.ActorTypeCivilian:
		g.SelectedActor.Type = core.ActorTypeGuard
	case core.ActorTypeGuard:
		g.SelectedActor.Type = core.ActorTypeEnforcer
	case core.ActorTypeEnforcer:
		g.SelectedActor.Type = core.ActorTypeTarget
	default:
		g.SelectedActor.Type = core.ActorTypeCivilian
	}
	g.PrintAsMessage(fmt.Sprintf("%s is now a %s", g.SelectedActor.Name, g.SelectedActor.Type))
	return
}

func (g *GameStateEditor) quickAddActor() {
	currentMap := g.engine.GetGame().GetMap()
	data := g.engine.GetData()

	pos := g.MousePositionInWorld
	g.LastSelectedPos = pos
	if currentMap.IsActorAt(pos) || (!currentMap.IsTileWalkable(pos)) || currentMap.IsObjectAt(pos) || currentMap.IsObjectAt(pos) {
		return
	}

	actorNumber := len(currentMap.Actors()) + 1
	actorName := fmt.Sprintf("actor #%d", actorNumber)

	newActor := core.NewActor(actorName, data.DefaultClothing())
	currentMap.AddActor(newActor, g.LastSelectedPos)
	currentMap.UpdateFieldOfView(newActor)
	g.selectActorAt(g.LastSelectedPos)
	g.changeUIStateTo(editActorUI)
}

func (g *GameStateEditor) addActor() {
	currentMap := g.engine.GetGame().GetMap()
	pos := g.MousePositionInWorld
	g.LastSelectedPos = pos
	if currentMap.IsActorAt(pos) || (!currentMap.IsTileWalkable(pos)) || currentMap.IsObjectAt(pos) || currentMap.IsObjectAt(pos) {
		return
	}

	g.handler = UIHandler{Name: "enter actor name", TextReceived: g.spawnActorWithName}
	g.showTextInput("Actor name: ", "")
	return
}

func (g *GameStateEditor) spawnActorWithName(text string) {
	currentMap := g.engine.GetGame().GetMap()
	data := g.engine.GetData()
	g.changeUIStateTo(addActorsUI)
	if text != "" {
		newActor := core.NewActor(text, data.DefaultClothing())
		currentMap.AddActor(newActor, g.LastSelectedPos)
		currentMap.UpdateFieldOfView(newActor)
		g.selectActorAt(g.LastSelectedPos)
		g.adjustLookDirectionForSelectedActor()
	}
}

func (g *GameStateEditor) adjustLookDirectionForSelectedActor() {
	g.handler = UIHandler{
		Name:          "edit look direction",
		MouseMoved:    g.updateSelectedActorDirection,
		CellsSelected: g.changeUIStateFunc(editActorUI),
	}
}

func (g *GameStateEditor) updateSelectedActorDirection() {
	actorPos := g.SelectedActor.Pos()
	mousePos := g.MousePositionInWorld
	direction := mousePos.Sub(actorPos)
	g.SelectedActor.LookDirection = geometry.DirectionVectorToAngleInDegrees(direction)
	return
}

func (g *GameStateEditor) openInventoryForSelectedActor() {
	if g.SelectedActor == nil || g.SelectedActor.Inventory.IsEmpty() {
		return
	}

	selectedItem := g.SelectedActor.Inventory.Items[0]
	userInterface := g.engine.GetUI()
	userInterface.OpenItemRingMenu(selectedItem, g.SelectedActor.Inventory.Items, func(item *core.Item) {
		g.changeUIStateTo(addItemsUI)
		g.selectedItem = item
		itemString := services.EncodeItemAsString(item)
		g.PrintAsMessage(fmt.Sprintf("Item: %s", itemString))
	}, func() {
		//g.UpdateHUD()
	})
}
