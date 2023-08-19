package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type PanicMovement struct {
	AIContext
	DangerousLocations []geometry.Point
	ThreatActor        *core.Actor
}

func (p *PanicMovement) NextAction() core.AIUpdate {
	//println(fmt.Sprintf("%s: next panic action", person.DebugDisplayName()))
	person := p.Person
	ai := person.AI
	ai.SuspicionCounter = 0
	person.Status = core.ActorStatusPanic
	return p.runAway()
}

func (p *PanicMovement) runAway() core.AIUpdate {
	person := p.Person
	moveDelta := geometry.Point{}
	game := p.Engine.GetGame()
	currentMap := game.GetMap()
	possibleMoves := currentMap.GetFilteredCardinalNeighbors(person.Pos(), currentMap.CurrentlyPassableForActor(person))
	maxDistance := 0.0
	for _, move := range possibleMoves {
		summedDistance := 0.0
		for _, danger := range p.DangerousLocations {
			summedDistance += geometry.Distance(move, danger)
		}
		if p.ThreatActor != nil {
			summedDistance += geometry.Distance(move, p.ThreatActor.Pos()) * 2
		}
		if summedDistance > maxDistance {
			maxDistance = summedDistance
			moveDelta = move.Sub(person.Pos())
		}
	}
	person.LookDirection = geometry.DirectionVectorToAngleInDegrees(moveDelta)
	movePos := person.Pos().Add(moveDelta)
	person.Move.Delta = moveDelta
	game.MoveActor(person, movePos)
	return NextUpdateIn(float64(person.MoveDelay() * 0.6))
}
