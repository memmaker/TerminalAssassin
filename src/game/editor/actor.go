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
    taskCount := aic.TaskCountFor(currentActor)
    aic.CalculateAllTaskPaths(currentActor)
    g.SelectedTaskIndex = -1
    g.LastSelectedPos = g.MousePositionInWorld
    // Point SelectedSchedule at the library entry so the schedule editor
    // can display and edit the actor's tasks without a separate selection step.
    if schedName := currentActor.AI.Schedule; schedName != "" {
        g.SelectedSchedule = currentMap.GetSchedule(schedName)
    }
    itemString := "(No Items)"
    if currentActor.Inventory != nil && len(currentActor.Inventory.Items) > 0 {
        itemString = fmt.Sprintf("items: %s", currentActor.Inventory.AsRunes())
    }
    teamString := ""
    if currentActor.Team != "" {
        teamString = fmt.Sprintf(" (%s)", currentActor.Team)
    }
    g.PrintAsMessage(fmt.Sprintf(currentActor.Name+teamString+" ("+strconv.Itoa(taskCount)+" tasks) - %s", itemString))
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
        follower.AI.Schedule = ""
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
        g.SelectedActor.Type = core.ActorTypeTarget
    case core.ActorTypeTarget:
        g.SelectedActor.Type = core.ActorTypeFence
    case core.ActorTypeFence:
        g.SelectedActor.Type = core.ActorTypePredator
    case core.ActorTypePredator:
        g.SelectedActor.Type = core.ActorTypeCivilian
    default:
        g.SelectedActor.Type = core.ActorTypeCivilian
    }
    g.PrintAsMessage(fmt.Sprintf("%s is now a %s", g.SelectedActor.Name, g.SelectedActor.Type))
    return
}

func (g *GameStateEditor) quickAddActor() {
    currentMap := g.engine.GetGame().GetMap()

    pos := g.MousePositionInWorld
    g.LastSelectedPos = pos
    if currentMap.IsActorAt(pos) || (!currentMap.IsTileWalkable(pos)) || currentMap.IsObjectAt(pos) || currentMap.IsObjectAt(pos) {
        return
    }

    actorNumber := len(currentMap.Actors()) + 1
    actorName := fmt.Sprintf("actor #%d", actorNumber)

    newActor := core.NewActor(actorName)
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
    g.changeUIStateTo(addActorsUI)
    if text != "" {
        newActor := core.NewActor(text)
        currentMap.AddActor(newActor, g.LastSelectedPos)
        currentMap.UpdateFieldOfView(newActor)
        g.selectActorAt(g.LastSelectedPos)
        g.adjustLookDirectionForSelectedActor()
    }
}

func (g *GameStateEditor) setTeamForActor() {
    if g.SelectedActor == nil {
        return
    }
    currentMap := g.engine.GetGame().GetMap()
    existingTeams := collectTeams(currentMap.Actors())
    if len(existingTeams) == 0 {
        g.promptNewTeam()
        return
    }
    menuItems := make([]services.MenuItem, 0, len(existingTeams)+2)
    for _, team := range existingTeams {
        t := team
        menuItems = append(menuItems, services.MenuItem{
            Label: t,
            Handler: func() {
                g.SelectedActor.Team = t
                g.PrintAsMessage(fmt.Sprintf("%s team: %s", g.SelectedActor.Name, t))
                g.changeUIStateTo(editActorUI)
            },
        })
    }
    menuItems = append(menuItems, services.MenuItem{
        Label:   "(new team...)",
        Handler: g.promptNewTeam,
    })
    menuItems = append(menuItems, services.MenuItem{
        Label: "(no team)",
        Handler: func() {
            g.SelectedActor.Team = ""
            g.PrintAsMessage(fmt.Sprintf("%s removed from team", g.SelectedActor.Name))
            g.changeUIStateTo(editActorUI)
        },
    })
    g.OpenMenuBarDropDown("Set team", 0, menuItems)
}

func (g *GameStateEditor) promptNewTeam() {
    g.handler = UIHandler{Name: "enter team name", TextReceived: func(name string) {
        if name == "" {
            return
        }
        g.SelectedActor.Team = name
        g.PrintAsMessage(fmt.Sprintf("%s team: %s", g.SelectedActor.Name, name))
        g.changeUIStateTo(editActorUI)
    }}
    g.showTextInput("Team name: ", "")
}

func collectTeams(actors []*core.Actor) []string {
    seen := make(map[string]bool)
    var teams []string
    for _, a := range actors {
        if a.Team != "" && !seen[a.Team] {
            seen[a.Team] = true
            teams = append(teams, a.Team)
        }
    }
    return teams
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
    }, nil)
}
