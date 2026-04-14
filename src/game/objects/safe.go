package objects

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"strings"
)

const (
	SafeTypeMechanical = core.LockTypeMechanical
	SafeTypeElectronic = core.LockTypeElectronic
)

// SafeState tracks whether the safe is locked, unlocked-but-closed, or open.
type SafeState uint8

const (
	SafeStateLocked SafeState = iota
	SafeStateClosed           // unlocked, not yet opened
	SafeStateOpen
)

// Safe is a placeable map object that can hold items behind a lock.
// Its contents are stored as item names and spawned onto the map when first opened.
// In objects.txt a safe record looks like:
//
//	ObjectAt: 12,8
//	Name: mechanical safe (locked)
//	Key: vault_key_01
//	Content: diamond necklace
//	Content: gold watch
type Safe struct {
	State            SafeState
	Type             core.LockType
	Keypad           bool
	KeyString        string
	Difficulty       core.LockDifficulty
	ContentItemNames []string
	position         geometry.Point
}

// ---- services.LockDifficultyHolder ----

func (s *Safe) GetLockDifficulty() core.LockDifficulty  { return s.Difficulty }
func (s *Safe) SetLockDifficulty(d core.LockDifficulty) { s.Difficulty = d }

func newMechanicalSafe() *Safe {
	return &Safe{State: SafeStateLocked, Type: SafeTypeMechanical}
}

func newElectronicSafe() *Safe {
	return &Safe{State: SafeStateLocked, Type: SafeTypeElectronic}
}

func newKeypadSafe() *Safe {
	return &Safe{State: SafeStateLocked, Type: SafeTypeElectronic, Keypad: true}
}

// ---- services.KeyBound ----

func (s *Safe) GetKey() string    { return s.KeyString }
func (s *Safe) SetKey(key string) { s.KeyString = key }

// ---- services.ContentHolder ----

func (s *Safe) GetContents() []string  { return s.ContentItemNames }
func (s *Safe) SetContents(c []string) { s.ContentItemNames = c }

// ---- services.Object ----

func (s *Safe) Pos() geometry.Point           { return s.position }
func (s *Safe) SetPos(pos geometry.Point)     { s.position = pos }
func (s *Safe) IsWalkable(*core.Actor) bool   { return false }
func (s *Safe) IsTransparent() bool           { return false }
func (s *Safe) IsPassableForProjectile() bool { return false }

func (s *Safe) Icon() rune {
	switch {
	case s.State == SafeStateOpen && s.Type == SafeTypeMechanical:
		return core.GlyphMechanicalSafeOpen
	case s.State == SafeStateOpen && s.Type == SafeTypeElectronic:
		return core.GlyphElectronicSafeOpen
	case s.Type == SafeTypeElectronic:
		return core.GlyphElectronicSafe
	default:
		return core.GlyphMechanicalSafe
	}
}

func (s *Safe) Style(st common.Style) common.Style {
	return common.Style{Foreground: core.CurrentTheme.ObjectForeground, Background: st.Background}
}

func (s *Safe) Description() string {
	switch s.State {
	case SafeStateLocked:
		if s.Keypad {
			return "a keypad safe (locked)"
		}
		if s.KeyString != "" {
			return fmt.Sprintf("a locked safe (%s)", s.KeyString)
		}
		return "a locked safe"
	case SafeStateClosed:
		if s.Keypad {
			return "a keypad safe (unlocked)"
		}
		return "an unlocked safe"
	case SafeStateOpen:
		if s.Keypad {
			return "a keypad safe (open)"
		}
		return "an open safe"
	}
	return "a safe"
}

// EncodeAsString returns the plain factory name for this safe.
// Contents and key are written as separate fields by the serialiser.
func (s *Safe) EncodeAsString() string {
	if s.Keypad {
		return "keypad safe"
	}
	if s.Type == SafeTypeElectronic {
		return "electronic safe (locked)"
	}
	return "mechanical safe (locked)"
}

func (s *Safe) ApplyStimulus(_ services.Engine, _ stimuli.Stimulus) {}

// IsActionAllowed returns true when the player can usefully interact with the safe.
func (s *Safe) IsActionAllowed(_ services.Engine, person *core.Actor) bool {
	switch s.State {
	case SafeStateLocked:
		return true
	case SafeStateClosed:
		return true
	case SafeStateOpen:
		return false
	}
	return false
}

// Action handles all player interactions with the safe.
func (s *Safe) Action(m services.Engine, person *core.Actor) {
	game := m.GetGame()
	openSafe := func() { s.State = SafeStateOpen; s.spawnContents(m) }

	switch {

	// ── Keypad: prompt for code ───────────────────────────────────────────────
	case s.Keypad && s.State == SafeStateLocked:
		m.GetUI().ShowTextInput("Enter code: ", "", func(code string) {
			if code == s.KeyString {
				s.State = SafeStateOpen
				s.spawnContents(m)
			} else {
				game.PrintMessage("Wrong code.")
			}
		}, func() {})

	// ── Unlock with a matching physical key ──────────────────────────────────
	case s.State == SafeStateLocked && s.isUnlockableWithKeyFrom(person):
		s.State = SafeStateOpen
		s.spawnContents(m)

	// ── Pick the lock — mechanical ───────────────────────────────────────────
	case s.State == SafeStateLocked && s.Type == SafeTypeMechanical && !s.isUnlockableWithKeyFrom(person):
		performMechanicalPickLock(m, person, s.Pos(), s.Difficulty, openSafe, openSafe)

	// ── Pick the lock — electronic ───────────────────────────────────────────
	case s.State == SafeStateLocked && s.Type == SafeTypeElectronic && !s.isUnlockableWithKeyFrom(person):
		performElectronicPickLock(m, person, s.Pos(), s.Difficulty, openSafe)

	// ── Unlocked but not yet opened ───────────────────────────────────────────
	case s.State == SafeStateClosed:
		s.State = SafeStateOpen
		s.spawnContents(m)
	}
}

// spawnContents creates each item and places it on the map near the safe.
func (s *Safe) spawnContents(m services.Engine) {
	game := m.GetGame()
	if len(s.ContentItemNames) == 0 {
		game.PrintMessage("The safe is empty.")
		return
	}
	factory := services.NewFactory(m)
	for _, name := range s.ContentItemNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		item := factory.DecodeStringToItem(name)
		game.PlaceItem(s.Pos(), &item)
	}
	s.ContentItemNames = nil
}

// ---- private helpers ----

func (s *Safe) isUnlockableWithKeyFrom(person *core.Actor) bool {
	return isUnlockableWithKeyFrom(s.Type, s.KeyString, person)
}

func (s *Safe) isUnlockableWithPickFrom(person *core.Actor) bool {
	return isUnlockableWithPickFrom(s.Type, s.Difficulty, person)
}
