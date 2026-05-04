package ai

import "github.com/memmaker/terminal-assassin/game/core"

// Idle is a no-op sentinel returned by GetState when the AI stack is empty.
type Idle struct{}

func (Idle) NextAction() core.AIUpdate { return core.AIUpdate{DelayInSeconds: 1} }
func (Idle) Status() core.ActorState   { return core.ActorStatusIdle }
