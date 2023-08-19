package ai

import (
	"math/rand"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type CombatMovement struct {
	AIContext
	Target            *core.Actor
	isAimingAt        geometry.Point
	LastKnownPosition *geometry.Point
	stepCounter       int
}

func (t *CombatMovement) OnDestinationReached() core.AIUpdate {
	return NextUpdateIn(0.4)
}

func (t *CombatMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(0.4)
}

func (t *CombatMovement) NextAction() core.AIUpdate {
	person := t.Person
	aic := t.Engine.GetAI()
	//println(fmt.Sprintf("%s: next combat action", person.DebugDisplayName()))
	person.Status = core.ActorStatusCombat
	if t.Target.IsDowned() || !person.EquipWeapon() || t.stepCounter > 50 {
		person.AI.PopState()
		return NextUpdateIn(0.4)
	}
	currentMap := t.Engine.GetGame().GetMap()

	if person.CanSeeActor(t.Target) && currentMap.IsPathPassableForProjectile(person.Pos(), t.Target.Pos()) {
		targetPosition := t.Target.Pos()
		t.LastKnownPosition = &targetPosition
		return t.handleFiring(person, aic)
	}
	t.stepCounter++
	return person.AI.Movement.Action(currentMap.GetRandomFreeNeighbor(*t.LastKnownPosition), t)
}

func (t *CombatMovement) handleFiring(person *core.Actor, aic services.AIInterface) core.AIUpdate {
	t.LastKnownPosition = &geometry.Point{X: t.Target.Pos().X, Y: t.Target.Pos().Y}
	t.stepCounter = 0
	currentMap := t.Engine.GetGame().GetMap()
	person.LookDirection = geometry.DirectionVectorToAngleInDegrees(t.Target.Pos().Sub(person.Pos()))
	aimedShot := rand.Intn(100) < 50
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
	return NextUpdateIn(rand.Float64()*0.5 + 0.5)
}
