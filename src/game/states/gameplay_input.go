package states

import (
    "fmt"
    "math"
    "sort"
    "strconv"
    "time"

    "github.com/memmaker/terminal-assassin/common"
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
            Label:   "Wait...",
            Handler: g.openWaitMenu,
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

func (g *GameStateGameplay) BeginMouseAiming() {
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
    if player.EquippedItem.RangedAttack == core.NoAction && player.EquippedItem.MeleeAttack == core.NoAction &&
        player.EquippedItem.Type != core.ItemTypeFlashlight {
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

    // Only activate ranged aiming when the player actually has a ranged/throwable item or a flashlight.
    if player.EquippedItem == nil || (player.EquippedItem.RangedAttack == core.NoAction &&
        player.EquippedItem.Type != core.ItemTypeFlashlight) {
        return
    }

    if g.Ui.ID != aimingUIState.ID {
        // Entering aim mode: mirror what BeginMouseAiming does for keyboard/mouse.
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

// assassinationTargets returns adjacent active NPCs that can be assassinated,
// ordered by angular proximity to the player's current look direction so the
// most "aimed-at" actor is always preferred.
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

    type candidate struct {
        actor      *core.Actor
        angleDelta float64
    }
    var candidates []candidate
    for _, pos := range currentMap.NeighborsAll(player.Pos(), func(p geometry.Point) bool {
        if !currentMap.Contains(p) || !player.CanSee(p) || !currentMap.IsActorAt(p) {
            return false
        }
        actor := currentMap.ActorAt(p)
        return actor.IsActive()
    }) {
        actor := currentMap.ActorAt(pos)
        bearing := geometry.DirectionVectorToAngleInDegrees(pos.Sub(player.Pos()))
        delta := geometry.AngleDeltaDegrees(player.LookDirection, bearing)
        candidates = append(candidates, candidate{actor: actor, angleDelta: delta})
    }
    if len(candidates) == 0 {
        return nil
    }

    // Prefer the actor whose bearing is closest to the player's look direction.
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].angleDelta < candidates[j].angleDelta
    })

    targets := make(map[*core.Actor]struct{})
    for i := 0; i < maxTargets && i < len(candidates); i++ {
        targets[candidates[i].actor] = struct{}{}
    }
    return targets
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

// aimInLookDirection sets TargetLoS to the line extending from the player's
// FoV source in the current LookDirection, up to AimDistance. Used so that
// R2 always fires even when no explicit aim has been set.
func (g *GameStateGameplay) aimInLookDirection() {
    player := g.engine.GetGame().GetMap().Player
    radians := player.LookDirection * math.Pi / 180.0
    dx := math.Cos(radians)
    dy := math.Sin(radians)
    target := geometry.VectorInDirectionWithLength(player.FoVSource(), dx, dy, player.AimDistance())
    g.AdjustPlayerAim(target)
}

func (g *GameStateGameplay) setAimFromPeekDirection(direction geometry.Point) {
    player := g.engine.GetGame().GetMap().Player
    g.AdjustPlayerAim(player.Pos().Add(direction.Mul(player.AimDistance())))
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
func (g *GameStateGameplay) UseRangedItemInLoS() {
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

// openWaitMenu presents the player with a list of preset times and a custom
// input option.  Selecting an entry instantly advances the in-game clock to
// the chosen time and updates the environment lighting accordingly.
func (g *GameStateGameplay) openWaitMenu() {
    currentMap := g.engine.GetGame().GetMap()
    now := currentMap.TimeOfDay.Format("15:04")
    ui := g.engine.GetUI()
    menuItems := []services.MenuItem{
        {
            Label:   "Until Dawn  (06:00)",
            Handler: func() { g.waitUntilHour(6, 0) },
            Icon:    core.GlyphStreetLight,
        },
        {
            Label:   "Until Noon  (12:00)",
            Handler: func() { g.waitUntilHour(12, 0) },
            Icon:    core.GlyphStreetLight,
        },
        {
            Label:   "Until Dusk  (18:00)",
            Handler: func() { g.waitUntilHour(18, 0) },
            Icon:    core.GlyphFog,
        },
        {
            Label:   "Until Night (22:00)",
            Handler: func() { g.waitUntilHour(22, 0) },
            Icon:    core.GlyphFog,
        },
        {
            Label:   "Custom time (HH:MM)...",
            Handler: g.waitUntilCustomTime,
            Icon:    core.GlyphSmartphone,
        },
    }
    ui.OpenFixedWidthAutoCloseMenu(fmt.Sprintf("Wait (%s)", now), menuItems)
}

// waitUntilHour instantly advances the in-game clock to the next occurrence
// of the given hour:minute and updates the ambient lighting.
func (g *GameStateGameplay) waitUntilHour(hour, minute int) {
    currentMap := g.engine.GetGame().GetMap()
    current := currentMap.TimeOfDay

    targetMins := hour*60 + minute
    currentMins := current.Hour()*60 + current.Minute()
    diff := targetMins - currentMins
    if diff <= 0 {
        diff += 24 * 60 // target is tomorrow
    }

    currentMap.TimeOfDay = current.Add(time.Duration(diff) * time.Minute)
    currentMap.SetAmbientLight(common.GetAmbientLightFromDayTime(currentMap.TimeOfDay).ToRGB())
    g.timeAccumulator = 0
    g.isDirty = true
    g.Print(fmt.Sprintf("Time passed. It is now %s.", currentMap.TimeOfDay.Format("15:04")))
    g.UpdateStatusLine()
}

// waitUntilCustomTime opens a text-input prompt where the player can type a
// target time in HH:MM format.
func (g *GameStateGameplay) waitUntilCustomTime() {
    currentMap := g.engine.GetGame().GetMap()
    prompt := fmt.Sprintf("Wait until (HH:MM, now %s): ", currentMap.TimeOfDay.Format("15:04"))
    g.engine.GetUI().ShowTextInput(prompt, "", func(text string) {
        var h, m int
        if _, err := fmt.Sscanf(text, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
            g.Print("Invalid time. Use HH:MM, e.g. 22:30.")
            return
        }
        g.waitUntilHour(h, m)
    }, func() {
        g.Print("Cancelled.")
    })
}
