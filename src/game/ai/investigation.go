package ai

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type InvestigationMovement struct {
	AIContext
	LookAroundCounter   int
	Incident            core.IncidentReport
	ReactionTimeAwaited bool
}

func (i *InvestigationMovement) NextAction() core.AIUpdate {
	person := i.Person
	person.Status = core.ActorStatusInvestigating

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
		if geometry.DistanceChebyshev(i.Incident.Location, person.AI.Knowledge.LastSightingOfDangerous.Location) <= 3 {
			person.AI.Knowledge.LastSightingOfDangerous.HandledByMe = true
			person.AI.Knowledge.LastSightingOfDangerous.Tick = 0
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
	person.AI.PopState()
	return NextUpdateIn(0.5)
}

func (i *InvestigationMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(3.0)
}
