package states

import (
    "fmt"
    "math"
    "strconv"

    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/gridmap"
)

func (g *GameStateGameplay) OpenPauseMenu() {
    userInterface := g.engine.GetUI()
    audio := g.engine.GetAudio()
    menuItems := []services.MenuItem{
        {
            Label: "Resume mission",
        },
        {
            Label:   "Abort mission",
            Handler: g.quitToMainMenu,
        },
        {
            DynamicLabel: func() string {
                return "Master Volume: " + strconv.Itoa(int(audio.GetMasterVolume()*100)) + "%"
            },
            LeftHandler: func() {
                audio.SetMasterVolume(audio.GetMasterVolume() - 0.1)
            },
            RightHandler: func() {
                audio.SetMasterVolume(audio.GetMasterVolume() + 0.1)
            },
        },
        {
            DynamicLabel: func() string {
                return "Music Volume: " + strconv.Itoa(int(audio.GetMusicVolume()*100)) + "%"
            },
            LeftHandler: func() {
                audio.SetMusicVolume(audio.GetMusicVolume() - 0.1)
            },
            RightHandler: func() {
                audio.SetMusicVolume(audio.GetMusicVolume() + 0.1)
            },
        },
        {
            DynamicLabel: func() string {
                return "Sound Volume: " + strconv.Itoa(int(audio.GetSoundVolume()*100)) + "%"
            },
            LeftHandler: func() {
                audio.SetSoundVolume(audio.GetSoundVolume() - 0.1)
            },
            RightHandler: func() {
                audio.SetSoundVolume(audio.GetSoundVolume() + 0.1)
            },
        },
    }
    userInterface.OpenFixedWidthAutoCloseMenuWithCallback("Paused", menuItems, nil)
    //g.engine.GetGame().PushState(&GameStateMainMenu{})
}

func (g *GameStateGameplay) quitToMainMenu() {
    g.engine.Reset()
}

func (g *GameStateGameplay) BeginAimOrUseItem() {
    currentMap := g.engine.GetGame().GetMap()
    player := currentMap.Player
    if player.EquippedItem == nil {
        g.Ui = aimingUIState
        g.AdjustPlayerAimFromMouse()
        return
    }
    if player.EquippedItem.InsteadOfUse != nil && player.EquippedItem.HasUsesLeft() {
        player.EquippedItem.DecreaseUsesLeft()
        player.EquippedItem.InsteadOfUse()
        return
    }
    if player.EquippedItem.RangedAttack == core.NoAction && player.EquippedItem.MeleeAttack == core.NoAction {
        return
    }
    if player.FovMode == gridmap.FoVModeScoped {
        g.resetPlayerState()
        currentMap.UpdateFieldOfView(player)
        return
    }
    g.Ui = aimingUIState
    if player.EquippedItem.Scope.FoVinDegrees > 0 {
        player.FovMode = gridmap.FoVModeScoped
        currentMap.UpdateFieldOfView(player)
    }
    g.AdjustPlayerAimFromMouse()
    return
}

func (g *GameStateGameplay) BeginExamine() {
    g.Ui = examineUIState
    return
}

func (g *GameStateGameplay) EquipNextInventoryItem(person *core.Actor) {
    defer g.UpdateHUD()
    if len(person.Inventory.Items) == 0 {
        return
    }
    if person.EquippedItem == nil {
        person.EquippedItem = person.Inventory.Items[0]
        return
    }
    if person.EquippedItem.IsBig || person.EquippedItem.OnCooldown {
        return
    }
    for i, item := range person.Inventory.Items {
        if item == person.EquippedItem {
            nextIndex := (i + 1) % len(person.Inventory.Items)
            person.EquippedItem = person.Inventory.Items[nextIndex]
            return
        }
    }
    return
}

func (g *GameStateGameplay) EquipPreviousInventoryItem(person *core.Actor) {
    defer g.UpdateHUD()
    if len(person.Inventory.Items) == 0 {
        return
    }
    if person.EquippedItem == nil {
        person.EquippedItem = person.Inventory.Items[0]
        return
    }
    if person.EquippedItem.IsBig || person.EquippedItem.OnCooldown {
        return
    }
    for i, item := range person.Inventory.Items {
        if item == person.EquippedItem {
            nextIndex := i - 1
            if nextIndex < 0 {
                nextIndex = len(person.Inventory.Items) - 1
            }
            person.EquippedItem = person.Inventory.Items[nextIndex]
            return
        }
    }
    return
}

func (g *GameStateGameplay) ShowFocusedActorPager() {
    if g.FocusedActor == nil {
        return
    }
    userInterface := g.engine.GetUI()
    userInterface.ShowPager(g.FocusedActor.Name, ToStyled(g.FocusedActor.StrList()), nil)
}

func ToStyled(list []string) []core.StyledText {
    result := make([]core.StyledText, len(list))
    for i, s := range list {
        result[i] = core.Text(s)
    }
    return result
}

