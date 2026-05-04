package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

type FollowerMovement struct {
	AIContext
	Leader         *core.Actor
	PosOffset      geometry.Point
	LeaderStartsAt geometry.Point
}

func (f *FollowerMovement) Status() core.ActorState { return core.ActorStatusFollowing }

func (f *FollowerMovement) OnMapLoad(m gridmap.GridMap[*core.Actor, *core.Item, services.Object]) {
	f.Leader = m.ActorAt(f.LeaderStartsAt)
	f.PosOffset = geometry.RotateVector(f.PosOffset, f.Leader.LookDirection)
}

func (f *FollowerMovement) NextAction() core.AIUpdate {
	if f.Leader == nil || f.Leader.IsDowned() {
		return NextUpdateIn(float64(f.Person.MoveDelay()))
	}
	if f.isInPosition(f.Person) && f.canSee(f.Person.Pos(), f.Leader.Pos()) {
		f.watchTheArea(f.Person)
		return NextUpdateIn(float64(f.Person.MoveDelay()))
	}
	targetPos := f.findTargetPosition(f.Person)
	return f.Person.AI.Movement.Action(targetPos, f)
}

func (f *FollowerMovement) OnDestinationReached() core.AIUpdate {
	f.watchTheArea(f.Person)
	return NextUpdateIn(float64(f.Person.MoveDelay()))
}

func (f *FollowerMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(float64(f.Person.MoveDelay()))
}

func (f *FollowerMovement) isInPosition(person *core.Actor) bool {
	return person.Pos() == f.Leader.Pos().Add(f.PosOffset)
}

func (f *FollowerMovement) canSee(sourcePos, targetPos geometry.Point) bool {
	visiblePredicate := func(p geometry.Point) bool {
		return f.Engine.GetGame().GetMap().IsTransparent(p)
	}
	los := geometry.LineOfSight(sourcePos, targetPos, visiblePredicate)
	return los[len(los)-1] == targetPos
}

func (f *FollowerMovement) isPossibleTargetPos(person *core.Actor, targetPos geometry.Point) bool {
	return f.Engine.GetGame().GetMap().CurrentlyPassableAndSafeForActor(person)(targetPos) && f.canSee(targetPos, f.Leader.Pos())
}

func (f *FollowerMovement) watchTheArea(person *core.Actor) {
	person.LookDirection = person.AI.StartLookDirection
}

func (f *FollowerMovement) findTargetPosition(person *core.Actor) geometry.Point {
	offset := f.PosOffset
	leaderDirection := f.Leader.LookDirection
	rotatedOffset := geometry.RotateVector(offset, leaderDirection)
	targetPos := f.Leader.Pos().Add(rotatedOffset)
	if !f.isPossibleTargetPos(person, targetPos) {
		freeTargetPredicate := func(p geometry.Point) bool {
			return f.isPossibleTargetPos(person, p)
		}
		freePos := f.Engine.GetGame().GetMap().GetFreeCellsForDistribution(targetPos, 1, freeTargetPredicate)
		if len(freePos) > 0 {
			targetPos = freePos[0]
		}
	}
	return targetPos
}
