package ai

import (
	"math/rand"

	"github.com/memmaker/terminal-assassin/game/core"
)

type GuardMovement struct {
	AIContext
}

func (u *GuardMovement) OnDestinationReached() core.AIUpdate {
	ai := u.Person.AI
	aic := u.Engine.GetAI()
	u.Person.LookDirection = ai.StartLookDirection
	if aic.IncidentsNeedCleanup(u.Person) && u.Person.Type == core.ActorTypeGuard {
		aic.SwitchToCleanup(u.Person)
		return NextUpdateIn(rand.Float64() + 1.0)
	}
	aic.UpdateVision(u.Person)
	return NextUpdateIn(rand.Float64() + 1.0)
}

func (u *GuardMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(rand.Float64() + 4.0)
}

func (u *GuardMovement) NextAction() core.AIUpdate {
	ai := u.Person.AI
	u.Person.Status = core.ActorStatusIdle
	return u.Person.AI.Movement.Action(ai.StartPosition, u)
}