func (g *GameStateGameplay) AdjustPlayerAimFromPad(xAxis float64, yAxis float64) {
	const deadzone = 0.08
	const aimSpeed = 0.3 // tiles per tick at full stick deflection

	if math.Abs(xAxis) < deadzone && math.Abs(yAxis) < deadzone {
		// Stick released: cursor stays in place, keep aiming mode active.
		return
	}

	game := g.engine.GetGame()
	currentMap := game.GetMap()
	player := currentMap.Player

	if g.Ui.ID != aimingUIState.ID {
		// Entering aim mode: mirror what BeginAimOrUseItem does for keyboard/mouse.
		g.Ui = aimingUIState
		g.padAimActive = true
		g.padAimPos = player.FoVSource().ToPointF()
		if player.EquippedItem != nil && player.EquippedItem.Scope.FoVinDegrees > 0 {
			player.FovMode = gridmap.FoVModeScoped
			currentMap.UpdateFieldOfView(player)
		}
	}

	g.padAimPos.X += xAxis * aimSpeed
	g.padAimPos.Y += yAxis * aimSpeed
	aimTarget := g.padAimPos.ToPointRounded()
	g.AdjustPlayerAim(aimTarget)

	// Scoped weapons: camera scrolling is handled by scrollCameraForScope,
	// identical to the mouse path. For non-scoped weapons scroll immediately.
	if player.FovMode != gridmap.FoVModeScoped {
		g.ensureWorldPosInView(aimTarget, 4)
	}
}

func (g *GameStateGameplay) AdjustPlayerAimFromMouse() {
    g.AdjustPlayerAim(g.MousePositionInWorld)
}

func (g *GameStateGameplay) AdjustPlayerAim(aimPosInWorld geometry.Point) {
    m := g.engine.GetGame()
    a := m.GetMap().Player
    //cam.CenterOn(mouseInWorld, g.engine.MapWindowWidth(), g.engine.MapWindowHeight())

    a.LookDirection = geometry.DirectionVectorToAngleInDegrees(aimPosInWorld.Sub(a.FoVSource()))
    aimDistanceSquared := a.VisionRange() * a.VisionRange()
    if a.EquippedItem == nil || (a.EquippedItem.RangedAttack == core.NoAction && a.EquippedItem.MeleeAttack != core.NoAction) {
        // melee
        aimDistanceSquared = 1
    } else if a.EquippedItem.Scope.Range > 0 {
        // scoped
        aimDistanceSquared = a.EquippedItem.Scope.Range * a.EquippedItem.Scope.Range
    }

    g.TargetLoS = geometry.LineOfSight(a.FoVSource(), aimPosInWorld, func(p geometry.Point) bool {
        return geometry.DistanceSquared(p, a.FoVSource()) < aimDistanceSquared
    })
    return
}

func (g *GameStateGameplay) ToggleShowCompleteMap() {
    g.engine.GetGame().GetMap().SetAllExplored()
    g.DebugModeActive = !g.DebugModeActive
    g.isDirty = true
}

func (g *GameStateGameplay) ToggleShowAllVisionCones() {
    g.DebugShowAllVisionCones = !g.DebugShowAllVisionCones
    return
}
func (g *GameStateGameplay) SetFocusedActor(actorToFocus *core.Actor) {
    if g.FocusedActor == actorToFocus || actorToFocus == nil {
        return
    }
    if g.FocusedActor != nil {
        g.FocusedActor.DisableDebugTrace()
    }
    g.FocusedActor = actorToFocus
    g.FocusedActor.EnableDebugTrace()
    println(fmt.Sprintf("Focused on %s", g.FocusedActor.DebugDisplayName()))
}
func (g *GameStateGameplay) AimedItemUse() {
    m := g.engine.GetGame()
    currentMap := m.GetMap()
    actions := m.GetActions()
    player := currentMap.Player
    if len(g.TargetLoS) == 0 {
        return
    }

    aimPoint := g.TargetLoS[len(g.TargetLoS)-1]
    // No Item = "prod"
    distanceManhattan := geometry.DistanceManhattan(player.Pos(), aimPoint)
    isRanged := distanceManhattan > 1
    isMeleeRange := distanceManhattan == 1
    isSelfApplied := distanceManhattan == 0
    if player.EquippedItem == nil {
        if isMeleeRange {
            actions.Prod(player, aimPoint)
        }
        return
    }

    if player.EquippedItem.OnCooldown ||
        !player.EquippedItem.HasUsesLeft() ||
        !player.CanUseItems() {
        return
    }

    if player.EquippedItem.InsteadOfUse != nil {
        player.EquippedItem.InsteadOfUse()
        return
    }

    if isSelfApplied && player.EquippedItem.SelfUse != core.NoAction {
        player.EquippedItem.DecreaseUsesLeft()
        actions.UseEquippedItemOnSelf(player)
    } else if isMeleeRange && player.EquippedItem.MeleeAttack != core.NoAction {
        if player.EquippedItem.MeleeAttack != core.ActionTypeMeleeAttack {
            player.EquippedItem.DecreaseUsesLeft()
        }
        actions.UseEquippedItemForMelee(player, aimPoint)
    } else if isRanged && player.EquippedItem.RangedAttack != core.NoAction {
        player.EquippedItem.DecreaseUsesLeft()
        actions.UseEquippedItemAtRange(player, aimPoint)
    }
    if player.EquippedItem == nil {
        g.resetPlayerState()
    }
}
