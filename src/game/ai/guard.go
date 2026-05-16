package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/rng"
)

type GuardMovement struct {
	AIContext
}

func (u *GuardMovement) Status() core.ActorState { return core.ActorStatusIdle }

func (u *GuardMovement) OnDestinationReached() core.AIUpdate {
	ai := u.Person.AI
	aic := u.Engine.GetAI()
	u.Person.LookDirection = ai.StartLookDirection
	aic.UpdateVision(u.Person)
	return NextUpdateIn(rng.R.Float64() + 1.0)
}

func (u *GuardMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(rng.R.Float64() + 4.0)
}

func (u *GuardMovement) NextAction() core.AIUpdate {
	ai := u.Person.AI
	return u.Person.AI.Movement.Action(ai.StartPosition, u)
}
