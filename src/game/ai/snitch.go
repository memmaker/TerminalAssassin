package ai

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type SnitchMovement struct {
	AIContext
	KnownGuard       *core.Actor
	incidentToReport core.IncidentReport
}

func (s *SnitchMovement) couldFindGuard() bool {
	person := s.Person
	currentMap := s.Engine.GetGame().GetMap()
	minDist := math.MaxInt
	nearestGuard := (*core.Actor)(nil)

	for _, actor := range currentMap.AllActors {
		if actor.IsAvailableGuard() {
			dist := geometry.DistanceManhattan(actor.Pos(), person.Pos())
			if dist < minDist {
				minDist = dist
				nearestGuard = actor
			}
		}
	}
	s.KnownGuard = nearestGuard
	return nearestGuard != nil
}

func (s *SnitchMovement) NextAction() core.AIUpdate {
	if s.trySnitching() {
		return NextUpdateIn(2)
	}
	if s.nothingToTell() {
		s.Person.AI.PopState()
		return NextUpdateIn(0.1)
	}

	if !s.hasGuardLocation() {
		s.noGuards()
		return NextUpdateIn(0.1)
	}
	return s.Person.AI.Movement.Action(s.KnownGuard.Pos(), s)
}

func (s *SnitchMovement) nothingToTell() bool {
	aic := s.Engine.GetAI()
	currentTick := s.Engine.CurrentTick()
	if s.Person.AI.Knowledge.LastSightingOfDangerousActor.Tick > 0 && currentTick-s.Person.AI.Knowledge.LastSightingOfDangerousActor.Tick < uint64(120*ebiten.TPS()) {
		return false
	}
	if s.incidentToReport == core.EmptyReport {
		s.incidentToReport = aic.GetIncidentForSnitching(s.Person)
		if s.incidentToReport == core.EmptyReport {
			return true
		}
	}
	return false
}

func (s *SnitchMovement) hasGuardLocation() bool {
	if s.KnownGuard == nil || s.KnownGuard.IsDowned() {
		return s.couldFindGuard()
	}
	return true
}

func (s *SnitchMovement) noGuards() {
	person := s.Person
	aic := s.Engine.GetAI()
	println(fmt.Sprintf("%s: NO GUARDS! Panic..?!", person.DebugDisplayName()))
	dangerReports := aic.GetDangerousIncidents(person)
	dangerLocations := make([]geometry.Point, 0)
	for _, report := range dangerReports {
		dangerLocations = append(dangerLocations, report.Location)
	}
	aic.SwitchToPanic(person, dangerLocations)
}

func (s *SnitchMovement) OnDestinationReached() core.AIUpdate {
	person := s.Person
	s.trySnitching()
	return NextUpdateIn(float64(person.MoveDelay()))
}

func (s *SnitchMovement) trySnitching() bool {
	if s.canSpeakToGuard() {
		aic := s.Engine.GetAI()
		person := s.Person
		aic.TransferKnowledge(person, s.KnownGuard)
		aic.SwitchStateBecauseOfNewKnowledge(s.KnownGuard)
		person.AI.PopState()
		person.IsEyeWitness = false
		return true
	}
	return false
}

func (s *SnitchMovement) canSpeakToGuard() bool {
	if s.KnownGuard == nil {
		return false
	}
	isNear := geometry.DistanceManhattan(s.KnownGuard.Pos(), s.Person.Pos()) <= 3
	canSee := s.Person.CanSeeActor(s.KnownGuard)
	isAvailable := s.KnownGuard.IsAvailableGuard()
	return isAvailable && isNear && canSee
}

func (s *SnitchMovement) OnCannotReachDestination() core.AIUpdate {
	person := s.Person
	person.AI.PopState()
	return NextUpdateIn(float64(person.MoveDelay()))
}
