package ai

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/core"
)

type ScriptedState struct {
	AIContext
}

func (s *ScriptedState) OnDestinationReached() core.AIUpdate {
	return NextUpdateIn(1)
}

func (s *ScriptedState) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(1)
}

func (s *ScriptedState) NextAction() core.AIUpdate {
	person := s.Person

	person.Status = core.ActorStatusScripted

	isInConversation := person.Dialogue.Active(s.Engine.CurrentTick())

	if isInConversation && person.Dialogue.Situation != nil {
		if person.Pos() != person.Dialogue.Situation.Location {
			println(fmt.Sprintf("%s is not at the correct location for the conversation, moving.", person.Name))
			return person.AI.Movement.Action(person.Dialogue.Situation.Location, s)
		}
		if person.LookDirection != person.Dialogue.Situation.Direction {
			person.LookDirection = person.Dialogue.Situation.Direction
		}
	}

	action, actionIsFromQueue := person.Script.GetCurrentAction(person.Pos())

	if action == nil {
		aic := s.Engine.GetAI()
		aic.UpdateVision(person)
		return NextUpdateIn(1)
	}
	//println(fmt.Sprintf("%s executing current scripted action: '%s'", person.Name, action.ToString()))
	update := action.Execute(person)
	if action.IsDone(person) && actionIsFromQueue {
		println(fmt.Sprintf("%s finished this action: '%s'", person.Name, action.ToString()))
		println(fmt.Sprintf("%s is queuing the next scripted action", person.Name))
		person.Script.Next()
		if person.Script.IsFinished() {
			person.Script.Clear()
			println(fmt.Sprintf("%s FINISHED ALL QUEUED scripted actions, clearing script for actor.", person.Name))
			return update
		}
	}
	return update
}
