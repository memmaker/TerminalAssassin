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
	KnownGuard *core.Actor
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
	currentTick := s.Engine.CurrentTick()
	sighting := s.Person.AI.Knowledge.LastSightingOfDangerous
	return sighting.Tick == 0 || sighting.HandledByMe || currentTick-sighting.Tick >= uint64(120*ebiten.TPS())
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
	dangerLocations := []geometry.Point{person.AI.Knowledge.LastSightingOfDangerous.Location}
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
