package ai

import (
	"math/rand"

	"github.com/memmaker/terminal-assassin/game/core"
)

type ScheduledMovement struct {
	AIContext
}

func (s *ScheduledMovement) NextAction() core.AIUpdate {
	//println(fmt.Sprintf("%s: next schedule action", s.Person.DebugDisplayName()))
	aic := s.Engine.GetAI()
	if aic.IncidentsNeedCleanup(s.Person) && s.Person.Type == core.ActorTypeGuard {
		aic.SwitchToCleanup(s.Person)
		return NextUpdateIn(rand.Float64() + 1.0)
	}
	s.Person.Status = core.ActorStatusOnSchedule
	return s.Person.AI.Movement.Action(s.Person.AI.Schedule.CurrentTask().Location, s)
}

func (s *ScheduledMovement) OnDestinationReached() core.AIUpdate {
	ai := s.Person.AI
	engine := s.Engine
	aic := engine.GetAI()
	previousTaskIndex := ai.Schedule.CurrentTaskIndex

	if !aic.TryContextActionAtTaskLocation(s.Person, s.Person.AI.Schedule.NextTask) {
		animationCompleted := false
		completedFunc := func() {
			animationCompleted = true
			// popping should not happen outside of state..
			s.Person.AI.Schedule.NextTask()
		}
		animator := engine.GetAnimator()
		until := func() bool {
			return animationCompleted
		}
		aic.SetEngaged(s.Person, core.ActorStatusEngaged, until) // this will push a new state
		// we really want that WaitState to pop itself once the animation is done
		animator.TaskAnimation(s.Person, ai.Schedule.CurrentTask().DurationInSeconds, completedFunc, completedFunc)
	}
	return DeferredUpdate(func() bool {
		return s.Person.AI.Schedule.CurrentTaskIndex != previousTaskIndex || s.Person.Status != core.ActorStatusEngaged
	})
}

// OnCannotReachDestination We just wait for 5 seconds and then try again
func (s *ScheduledMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(5)
}
