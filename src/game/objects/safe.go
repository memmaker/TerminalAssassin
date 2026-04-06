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

// safePickTime is how long (in seconds) it takes to pick the safe's lock.
// Safes are harder to crack than doors.
const safePickTime = 8.0

// SafeType distinguishes mechanical (key + lockpick) from electronic (keycard + e-pick).
type SafeType bool

const (
    SafeTypeMechanical SafeType = false
    SafeTypeElectronic SafeType = true
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
//	FgColor: ...
//	BgColor: ...
type Safe struct {
    State            SafeState
    Type             SafeType
    Keypad           bool
    KeyString        string
    Difficulty       core.LockDifficulty
    ContentItemNames []string
    position         geometry.Point
    definedStyle     common.Style
}

// ---- services.LockDifficultyHolder ----

func (s *Safe) GetLockDifficulty() core.LockDifficulty     { return s.Difficulty }
func (s *Safe) SetLockDifficulty(d core.LockDifficulty)    { s.Difficulty = d }

func newMechanicalSafe() *Safe {
    return &Safe{State: SafeStateLocked, Type: SafeTypeMechanical, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

func newElectronicSafe() *Safe {
    return &Safe{State: SafeStateLocked, Type: SafeTypeElectronic, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

func newKeypadSafe() *Safe {
    return &Safe{State: SafeStateLocked, Type: SafeTypeElectronic, Keypad: true, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

// ---- services.KeyBound ----

func (s *Safe) GetKey() string    { return s.KeyString }
func (s *Safe) SetKey(key string) { s.KeyString = key }

// ---- services.ContentHolder ----

func (s *Safe) GetContents() []string  { return s.ContentItemNames }
func (s *Safe) SetContents(c []string) { s.ContentItemNames = c }

// ---- services.Object ----

func (s *Safe) GetStyle() common.Style        { return s.definedStyle }
func (s *Safe) SetStyle(style common.Style)   { s.definedStyle = style }
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
    return s.definedStyle.WithBg(st.Background)
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
        if s.Keypad {
            return true
        }
        // Always allow interaction: the player will receive key usage or a
        // "needs X picks" message inside Action when picks are insufficient.
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
    aic := m.GetAI()
    animator := m.GetAnimator()

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
        needed := s.Difficulty.PickCount()
        hasPicks := person.CountItemTypeInInventory(core.ItemTypeMechanicalLockpick) >= needed
        hasCrowbar := s.Difficulty != core.LockDifficultyHard && person.CountItemTypeInInventory(core.ItemTypeCrowbar) >= 1
        if hasPicks {
            pickTime := s.Difficulty.PickTime()
            done := false
            aic.SetEngaged(person, core.ActorStatusEngagedIllegal, func() bool { return done })
            game.IllegalActionAt(person.Pos(), core.ObservationIllegalAction)
            animator.ActorEngagedAnimation(person, core.GlyphLockpick, s.Pos(), pickTime, func() {
                done = true
                person.ConsumeItemsFromInventory(core.ItemTypeMechanicalLockpick, needed)
                s.State = SafeStateOpen
                s.spawnContents(m)
            })
        } else if hasCrowbar {
            crowbarTime := s.Difficulty.PickTime() * 3
            game.SoundEventAt(s.Pos(), core.ObservationMeleeNoises, 20)
            game.IllegalActionAt(person.Pos(), core.ObservationIllegalAction)
            done := false
            aic.SetEngaged(person, core.ActorStatusEngagedIllegal, func() bool { return done })
            animator.ActorEngagedAnimation(person, core.GlyphCrowbar, s.Pos(), crowbarTime, func() {
                done = true
                person.ConsumeItemsFromInventory(core.ItemTypeCrowbar, 1)
                s.State = SafeStateOpen
                s.spawnContents(m)
            })
        } else {
            if s.Difficulty == core.LockDifficultyHard {
                game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s).", s.Difficulty.ToString(), needed))
            } else {
                game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s) or a crowbar.", s.Difficulty.ToString(), needed))
            }
        }

    // ── Pick the lock — electronic ───────────────────────────────────────────
    case s.State == SafeStateLocked && s.Type == SafeTypeElectronic && !s.isUnlockableWithKeyFrom(person):
        needed := s.Difficulty.PickCount()
        if person.CountItemTypeInInventory(core.ItemTypeElectronicalLockpick) < needed {
            game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s).", s.Difficulty.ToString(), needed))
            return
        }
        pickTime := s.Difficulty.PickTime()
        done := false
        aic.SetEngaged(person, core.ActorStatusEngagedIllegal, func() bool { return done })
        game.IllegalActionAt(person.Pos(), core.ObservationIllegalAction)
        animator.ActorEngagedAnimation(person, core.GlyphLockpickElectronic, s.Pos(), pickTime, func() {
            done = true
            person.ConsumeItemsFromInventory(core.ItemTypeElectronicalLockpick, needed)
            s.State = SafeStateOpen
            s.spawnContents(m)
        })

    // ── Already open ─────────────────────────────────────────────────────────
    case s.State == SafeStateOpen:
        game.PrintMessage("The safe is already open.")

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
    if person == nil || s.KeyString == "" {
        return false
    }
    switch s.Type {
    case SafeTypeMechanical:
        return person.HasKeyInInventory(s.KeyString)
    case SafeTypeElectronic:
        return person.HasKeyCardInInventory(s.KeyString)
    }
    return false
}

func (s *Safe) isUnlockableWithPickFrom(person *core.Actor) bool {
    if person == nil {
        return false
    }
    needed := s.Difficulty.PickCount()
    switch s.Type {
    case SafeTypeMechanical:
        return person.CountItemTypeInInventory(core.ItemTypeMechanicalLockpick) >= needed
    case SafeTypeElectronic:
        return person.CountItemTypeInInventory(core.ItemTypeElectronicalLockpick) >= needed
    }
    return false
}
