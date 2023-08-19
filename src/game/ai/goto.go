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

func (g *GotoBehaviour) OnDestinationReached() core.AIUpdate {
	// pop
	g.Person.AI.PopState()
	if g.CallOnArrival != nil {
		g.CallOnArrival()
	}
	return NextUpdateIn(1)
}

func (g *GotoBehaviour) OnCannotReachDestination() core.AIUpdate {
	// just try again for now..
	println("CANNOT reach destination using (goto) behaviour")
	return NextUpdateIn(1)
}

func (g *GotoBehaviour) NextAction() core.AIUpdate {
	aic := g.Engine.GetAI()
	aic.UpdateVision(g.Person)
	println(fmt.Sprintf("%s is moving to %s (goto)", g.Person.Name, g.TargetLocation))
	return g.Person.AI.Movement.Action(g.TargetLocation, g)
}
