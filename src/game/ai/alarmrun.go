package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

// AlarmRunMovement sends a guard to the nearest active alarm to trigger it.
// Stack layout when pushed: [..., Investigation, AlarmRun]
// After triggering the alarm, AlarmRun pops itself; Investigation runs next.
type AlarmRunMovement struct {
	AIContext
	Incident    core.IncidentReport
	TargetAlarm services.Object // cached alarm target; re-searched if nil or no longer active
}

func (a *AlarmRunMovement) Status() core.ActorState { return core.ActorStatusAlarmRun }

func (a *AlarmRunMovement) NextAction() core.AIUpdate {
	person := a.Person

	if a.TargetAlarm == nil || !a.isTargetValid() {
		alarm := a.findNearestActiveAlarm()
		if alarm == nil {
			// All alarms gone — pop AlarmRun, Investigation below takes over
			person.AI.PopState()
			return NextUpdateIn(0.3)
		}
		a.TargetAlarm = alarm
	}
	return person.AI.Movement.Action(a.TargetAlarm.Pos(), a)
}

func (a *AlarmRunMovement) OnDestinationReached() core.AIUpdate {
	person := a.Person
	engine := a.Engine
	if a.TargetAlarm == nil {
		person.AI.PopState()
		return NextUpdateIn(0.3)
	}
	// Trigger the alarm
	if dev, ok := a.TargetAlarm.(services.AlarmDevice); ok && dev.IsActiveAlarm() {
		dev.TriggerAlarm(engine, a.Incident.Location)
	}
	person.AI.PopState()
	return NextUpdateIn(0.3)
}

func (a *AlarmRunMovement) OnCannotReachDestination() core.AIUpdate {
	// Try another alarm
	a.TargetAlarm = nil
	return NextUpdateIn(0.5)
}

func (a *AlarmRunMovement) isTargetValid() bool {
	dev, ok := a.TargetAlarm.(services.AlarmDevice)
	return ok && dev.IsActiveAlarm()
}

func (a *AlarmRunMovement) findNearestActiveAlarm() services.Object {
	currentMap := a.Engine.GetGame().GetMap()
	person := a.Person
	bestDist := int(^uint(0) >> 1)
	var best services.Object
	for _, obj := range currentMap.AllObjects {
		dev, ok := obj.(services.AlarmDevice)
		if !ok || !dev.IsActiveAlarm() {
			continue
		}
		d := geometry.DistanceManhattan(person.Pos(), obj.Pos())
		if d < bestDist {
			bestDist = d
			best = obj
		}
	}
	return best
}


