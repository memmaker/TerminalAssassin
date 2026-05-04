package ai

import "github.com/memmaker/terminal-assassin/game/core"

// SleepingState is pushed onto the AI stack when an NPC is knocked out.
// It does nothing — the actor is in the downed list and won't be updated.
// Popping it (on WakeUp) restores the previous behaviour.
type SleepingState struct {
	AIContext
}

func (s *SleepingState) NextAction() core.AIUpdate { return core.AIUpdate{DelayInSeconds: 10} }
func (s *SleepingState) Status() core.ActorState   { return core.ActorStatusSleeping }
