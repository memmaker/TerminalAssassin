package ai

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

type VomitMovement struct {
	AIContext
	chosenToilet geometry.Point
	toiletFound  bool
	vomitCounter int
}

func (v *VomitMovement) OnDestinationReached() core.AIUpdate {
	return v.startVomiting()
}

func (v *VomitMovement) OnCannotReachDestination() core.AIUpdate {
	v.Person.AI.PopState()
	return NextUpdateIn(1)
}

func (v *VomitMovement) NextAction() core.AIUpdate {
	println(fmt.Sprintf("%s: next vomit action", v.Person.DebugDisplayName()))

	currentMap := v.Engine.GetGame().GetMap()
	if !v.toiletFound {
		nearestToilet := currentMap.GetNearestSpecialTile(v.Person.Pos(), gridmap.SpecialTileToilet)
		v.chosenToilet = currentMap.GetRandomFreeNeighbor(nearestToilet)
		v.toiletFound = true
	}

	return v.Person.AI.Movement.Action(v.chosenToilet, v)
}

func (v *VomitMovement) startVomiting() core.AIUpdate {
	person := v.Person
	currentMap := v.AIContext.Engine.GetGame().GetMap()
	animator := v.AIContext.Engine.GetAnimator()
	previousVomitCount := v.vomitCounter
	completed := func() {
		v.vomitCounter++
		person.AI.PopState()
	}
	aic := v.AIContext.Engine.GetAI()
	until := func() bool {
		return previousVomitCount != v.vomitCounter
	}
	aic.SetEngaged(person, core.ActorStatusVomiting, until)
	animator.VomitingAnimation(person, currentMap.GetNeighborWithSpecial(person.Pos(), gridmap.SpecialTileToilet), completed)
	return DeferredUpdate(until)
}
