package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/rng"
)

type CombatMovement struct {
	AIContext
	Target            *core.Actor
	isAimingAt        geometry.Point
	LastKnownPosition *geometry.Point
	stepCounter       int
	noShotCounter     int // ticks where target is visible but shot is blocked
}

func (t *CombatMovement) Status() core.ActorState { return core.ActorStatusCombat }

func (t *CombatMovement) OnDestinationReached() core.AIUpdate {
	return NextUpdateIn(0.4)
}

func (t *CombatMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(0.4)
}

func (t *CombatMovement) NextAction() core.AIUpdate {
	person := t.Person
	aic := t.Engine.GetAI()
	if t.Target.IsDowned() || !person.EquipWeapon() || t.stepCounter > 50 || t.noShotCounter > 30 {
		t.exitCombat(person)
		return NextUpdateIn(0.4)
	}
	currentMap := t.Engine.GetGame().GetMap()

	if person.CanSeeActor(t.Target) && currentMap.IsPathPassableForProjectile(person.Pos(), t.Target.Pos()) {
		targetPosition := t.Target.Pos()
		t.LastKnownPosition = &targetPosition
		t.noShotCounter = 0
		return t.handleFiring(person, aic)
	}
	if person.CanSeeActor(t.Target) {
		targetPosition := t.Target.Pos()
		t.LastKnownPosition = &targetPosition
		t.stepCounter = 0
		t.noShotCounter++
	} else {
		t.stepCounter++
		t.noShotCounter = 0
	}
	return person.AI.Movement.Action(currentMap.GetRandomFreeNeighbor(*t.LastKnownPosition), t)
}

// exitCombat cleans up the combat state.
// Pops Combat, then pushes Investigation for the last known position so the
// guard searches where the target was last seen (§5).
func (t *CombatMovement) exitCombat(person *core.Actor) {
	ai := person.AI
	aic := t.Engine.GetAI()
	ai.PopState()

	if t.LastKnownPosition != nil && !person.IsInvestigating() {
		lkp := *t.LastKnownPosition
		report := core.IncidentReport{
			Type:     core.ObservationCombatSeen,
			Location: lkp,
			Time:     t.Engine.CurrentGameTime(),
		}
		aic.SwitchToInvestigation(person, report)
	}
}

func (t *CombatMovement) handleFiring(person *core.Actor, aic services.AIInterface) core.AIUpdate {
	t.LastKnownPosition = &geometry.Point{X: t.Target.Pos().X, Y: t.Target.Pos().Y}
	t.stepCounter = 0
	currentMap := t.Engine.GetGame().GetMap()
	person.LookDirection = geometry.DirectionVectorToAngleInDegrees(t.Target.Pos().Sub(person.Pos()))
	aimedShot := rng.R.Intn(100) < 50
	if person.EquippedItem.OnCooldown || (t.isAimingAt != t.Target.Pos() && aimedShot) {
		t.isAimingAt = t.Target.Pos()
	} else {
		game := t.Engine.GetGame()
		actions := game.GetActions()
		actions.UseEquippedItemAtRange(person, t.Target.Pos())
		if person.HasBurstWeaponEquipped() {
			t.Engine.Schedule(0.066, func() {
				actions.UseEquippedItemAtRange(person, currentMap.RandomPosAround(t.Target.Pos()))
			})
			t.Engine.Schedule(0.136, func() {
				actions.UseEquippedItemAtRange(person, currentMap.RandomPosAround(t.Target.Pos()))
			})
		}
	}
	return NextUpdateIn(rng.R.Float64()*0.5 + 0.5)
}
