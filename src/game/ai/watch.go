package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type WatchMovement struct {
	AIContext
	suspiciousActor   *core.Actor
	incident          core.IncidentReport
	lastKnownLocation geometry.Point
	chaseCounter      int
}

func (v *WatchMovement) OnDestinationReached() core.AIUpdate {
	return NextUpdateIn(1)
}

func (v *WatchMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(4)
}

func (v *WatchMovement) NextAction() core.AIUpdate {
	person := v.Person
	aic := v.Engine.GetAI()
	//println(fmt.Sprintf("%s: next watch action", person.DebugDisplayName()))
	person.Status = core.ActorStatusWatching
	// can we see the suspicious actor? -> raise suspicion
	// no -> investigate
	if v.suspiciousActor == nil || v.suspiciousActor.IsDowned() {
		person.AI.PopState()
		return NextUpdateIn(1)
	}

	if person.CanSeeActor(v.suspiciousActor) {
		v.lastKnownLocation = v.suspiciousActor.Pos()
		currentMap := v.Engine.GetGame().GetMap()
		currentMap.IsTrespassing(person)
		suspicionDelayInMS := v.getSuspicionDelayInMS()

		if v.incident.Type.IsTrespassing() && !currentMap.IsTrespassing(v.suspiciousActor) {
			if person.AI.SuspicionCounter == 0 {
				person.AI.PopState()
				return NextUpdateIn(1)
			}
			person.AI.LowerSuspicion()
		} else {
			aic.RaiseSuspicionAt(person, v.suspiciousActor, suspicionDelayInMS)
		}

		return NextUpdateIn(float64(suspicionDelayInMS / 1000.0))
	}

	person.AI.LowerSuspicion()
	v.chaseCounter++
	if v.chaseCounter > 20 || person.AI.SuspicionCounter == 0 {
		person.AI.SuspicionCounter = 0
		person.AI.PopState()
		return NextUpdateIn(1)
	}
	return person.AI.Movement.Action(v.lastKnownLocation, v)
}

func (v *WatchMovement) getSuspicionDelayInMS() int {

	if v.incident.Type == core.ObservationTrespassingInHostileZone {
		return 200
	}
	if v.incident.Type == core.ObservationIllegalAction {
		return 300
	}
	if v.incident.Type == core.ObservationNearActiveIllegalIncident {
		return 450
	}
	if v.incident.Type == core.ObservationOpenCarry {
		return 600
	}

	return 800
}
