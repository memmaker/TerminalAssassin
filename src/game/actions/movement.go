package actions

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/ai"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Movement struct {
	desiredLocation *geometry.Point
	Engine          services.Engine
	Person          *core.Actor
}

// Action this call will spend this action moving towards the given location
// if that is not possible, it will call the appropriate callback, expecting the action to be handled there.
func (m *Movement) Action(location geometry.Point, handler core.MoveHandler) core.AIUpdate {
	if m.desiredLocation == nil || *m.desiredLocation != location {
		m.desiredLocation = &location
		//println(fmt.Sprintf("%s: new destination %v", m.Person.DebugDisplayName(), location))
	}
	targetLocation := *m.desiredLocation
	if !m.Engine.GetGame().GetMap().CurrentlyPassableAndSafeForActor(m.Person)(*m.desiredLocation) {
		targetLocation = m.Engine.GetGame().GetMap().GetNearestWalkableNeighbor(m.Person.Pos(), *m.desiredLocation)
	}

	if m.Person.Pos() == targetLocation {
		//println(fmt.Sprintf("%s: destination reached (%v)", m.Person.DebugDisplayName(), location))
		return handler.OnDestinationReached()
	}

	if m.Person.HasPathTo(targetLocation) {
		//println(fmt.Sprintf("%s: continuing path to %v", m.Person.DebugDisplayName(), location))
		aic := m.Engine.GetAI()
		return aic.MoveOnPath(m.Person)
	}

	return m.tryFindPath(handler.OnCannotReachDestination)
}
func (m *Movement) OnBlockedPath() core.AIUpdate {
	println(fmt.Sprintf("%s: my path to %v is blocked", m.Person.DebugDisplayName(), *m.desiredLocation))
	return m.tryFindPath(func() core.AIUpdate {
		println(fmt.Sprintf("%s: CANNOT REACH destination", m.Person.DebugDisplayName()))
		return ai.NextUpdateIn(4)
	})
}

func (m *Movement) tryFindPath(onCannotReachDestination func() core.AIUpdate) core.AIUpdate {
	aic := m.Engine.GetAI()
	currentMap := m.Engine.GetGame().GetMap()
	//println(fmt.Sprintf("%s: trying to find path to %v", m.Person.DebugDisplayName(), *m.desiredLocation))
	if aic.PathSet(m.Person, *m.desiredLocation, currentMap.CurrentlyPassableAndSafeForActor(m.Person)) {
		//println(fmt.Sprintf("%s: found new path to %v", m.Person.DebugDisplayName(), *m.desiredLocation))
		return aic.MoveOnPath(m.Person)
	}
	println(fmt.Sprintf("%s: CANNOT FIND PATH to %v", m.Person.DebugDisplayName(), *m.desiredLocation))
	return onCannotReachDestination()
}

func (m *Movement) isAtLocation(person *core.Actor, point geometry.Point) bool {
	return person.Pos() == point
}

func (m *Movement) isNextToDesiredLocation(person *core.Actor, maxDistance int) bool {
	if m.desiredLocation == nil {
		return false
	}
	return person.CanSee(*m.desiredLocation) && geometry.DistanceManhattan(person.Pos(), *m.desiredLocation) <= maxDistance
}
