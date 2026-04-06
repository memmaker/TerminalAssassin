package ai

import (
    "math/rand"

    "github.com/memmaker/terminal-assassin/game/core"
)

type ScheduledMovement struct {
    AIContext
}

func (s *ScheduledMovement) NextAction() core.AIUpdate {
    aic := s.Engine.GetAI()
    if aic.IncidentsNeedCleanup(s.Person) && s.Person.Type == core.ActorTypeGuard {
        aic.SwitchToCleanup(s.Person)
        return NextUpdateIn(rand.Float64() + 1.0)
    }
    s.Person.Status = core.ActorStatusOnSchedule

    schedule := s.Engine.GetGame().GetMap().GetSchedule(s.Person.AI.Schedule)
    if schedule == nil || len(schedule.Tasks) == 0 {
        return NextUpdateIn(5)
    }

    task := s.Person.AI.CurrentTask(schedule)

    return s.Person.AI.Movement.Action(task.Location, s)
}

func (s *ScheduledMovement) OnDestinationReached() core.AIUpdate {
    ai := s.Person.AI
    engine := s.Engine
    aic := engine.GetAI()

    schedule := engine.GetGame().GetMap().GetSchedule(ai.Schedule)
    if schedule == nil || len(schedule.Tasks) == 0 {
        return NextUpdateIn(5)
    }

    previousTaskIndex := ai.CurrentTaskIndex

    advanceTask := func() { ai.AdvanceTask(schedule) }

    if !aic.TryContextActionAtTaskLocation(s.Person, advanceTask) {
        animationCompleted := false
        completedFunc := func() {
            animationCompleted = true
            ai.AdvanceTask(schedule)
        }
        animator := engine.GetAnimator()
        until := func() bool {
            return animationCompleted
        }
        aic.SetEngaged(s.Person, core.ActorStatusEngaged, until)
        currentTask := ai.CurrentTask(schedule)

        animator.TaskAnimation(s.Person, currentTask.DurationInSeconds, currentTask.LookDirections, completedFunc, completedFunc)
    }
    return DeferredUpdate(func() bool {
        return s.Person.AI.CurrentTaskIndex != previousTaskIndex || s.Person.Status != core.ActorStatusEngaged
    })
}

// OnCannotReachDestination We just wait for 5 seconds and then try again
func (s *ScheduledMovement) OnCannotReachDestination() core.AIUpdate {
    return NextUpdateIn(5)
}
