package objects

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

// isUnlockableWithKeyFrom returns true when person carries the key or keycard
// that matches keyString for the given lockType.
func isUnlockableWithKeyFrom(lockType core.LockType, keyString string, person *core.Actor) bool {
	if person == nil || keyString == "" {
		return false
	}
	switch lockType {
	case core.LockTypeMechanical:
		return person.HasKeyInInventory(keyString)
	case core.LockTypeElectronic:
		return person.HasKeyCardInInventory(keyString)
	}
	return false
}

// isUnlockableWithPickFrom returns true when person carries enough lockpicks
// of the correct type for the given lockType and difficulty.
func isUnlockableWithPickFrom(lockType core.LockType, difficulty core.LockDifficulty, person *core.Actor) bool {
	if person == nil {
		return false
	}
	needed := difficulty.PickCount()
	switch lockType {
	case core.LockTypeMechanical:
		return person.CountItemTypeInInventory(core.ItemTypeMechanicalLockpick) >= needed
	case core.LockTypeElectronic:
		return person.CountItemTypeInInventory(core.ItemTypeElectronicLockpick) >= needed
	}
	return false
}

// performMechanicalPickLock handles the lockpick/crowbar interaction flow for a
// mechanical lock at lockPos.
//   - onPick    is called when the lock is opened with lockpicks.
//   - onCrowbar is called when the lock is pried open with a crowbar.
//     Pass the same function as onPick when both outcomes are equivalent.
func performMechanicalPickLock(
	m services.Engine,
	person *core.Actor,
	lockPos geometry.Point,
	difficulty core.LockDifficulty,
	onPick func(),
	onCrowbar func(),
) {
	game := m.GetGame()
	aic := m.GetAI()
	animator := m.GetAnimator()
	needed := difficulty.PickCount()

	hasPicks := person.CountItemTypeInInventory(core.ItemTypeMechanicalLockpick) >= needed
	hasCrowbar := difficulty != core.LockDifficultyHard && person.CountItemTypeInInventory(core.ItemTypeCrowbar) >= 1

	switch {
	case hasPicks:
		pickTime := difficulty.PickTime()
		done := false
		aic.SetEngaged(person, core.ActorStatusEngagedIllegal, func() bool { return done })
		game.IllegalActionAt(lockPos, core.ObservationIllegalAction)
		animator.ActorEngagedAnimationWithCancel(person, core.GlyphLockpick, lockPos, pickTime, func() {
			done = true
			person.ConsumeItemsFromInventory(core.ItemTypeMechanicalLockpick, needed)
			onPick()
		}, func() {
			done = true
		})

	case hasCrowbar:
		crowbarTime := difficulty.PickTime() * 3
		game.SoundEventAt(lockPos, core.ObservationMeleeNoises, 20)
		game.IllegalActionAt(lockPos, core.ObservationIllegalAction)
		done := false
		aic.SetEngaged(person, core.ActorStatusEngagedIllegal, func() bool { return done })
		animator.ActorEngagedAnimationWithCancel(person, core.GlyphCrowbar, lockPos, crowbarTime, func() {
			done = true
			person.ConsumeItemsFromInventory(core.ItemTypeCrowbar, 1)
			onCrowbar()
		}, func() {
			done = true
		})

	default:
		if difficulty == core.LockDifficultyHard {
			game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s).", difficulty.ToString(), needed))
		} else {
			game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s) or a crowbar.", difficulty.ToString(), needed))
		}
	}
}

// performElectronicPickLock handles the e-lockpick interaction flow for an
// electronic lock at lockPos. onSuccess is called when the lock is opened.
func performElectronicPickLock(
	m services.Engine,
	person *core.Actor,
	lockPos geometry.Point,
	difficulty core.LockDifficulty,
	onSuccess func(),
) {
	game := m.GetGame()
	aic := m.GetAI()
	animator := m.GetAnimator()
	needed := difficulty.PickCount()

	if person.CountItemTypeInInventory(core.ItemTypeElectronicLockpick) < needed {
		game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s).", difficulty.ToString(), needed))
		return
	}

	pickTime := difficulty.PickTime()
	done := false
	aic.SetEngaged(person, core.ActorStatusEngagedIllegal, func() bool { return done })
	game.IllegalActionAt(lockPos, core.ObservationIllegalAction)
	animator.ActorEngagedAnimationWithCancel(person, core.GlyphLockpickElectronic, lockPos, pickTime, func() {
		done = true
		person.ConsumeItemsFromInventory(core.ItemTypeElectronicLockpick, needed)
		onSuccess()
	}, func() {
		done = true
	})
}

