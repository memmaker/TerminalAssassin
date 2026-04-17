package objects

import (
	"fmt"
	"strings"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

// SearchableContainerState tracks whether the container is locked, available to search, or already searched.
type SearchableContainerState uint8

const (
	SearchableContainerStateLocked   SearchableContainerState = iota
	SearchableContainerStateUnlocked                          // has items, ready to search
	SearchableContainerStateSearched                          // items taken
)

// SearchableContainer is a map object (chest of drawers, drawers, desk) that
// holds items behind an optional lock. The player searches it over ~3 seconds
// and receives the contents directly into their inventory.
type SearchableContainer struct {
	State            SearchableContainerState
	LockType         core.LockType // only relevant when State == Locked
	HasLock          bool
	KeyString        string
	Difficulty       core.LockDifficulty
	ContentItemNames []string
	icon             rune
	name             string
	position         geometry.Point
}

// ---- constructors ----

func newSearchableContainer(displayName string, icon rune) *SearchableContainer {
	return &SearchableContainer{
		State: SearchableContainerStateUnlocked,
		icon:  icon,
		name:  displayName,
	}
}

func newLockedMechanicalSearchableContainer(displayName string, icon rune) *SearchableContainer {
	return &SearchableContainer{
		State:    SearchableContainerStateLocked,
		HasLock:  true,
		LockType: core.LockTypeMechanical,
		icon:     icon,
		name:     displayName,
	}
}

func newLockedElectronicSearchableContainer(displayName string, icon rune) *SearchableContainer {
	return &SearchableContainer{
		State:    SearchableContainerStateLocked,
		HasLock:  true,
		LockType: core.LockTypeElectronic,
		icon:     icon,
		name:     displayName,
	}
}

// ---- services.LockDifficultyHolder ----

func (sc *SearchableContainer) GetLockDifficulty() core.LockDifficulty  { return sc.Difficulty }
func (sc *SearchableContainer) SetLockDifficulty(d core.LockDifficulty) { sc.Difficulty = d }

// ---- services.KeyBound ----

func (sc *SearchableContainer) GetKey() string    { return sc.KeyString }
func (sc *SearchableContainer) SetKey(k string)   { sc.KeyString = k }

// ---- services.ContentHolder ----

func (sc *SearchableContainer) GetContents() []string  { return sc.ContentItemNames }
func (sc *SearchableContainer) SetContents(c []string) { sc.ContentItemNames = c }

// ---- services.Object ----

func (sc *SearchableContainer) Pos() geometry.Point         { return sc.position }
func (sc *SearchableContainer) SetPos(p geometry.Point)     { sc.position = p }
func (sc *SearchableContainer) IsWalkable(*core.Actor) bool { return false }
func (sc *SearchableContainer) IsTransparent() bool         { return false }
func (sc *SearchableContainer) IsPassableForProjectile() bool { return false }
func (sc *SearchableContainer) ApplyStimulus(_ services.Engine, _ stimuli.Stimulus) {}

func (sc *SearchableContainer) Icon() rune { return sc.icon }

func (sc *SearchableContainer) Style(st common.Style) common.Style {
	return common.Style{Foreground: core.CurrentTheme.ObjectForeground, Background: st.Background}
}

func (sc *SearchableContainer) Description() string {
	switch sc.State {
	case SearchableContainerStateLocked:
		if sc.LockType == core.LockTypeElectronic {
			return fmt.Sprintf("%s (electronically locked)", sc.name)
		}
		return fmt.Sprintf("%s (locked)", sc.name)
	case SearchableContainerStateSearched:
		return fmt.Sprintf("%s (empty)", sc.name)
	default:
		if len(sc.ContentItemNames) > 0 {
			return fmt.Sprintf("%s (contains items)", sc.name)
		}
		return sc.name
	}
}

func (sc *SearchableContainer) EncodeAsString() string {
	switch {
	case sc.HasLock && sc.LockType == core.LockTypeElectronic:
		return sc.name + " (electronic)"
	case sc.HasLock:
		return sc.name + " (locked)"
	default:
		return sc.name
	}
}

// IsActionAllowed — search allowed when unlocked and has items; lock-picking when locked.
func (sc *SearchableContainer) IsActionAllowed(_ services.Engine, _ *core.Actor) bool {
	switch sc.State {
	case SearchableContainerStateLocked:
		return true
	case SearchableContainerStateUnlocked:
		return len(sc.ContentItemNames) > 0
	}
	return false
}

// Action handles lock-picking (when locked) or searching (when unlocked with contents).
func (sc *SearchableContainer) Action(m services.Engine, person *core.Actor) {
	game := m.GetGame()

	unlock := func() {
		sc.State = SearchableContainerStateUnlocked
	}

	switch sc.State {

	// ── Locked: attempt to pick / use key ──────────────────────────────────
	case SearchableContainerStateLocked:
		switch {
		case sc.isUnlockableWithKeyFrom(person):
			sc.State = SearchableContainerStateUnlocked
			game.PrintMessage("Unlocked.")

		case sc.LockType == core.LockTypeMechanical:
			performMechanicalPickLock(m, person, sc.Pos(), sc.Difficulty, unlock, unlock)

		case sc.LockType == core.LockTypeElectronic:
			performElectronicPickLock(m, person, sc.Pos(), sc.Difficulty, unlock)
		}

	// ── Unlocked: search animation then transfer items ─────────────────────
	case SearchableContainerStateUnlocked:
		if len(sc.ContentItemNames) == 0 {
			game.PrintMessage("Nothing here.")
			return
		}
		sc.beginSearch(m, person)
	}
}

// beginSearch starts the 3-second search animation and transfers items on completion.
func (sc *SearchableContainer) beginSearch(m services.Engine, person *core.Actor) {
	animator := m.GetAnimator()
	aic := m.GetAI()

	done := false
	aic.SetEngaged(person, core.ActorStatusEngaged, func() bool { return done })

	animator.ActorEngagedAnimationWithCancel(person, core.GlyphEmptyHand, sc.Pos(), 3.0, func() {
		done = true
		sc.transferContentsToInventory(m, person)
	}, func() {
		done = true
	})
}

// transferContentsToInventory spawns each item at the container and immediately picks it up.
func (sc *SearchableContainer) transferContentsToInventory(m services.Engine, person *core.Actor) {
	game := m.GetGame()
	if len(sc.ContentItemNames) == 0 {
		game.PrintMessage("The container is empty.")
		return
	}
	factory := services.NewFactory(m)
	for _, name := range sc.ContentItemNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		item := factory.DecodeStringToItem(name)
		game.PlaceItem(sc.Pos(), &item)
		game.PickUpItemAt(person, sc.Pos())
	}
	sc.ContentItemNames = nil
	sc.State = SearchableContainerStateSearched
	game.PrintMessage("You search the " + sc.name + " and take the contents.")
}

func (sc *SearchableContainer) isUnlockableWithKeyFrom(person *core.Actor) bool {
	return isUnlockableWithKeyFrom(sc.LockType, sc.KeyString, person)
}

