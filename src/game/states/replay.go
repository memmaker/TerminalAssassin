package states

import (
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/rng"
)

// GameStateReplay loads a replay file, verifies the map hash, seeds the RNG,
// and runs GameStateGameplay while feeding recorded commands instead of live input.
type GameStateReplay struct {
	ReplayPath  string
	OnComplete  func() // called when the replay finishes; nil = just pop state
	engine      services.Engine
	gameplay    *GameStateGameplay
	replayInput *replayInputSource
	isDirty     bool
}

func (r *GameStateReplay) ClearOverlay() {
	if r.gameplay != nil {
		r.gameplay.ClearOverlay()
	}
}
func (r *GameStateReplay) SetDirty() {
	r.isDirty = true
	if r.gameplay != nil {
		r.gameplay.SetDirty()
	}
}

func (r *GameStateReplay) Init(engine services.Engine) {
	r.engine = engine

	rf, err := services.LoadReplayFile(r.ReplayPath)
	if err != nil {
		engine.GetUI().ShowAlert([]string{"Failed to load replay:", err.Error()})
		engine.GetGame().PopState()
		return
	}

	// Load the map.
	loadedMap, err := engine.LoadMap(rf.MapPath)
	if err != nil {
		engine.GetUI().ShowAlert([]string{"Failed to load map:", err.Error()})
		engine.GetGame().PopState()
		return
	}

	// Verify hash.
	if loadedMap.MapHash() != rf.MapHash {
		engine.GetUI().ShowAlert([]string{
			"Map has changed since recording.",
			"Replay may not match original.",
		})
	}

	engine.GetGame().InitLoadedMap(loadedMap)

	// Seed the RNG deterministically.
	rng.Seed(rf.Seed)

	// Build the replay input wrapper and hand it to gameplay.
	r.replayInput = &replayInputSource{
		entries:       rf.Entries,
		currentTick:   engine.CurrentRawTick,
		durationTicks: rf.DurationTicks,
	}

	engine.SetInputOverride(r.replayInput)
	r.gameplay = &GameStateGameplay{}
	r.gameplay.Init(engine)
	r.isDirty = true
}

func (r *GameStateReplay) Update(input services.InputInterface) {
	if r.gameplay == nil {
		return
	}
	r.gameplay.Update(input)

	if r.replayInput != nil && r.replayInput.replayDone() {
		r.engine.SetInputOverride(nil)
		r.engine.GetGame().PopState()
		if r.OnComplete != nil {
			r.OnComplete()
		}
	}
}

func (r *GameStateReplay) Draw(con console.CellInterface) {
	if r.gameplay == nil {
		return
	}
	r.gameplay.Draw(con)
}

// replayInputSource implements services.InputInterface by replaying stored entries.
type replayInputSource struct {
	entries       []services.ReplayEntry
	currentTick   func() uint64
	durationTicks uint64 // 0 = no end marker (truncated recording)
	cursor        int
}

func (r *replayInputSource) PollGameCommands() []core.InputCommand {
	tick := r.currentTick()
	cmds := make([]core.InputCommand, 0)
	for r.cursor < len(r.entries) && r.entries[r.cursor].Tick <= tick {
		cmds = append(cmds, r.entries[r.cursor].Command)
		r.cursor++
	}
	return cmds
}

// All other InputInterface methods are no-ops / zero-value returns for replay.
func (r *replayInputSource) SetMovementDelayForSneaking()            {}
func (r *replayInputSource) SetMovementDelayForWalkingAndRunning()   {}
func (r *replayInputSource) PollUICommands() []core.InputCommand     { return r.PollGameCommands() }
func (r *replayInputSource) PollEditorCommands() []core.InputCommand { return nil }
func (r *replayInputSource) ConfirmOrCancel() bool                   { return false }
func (r *replayInputSource) DevTerminalKeyPressed() bool             { return false }
func (r *replayInputSource) PollText() []core.InputCommand           { return nil }
func (r *replayInputSource) IsShiftPressed() bool                    { return false }
func (r *replayInputSource) GetKeyDefinitions() services.KeyDefinitions {
	return services.KeyDefinitions{}
}

// replayDone returns true once the recorded tick duration has elapsed.
// Falls back to cursor exhaustion for truncated recordings.
func (r *replayInputSource) replayDone() bool {
	if r.durationTicks > 0 {
		return r.currentTick() >= r.durationTicks
	}
	return r.cursor >= len(r.entries)
}
