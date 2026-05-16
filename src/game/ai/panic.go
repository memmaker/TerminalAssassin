package ai

import (
	"time"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

// panicTimeout: 24 in-game hours (TimeOfDay-based).
const panicTimeout = 24 * time.Hour

type PanicMovement struct {
	AIContext
	DangerousLocations []geometry.Point
	ThreatActor        *core.Actor
	failedRetryCount   int
	startTime          time.Time
	usingExitFallback  bool
}

func (p *PanicMovement) Status() core.ActorState { return core.ActorStatusPanic }

func (p *PanicMovement) NextAction() core.AIUpdate {
	person := p.Person
	engine := p.Engine
	ai := person.AI
	ai.SuspicionCounter = 0

	// Initialise start time on first call
	if p.startTime.IsZero() {
		p.startTime = engine.CurrentGameTime()
	}

	// 24 in-game hours timeout → set alert flag and pop
	if engine.CurrentGameTime().Sub(p.startTime) >= panicTimeout {
		engine.GetAI().SetAlerted(person)
		person.AI.PopState()
		return NextUpdateIn(0.3)
	}

	return p.runAway()
}

func (p *PanicMovement) runAway() core.AIUpdate {
	person := p.Person
	moveDelta := geometry.Point{}
	game := p.Engine.GetGame()
	currentMap := game.GetMap()

	if p.usingExitFallback {
		// Already pathfinding to exit/corner — let the movement system handle it
		return person.AI.Movement.Action(p.getExitOrFarCorner(), p)
	}

	possibleMoves := currentMap.GetFilteredCardinalNeighbors(person.Pos(), currentMap.CurrentlyPassableForActor(person))
	if len(possibleMoves) == 0 {
		p.failedRetryCount++
		if p.failedRetryCount >= 10 {
			p.usingExitFallback = true
			p.failedRetryCount = 0
		}
		return NextUpdateIn(float64(person.MoveDelay()))
	}
	p.failedRetryCount = 0

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

// getExitOrFarCorner returns the nearest SpecialTilePlayerExit position,
// or the map corner farthest from all danger locations if no exit exists.
func (p *PanicMovement) getExitOrFarCorner() geometry.Point {
	person := p.Person
	game := p.Engine.GetGame()
	gmap := game.GetMap()
	bestDist := int(^uint(0) >> 1)
	exitPos := geometry.Point{X: -1}
	for y := 0; y < gmap.MapHeight; y++ {
		for x := 0; x < gmap.MapWidth; x++ {
			pt := geometry.Point{X: x, Y: y}
			if gmap.IsTileWithSpecialAt(pt, gridmap.SpecialTilePlayerExit) {
				d := geometry.DistanceManhattan(person.Pos(), pt)
				if d < bestDist {
					bestDist = d
					exitPos = pt
				}
			}
		}
	}
	if exitPos.X >= 0 {
		return exitPos
	}
	// No exit defined — pick map corner farthest from danger
	corners := []geometry.Point{
		{X: 0, Y: 0},
		{X: gmap.MapWidth - 1, Y: 0},
		{X: 0, Y: gmap.MapHeight - 1},
		{X: gmap.MapWidth - 1, Y: gmap.MapHeight - 1},
	}
	bestCorner := corners[0]
	maxDist := 0.0
	for _, c := range corners {
		sum := 0.0
		for _, danger := range p.DangerousLocations {
			sum += geometry.Distance(c, danger)
		}
		if sum > maxDist {
			maxDist = sum
			bestCorner = c
		}
	}
	return bestCorner
}

func (p *PanicMovement) OnDestinationReached() core.AIUpdate {
	p.usingExitFallback = false // reached exit; let flee logic resume
	return NextUpdateIn(float64(p.Person.MoveDelay()))
}

func (p *PanicMovement) OnCannotReachDestination() core.AIUpdate {
	p.usingExitFallback = false
	return NextUpdateIn(float64(p.Person.MoveDelay()))
}
