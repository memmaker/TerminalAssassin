package ai

import (
	"math"

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

func (f *FollowerMovement) OnMapLoad(m gridmap.GridMap[*core.Actor, *core.Item, services.Object]) {
	f.Leader = m.ActorAt(f.LeaderStartsAt)
	f.PosOffset = geometry.RotateVector(f.PosOffset, f.Leader.LookDirection)
}

func (f *FollowerMovement) NextAction() {
	// the plan:
	// the follower has a position he wants to be at relative to the Leader
	// the follower position must always have LoS to the Leader
	// first we need to determine if the position can be occupied at all
	// we are satisfied if the follower is at the right position and can see the Leader
	if f.isInPosition(f.Person) && f.canSee(f.Person.Pos(), f.Leader.Pos()) {
		f.watchTheArea(f.Person)
	} else {
		f.Replan(f.Person)
	}
}
func (f *FollowerMovement) isInPosition(person *core.Actor) bool {
	targetPos := f.Leader.Pos().Add(f.PosOffset)
	if person.Pos() == targetPos {
		return true
	}
	return false
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
	ai := person.AI
	person.LookDirection = ai.StartLookDirection
	//aic.SendAINext(person, 0.7)
	//aic.ScheduleUpdateFor(person, 0.7)
}

func (f *FollowerMovement) Replan(person *core.Actor) {
	//currentMap := f.Engine.GetGame().GetMap()
	//targetPos := f.findTargetPosition(person)
	/*
		if f.isInPosition(person) && f.canSee(person.Pos(), f.Leader.Pos()) || targetPos == person.Pos() {
			f.watchTheArea(person)
		} else if f.canSee(person.Pos(), targetPos) && f.isPossibleTargetPos(aic, person, targetPos) {
			f.moveTowards(person, targetPos)
		} else {
			aic.PathSet(person, targetPos, currentMap.CurrentlyPassableAndSafeForActor(person))
			aic.MoveOnPath(person)
		}

	*/
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
		} else {
			// TODO: what to do if there is no free position?
			//			Log("no free position for follower")
		}
	}
	return targetPos
}

func (f *FollowerMovement) moveTowards(person *core.Actor, targetPos geometry.Point) {
	if person.Pos() == targetPos {
		return
	}
	moveDelta := geometry.Point{}
	directionVector := targetPos.Sub(person.Pos())
	if math.Abs(float64(directionVector.X)) > math.Abs(float64(directionVector.Y)) {
		// move in X direction
		if directionVector.X > 0 {
			moveDelta.X = 1
		} else {
			moveDelta.X = -1
		}
	} else {
		// move in Y direction
		if directionVector.Y > 0 {
			moveDelta.Y = 1
		} else {
			moveDelta.Y = -1
		}
	}
	currentMap := f.Engine.GetGame().GetMap()
	movePos := person.Pos().Add(moveDelta)

	if !currentMap.CurrentlyPassableForActor(person)(movePos) {
		//Log("direct move not possible, replanning")
		/*
			aic.PathSet(person, targetPos, currentMap.CurrentlyPassableAndSafeForActor(person))
			aic.MoveOnPath(person)
			return person.AI.Movement.Action(targetPos, f)

		*/
	}

	person.Move.Delta = moveDelta
	currentMap.MoveActor(person, movePos)
	//aic.AIMovementCommand(person, person.Move.Delta)
	//aic.ScheduleUpdateFor(person, float64(person.MoveDelay()))
}
