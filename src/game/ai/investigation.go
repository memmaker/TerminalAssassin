package ai

import (
	"fmt"
	"time"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type InvestigationMovement struct {
	AIContext
	LookAroundCounter   int
	Incident            core.IncidentReport
	ReactionTimeAwaited bool
}

func (i *InvestigationMovement) Status() core.ActorState { return core.ActorStatusInvestigating }

func (i *InvestigationMovement) NextAction() core.AIUpdate {
	person := i.Person
	engine := i.Engine

	// Discard stale suspicious-sighting investigations (2 in-game hours old)
	if i.Incident.Type.IsSuspiciousLocation() {
		gameTime := engine.CurrentGameTime()
		if gameTime.Sub(i.Incident.Time) > 2*time.Hour {
			aic := engine.GetAI()
			aic.MarkAsDone(person, i.Incident)
			aic.UntrackInvestigation(i.Incident.Hash())
			person.AI.PopState()
			return NextUpdateIn(0.3)
		}
	}
	return person.AI.Movement.Action(i.Incident.Location, i)
}

func (i *InvestigationMovement) OnDestinationReached() core.AIUpdate {
	person := i.Person
	engine := i.AIContext.Engine
	aic := engine.GetAI()

	person.TurnLeft(45)
	aic.UpdateVision(person)
	i.LookAroundCounter++
	if i.LookAroundCounter <= 7 {
		return NextUpdateIn(0.5)
	}
	if i.Incident.Type.IsContact() {
		if person.AI.Knowledge.LastSightingOfDangerous.Location == i.Incident.Location {
			person.AI.Knowledge.LastSightingOfDangerous.HandledByMe = true
			person.AI.Knowledge.LastSightingOfDangerous.Time = time.Time{}
		}
		println(fmt.Sprintf("%s FINISHED HANDLING contact. Could not confirm sighting of '%s'", person.Name, i.Incident.Type))
	} else if i.Incident.Type.IsEnvironmentalToggle() {
		distance := geometry.DistanceManhattan(person.Pos(), i.Incident.Location)
		currentMap := engine.GetGame().GetMap()
		if distance > 1 {
			dest := currentMap.GetNearestWalkableNeighbor(person.Pos(), i.Incident.Location)
			return person.AI.Movement.Action(dest, i)
		} else {
			objectAt := currentMap.ObjectAt(i.Incident.Location)
			if objectAt != nil && objectAt.IsActionAllowed(engine, person) {
				objectAt.Action(engine, person)
			}
		}
	}
	aic.MarkAsDone(person, i.Incident)
	aic.UntrackInvestigation(i.Incident.Hash())
	person.AI.PopState()
	return NextUpdateIn(0.5)
}

func (i *InvestigationMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(3.0)
}
