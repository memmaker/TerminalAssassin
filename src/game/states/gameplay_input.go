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
		g.AdjustPlayerAim()
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
	g.AdjustPlayerAim()
	return
}

func (g *GameStateGameplay) BeginExamine() {
	g.Ui = examineUIState
	return
}

func (g *GameStateGameplay) EquipNextInventoryItem(person *core.Actor) {
	defer g.UpdateContextActions()
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
	defer g.UpdateContextActions()
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
	currentMap := g.engine.GetGame().GetMap()
	player := currentMap.Player
	if player.EquippedItem == nil || player.EquippedItem.RangedAttack == core.NoAction {
		return
	}
	if math.Abs(xAxis) < 0.08 && math.Abs(yAxis) < 0.08 { // TODO: this is hacky and can lead to the controls feeling broken
		//g.ToNormalUIState() // removed for now.. let's see what we can do here later
		return
	}

	g.Ui = aimingUIState
	if player.EquippedItem.Scope.FoVinDegrees > 0 {
		player.FovMode = gridmap.FoVModeScoped
		currentMap.UpdateFieldOfView(player)
	}

	playerPos := player.Pos()
	player.LookDirection = geometry.DirectionVectorToAngleInDegreesF(xAxis, yAxis)
	scopeRangeSquared := 0
	scopeRange := 0
	if player.EquippedItem != nil && player.EquippedItem.Scope.Range > 0 {
		scopeRangeSquared = player.EquippedItem.Scope.Range * player.EquippedItem.Scope.Range
		scopeRange = player.EquippedItem.Scope.Range
	}
	losRange := player.VisionRange()
	losRangeSquared := player.VisionRange() * player.VisionRange()
	aimDistance := geometry.IntMax(scopeRange, losRange)
	aimDistanceSquared := geometry.IntMax(scopeRangeSquared, losRangeSquared)
	targetPos := geometry.VectorInDirectionWithLength(playerPos, xAxis, yAxis, aimDistance)
	g.TargetLoS = geometry.LineOfSight(player.FoVSource(), targetPos, func(p geometry.Point) bool {
		return geometry.DistanceSquared(p, player.FoVSource()) <= aimDistanceSquared
	})
}

func (g *GameStateGameplay) AdjustPlayerAim() {
	m := g.engine.GetGame()
	a := m.GetMap().Player
	cam := m.GetCamera()
	losRangeSquared := a.VisionRange() * a.VisionRange()
	mouseInWorld := cam.ScreenToWorld(g.MousePositionOnScreen)
	//cam.CenterOn(mouseInWorld, g.engine.MapWindowWidth(), g.engine.MapWindowHeight())

	a.LookDirection = geometry.DirectionVectorToAngleInDegrees(mouseInWorld.Sub(a.FoVSource()))
	aimDistanceSquared := losRangeSquared
	if a.EquippedItem == nil || (a.EquippedItem.RangedAttack == core.NoAction && a.EquippedItem.MeleeAttack != core.NoAction) {
		// melee
		aimDistanceSquared = 1
	} else if a.EquippedItem.Scope.Range > 0 {
		// scoped
		aimDistanceSquared = a.EquippedItem.Scope.Range * a.EquippedItem.Scope.Range
	}

	g.TargetLoS = geometry.LineOfSight(a.FoVSource(), mouseInWorld, func(p geometry.Point) bool {
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
