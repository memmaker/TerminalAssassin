package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
)

type WaitMovement struct {
	AIContext
	Until func() bool
}

func (w *WaitMovement) NextAction() core.AIUpdate {
	aic := w.Engine.GetAI()
	aic.UpdateVision(w.Person)
	if w.Until != nil && w.Until() {
		w.Person.Status = core.ActorStatusIdle
		w.Person.AI.PopState()
		return NextUpdateIn(1)
	}
	return NextUpdateIn(1)
}
