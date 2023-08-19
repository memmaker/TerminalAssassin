package core

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/geometry"
)

type ScriptedAction interface {
	IsDone(person *Actor) bool
	Execute(person *Actor) AIUpdate
	ToString() string
}
type ScriptComponent struct {
	queuedActions              []ScriptedAction
	currentAction              int
	preferredLocation          geometry.Point
	stayInPreferredLocation    bool
	defaultMoveActionGenerator func(geometry.Point) ScriptedAction
}

func (s *ScriptComponent) SetDefaultMoveActionGenerator(generator func(geometry.Point) ScriptedAction) {
	s.defaultMoveActionGenerator = generator
}

func (s *ScriptComponent) AddAction(action ScriptedAction) {
	s.queuedActions = append(s.queuedActions, action)
}

func (s *ScriptComponent) GetCurrentAction(actorLocation geometry.Point) (ScriptedAction, bool) {
	if s.stayInPreferredLocation && s.preferredLocation != actorLocation && s.defaultMoveActionGenerator != nil {
		return s.defaultMoveActionGenerator(s.preferredLocation), false
	}
	if len(s.queuedActions) == 0 {
		return nil, false
	}
	return s.queuedActions[s.currentAction], true
}

func (s *ScriptComponent) Next() {
	s.currentAction = (s.currentAction + 1) % len(s.queuedActions)
	if s.currentAction == 0 {
		s.queuedActions = make([]ScriptedAction, 0)
	} else {
		println(fmt.Sprintf("ScriptComponent: Next action is %v", s.queuedActions[s.currentAction].ToString()))
	}
}

func (s *ScriptComponent) IsFinished() bool {
	return len(s.queuedActions) == 0
}

func (s *ScriptComponent) Clear() {
	s.queuedActions = make([]ScriptedAction, 0)
	s.currentAction = 0
}

func (s *ScriptComponent) SetPreferredLocation(location geometry.Point) {
	s.preferredLocation = location
	s.stayInPreferredLocation = true
}

func (s *ScriptComponent) StopStayingInPreferredLocation() {
	s.stayInPreferredLocation = false
}
