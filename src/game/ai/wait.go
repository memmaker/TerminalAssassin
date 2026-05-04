package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
)

type Wait struct {
	AIContext
	Until func() bool
}

func (w *Wait) Status() core.ActorState {
	if below := w.Person.AI.PeekBelow(); below != nil {
		return below.Status()
	}
	return core.ActorStatusIdle
}

func (w *Wait) NextAction() core.AIUpdate {
	aic := w.Engine.GetAI()
	aic.UpdateVision(w.Person)
	if w.Until != nil && w.Until() {
		w.Person.AI.PopState()
		return NextUpdateIn(1)
	}
	return NextUpdateIn(1)
}
