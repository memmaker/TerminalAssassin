package ai

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type GotoBehaviour struct {
	AIContext
	TargetLocation geometry.Point
	CallOnArrival  func()
}

// Status returns the status of the state below this one on the stack so that
// the actor's semantic state remains unchanged while travelling.
func (g *GotoBehaviour) Status() core.ActorState {
	if below := g.Person.AI.PeekBelow(); below != nil {
		return below.Status()
	}
	return core.ActorStatusIdle
}

func (g *GotoBehaviour) OnDestinationReached() core.AIUpdate {
	g.Person.AI.PopState()
	if g.CallOnArrival != nil {
		g.CallOnArrival()
	}
	return NextUpdateIn(1)
}

func (g *GotoBehaviour) OnCannotReachDestination() core.AIUpdate {
	println("CANNOT reach destination using (goto) behaviour")
	return NextUpdateIn(1)
}

func (g *GotoBehaviour) NextAction() core.AIUpdate {
	aic := g.Engine.GetAI()
	aic.UpdateVision(g.Person)
	if g.Person.DebugFlag {
		println(fmt.Sprintf("%s is moving to %s (goto)", g.Person.Name, g.TargetLocation))
	}
	return g.Person.AI.Movement.Action(g.TargetLocation, g)
}
