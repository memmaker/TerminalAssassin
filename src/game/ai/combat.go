package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/rng"
)

type CombatMovement struct {
	AIContext
	Target               *core.Actor
	isAimingAt           geometry.Point
	LastKnownPosition    *geometry.Point
	stepCounter          int
	noShotCounter        int  // ticks where target is visible but ranged shot is blocked
	patternIndex         int  // position in the melee attack pattern
	barehandedOnCooldown bool // unarmed attack cooldown
}

func (t *CombatMovement) Status() core.ActorState { return core.ActorStatusCombat }

func (t *CombatMovement) OnDestinationReached() core.AIUpdate {
	return NextUpdateIn(0.4)
}

func (t *CombatMovement) OnCannotReachDestination() core.AIUpdate {
	return NextUpdateIn(0.4)
}

func (t *CombatMovement) NextAction() core.AIUpdate {
	person := t.Person
	if t.Target.IsDowned() || t.stepCounter > 50 {
		t.exitCombat(person)
		return NextUpdateIn(0.4)
	}

	// Step 1: equip best available weapon (ranged preferred, melee fallback).
	person.EquipBestWeapon()

	currentMap := t.Engine.GetGame().GetMap()
	hasRanged := person.EquippedItem != nil && person.EquippedItem.IsRangedWeapon()
	hasMelee := true

	// Step 2 (ranged): shoot if we have a clear line of fire.
	if hasRanged && person.CanSeeActor(t.Target) && currentMap.IsPathPassableForProjectile(person.Pos(), t.Target.Pos()) {
		targetPos := t.Target.Pos()
		t.LastKnownPosition = &targetPos
		t.noShotCounter = 0
		return t.handleFiring(person)
	}

	// Update tracking counters.
	if person.CanSeeActor(t.Target) {
		targetPos := t.Target.Pos()
		t.LastKnownPosition = &targetPos
		t.stepCounter = 0
		if hasRanged {
			t.noShotCounter++
		}
	} else {
		t.stepCounter++
		t.noShotCounter = 0
	}

	// Exit ranged combat when line of fire stays blocked too long.
	if hasRanged && t.noShotCounter > 30 {
		t.exitCombat(person)
		return NextUpdateIn(0.4)
	}

	if t.LastKnownPosition == nil {
		return NextUpdateIn(0.4)
	}

	// Step 2 (melee/unarmed): close distance.
	// Step 3: attack if adjacent and weapon is ready.
	if hasMelee {
		dist := geometry.DistanceChebyshev(person.Pos(), t.Target.Pos())
		if dist <= 1 && person.CanSeeActor(t.Target) {
			return t.handleMeleeAttack(person)
		}
		return person.AI.Movement.Action(*t.LastKnownPosition, t)
	}

	// Unarmed or no weapon: close distance, no attack.
	return person.AI.Movement.Action(currentMap.GetRandomFreeNeighbor(*t.LastKnownPosition), t)
}

// exitCombat cleans up the combat state and pushes Investigation for the LKP.
func (t *CombatMovement) exitCombat(person *core.Actor) {
	ai := person.AI
	aic := t.Engine.GetAI()
	ai.PopState()

	if t.LastKnownPosition != nil && !person.IsInvestigating() {
		lkp := *t.LastKnownPosition
		report := core.IncidentReport{
			Type:     core.ObservationCombatSeen,
			Location: lkp,
			Time:     t.Engine.CurrentGameTime(),
		}
		aic.SwitchToInvestigation(person, report)
	}
}

func (t *CombatMovement) handleFiring(person *core.Actor) core.AIUpdate {
	t.LastKnownPosition = &geometry.Point{X: t.Target.Pos().X, Y: t.Target.Pos().Y}
	t.stepCounter = 0
	currentMap := t.Engine.GetGame().GetMap()
	person.LookDirection = geometry.DirectionVectorToAngleInDegrees(t.Target.Pos().Sub(person.Pos()))
	aimedShot := rng.R.Intn(100) < 50
	if person.EquippedItem.OnCooldown || (t.isAimingAt != t.Target.Pos() && aimedShot) {
		t.isAimingAt = t.Target.Pos()
	} else {
		game := t.Engine.GetGame()
		actions := game.GetActions()
		actions.UseEquippedItemAtRange(person, t.Target.Pos())
		if person.HasBurstWeaponEquipped() {
			t.Engine.Schedule(0.066, func() {
				actions.UseEquippedItemAtRange(person, currentMap.RandomPosAround(t.Target.Pos()))
			})
			t.Engine.Schedule(0.136, func() {
				actions.UseEquippedItemAtRange(person, currentMap.RandomPosAround(t.Target.Pos()))
			})
		}
	}
	return NextUpdateIn(rng.R.Float64()*0.5 + 0.5)
}

// handleMeleeAttack signals the incoming attack, then executes it after the
// reaction window. The pattern cycles through MeleePatternStandard.
func (t *CombatMovement) handleMeleeAttack(person *core.Actor) core.AIUpdate {
	if person.EquippedItem != nil && person.EquippedItem.OnCooldown {
		return NextUpdateIn(person.EquippedItem.Type.CooldownSecs())
	}
	if person.EquippedItem == nil && t.barehandedOnCooldown {
		return NextUpdateIn(0.5)
	}
	person.LookDirection = geometry.DirectionVectorToAngleInDegrees(t.Target.Pos().Sub(person.Pos()))

	pattern := core.MeleePatternStandard
	attackType := pattern[t.patternIndex%len(pattern)]
	t.patternIndex++

	done := false
	target := t.Target
	unarmed := person.EquippedItem == nil
	t.Engine.GetAnimator().MeleeSignalAnimation(person, attackType, core.MeleeAttackWindowSecs, func() {
		if !target.IsDowned() {
			if unarmed {
				t.Engine.GetGame().ApplyStimulusToActor(target, core.NewEffectSourceFromActor(person),
					stimuli.Stim{StimType: stimuli.StimulusBluntDamage, StimForce: 20})
				t.Engine.GetGame().IllegalActionAt(person.Pos(), core.ObservationCombatSeen)
			} else {
				t.Engine.GetGame().GetActions().UseEquippedItemForMelee(person, target.Pos())
			}
		}
		if unarmed {
			t.barehandedOnCooldown = true
			t.Engine.Schedule(0.5, func() { t.barehandedOnCooldown = false })
		}
		done = true
	}, func() {
		done = true
	})
	return DeferredUpdate(func() bool { return done })
}
