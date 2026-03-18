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

// LockOnAngleDegrees is the half-width of the snap cone for thrown-item
// target locking. Any visible NPC whose bearing from the player is within this
// many degrees of the right-stick direction will be considered for lock-on.
// The NPC with the smallest angular delta wins.
var LockOnAngleDegrees = 15.0

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

// tryLockOn returns the position of the visible, active NPC within maxRange
// tiles whose bearing from the player is closest to the stick direction and
// falls inside LockOnAngleDegrees. Returns (pos, true) when a suitable
// target is found, (zero, false) otherwise.
//
// Rather than iterating every actor on the map, only the tiles inside the
// circular area of radius maxRange are examined via TryGetActorAt.
func (g *GameStateGameplay) tryLockOn(player *core.Actor, stickX, stickY float64, maxRange int) (geometry.Point, bool) {
    stickAngle := geometry.DirectionVectorToAngleInDegreesF(stickX, stickY)
    currentMap := g.engine.GetGame().GetMap()
    origin := player.FoVSource()
    maxRangeSq := maxRange * maxRange

    bestPos := geometry.Point{}
    bestDelta := LockOnAngleDegrees
    found := false

    for _, actor := range currentMap.Actors() {
        p := actor.Pos()
        if actor == player || !actor.IsActive() || !player.CanSee(p) {
            continue
        }
        if geometry.DistanceSquared(origin, p) > maxRangeSq {
            continue
        }
        delta := geometry.AngleDeltaDegrees(stickAngle, geometry.DirectionVectorToAngleInDegrees(p.Sub(origin)))
        if delta < bestDelta {
            bestDelta = delta
            bestPos = p
            found = true
        }
    }
    return bestPos, found
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

    aimDistance := player.AimDistance()

    origin := player.FoVSource().ToPointF()
    dx := g.padAimPos.X - origin.X
    dy := g.padAimPos.Y - origin.Y
    if dist := math.Sqrt(dx*dx + dy*dy); dist > float64(aimDistance) {
        scale := float64(aimDistance) / dist
        g.padAimPos.X = origin.X + dx*scale
        g.padAimPos.Y = origin.Y + dy*scale
    }

    aimTarget := g.padAimPos.ToPointRounded()

    // Lock-on: only for thrown items, snap to the best NPC within range.
    if player.HasThrownItemEquipped() {
        if lockTarget, locked := g.tryLockOn(player, xAxis, yAxis, aimDistance); locked {
            aimTarget = lockTarget
            g.padAimPos = lockTarget.ToPointF()
        }
    }

    g.AdjustPlayerAim(aimTarget)

    // Scoped weapons: camera scrolling is handled by scrollCameraForScope,
    // identical to the mouse path. For non-scoped weapons scroll immediately.
    if player.FovMode != gridmap.FoVModeScoped {
        g.ensureWorldPosInView(aimTarget, 4)
    }
}

// assassinationTargets returns adjacent active NPCs that can be assassinated.
// Single assassination: 1 piercing weapon + 1 adjacent NPC (no skill required).
// Double assassination: DoubleAssassination skill + 2 piercing weapons + 2 adjacent NPCs.
// Returns nil if no valid targets exist.
func assassinationTargets(engine services.Engine) map[*core.Actor]struct{} {
    player := engine.GetGame().GetMap().Player

    piercingCount := 0
    for _, item := range player.Inventory.Items {
        if item.HasMeleePiercingDamage() {
            piercingCount++
        }
    }
    if piercingCount < 1 {
        return nil
    }

    maxTargets := 1
    if piercingCount >= 2 && engine.GetCareer().UnlockedSkills.DoubleAssassination {
        maxTargets = 2
    }

    currentMap := engine.GetGame().GetMap()
    targets := make(map[*core.Actor]struct{})
    for _, pos := range currentMap.NeighborsAll(player.Pos(), func(p geometry.Point) bool {
        return currentMap.IsActorAt(p)
    }) {
        if actor := currentMap.ActorAt(pos); actor.IsActive() {
            targets[actor] = struct{}{}
            if len(targets) == maxTargets {
                return targets
            }
        }
    }
    if len(targets) > 0 {
        return targets
    }
    return nil
}

func (g *GameStateGameplay) AdjustPlayerAimFromMouse() {
    g.AdjustPlayerAim(g.MousePositionInWorld)
}

func (g *GameStateGameplay) AdjustPlayerAim(aimPosInWorld geometry.Point) {
    m := g.engine.GetGame()
    a := m.GetMap().Player

    a.LookDirection = geometry.DirectionVectorToAngleInDegrees(aimPosInWorld.Sub(a.FoVSource()))
    aimDistance := a.AimDistance()
    aimDistanceSquared := aimDistance * aimDistance

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
