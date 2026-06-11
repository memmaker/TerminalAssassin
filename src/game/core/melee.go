package core

import "github.com/memmaker/terminal-assassin/common"

// MeleeAttackWindowSecs is the time window the player has to react to a
// signalled melee attack (block / parry / dodge). Change to taste.
var MeleeAttackWindowSecs = 0.6

// MeleeAttackType classifies a single hit in an attack pattern.
type MeleeAttackType uint8

const (
	MeleeAttackDefault     MeleeAttackType = iota // white – can be blocked and parried
	MeleeAttackUnblockable                        // blue  – cannot be blocked
	MeleeAttackUnparriable                        // red   – cannot be parried
)

// Color returns the signal glyph colour for this attack type.
func (t MeleeAttackType) Color() common.Color {
	switch t {
	case MeleeAttackUnblockable:
		return common.Blue
	case MeleeAttackUnparriable:
		return common.Red
	default:
		return common.White
	}
}

// MeleePattern is a repeating sequence of attack types used by an NPC in melee.
type MeleePattern []MeleeAttackType

// Global attack patterns – any NPC can use them.
var (
	// MeleePatternStandard: default, default, unblockable
	MeleePatternStandard = MeleePattern{MeleeAttackDefault, MeleeAttackDefault, MeleeAttackUnblockable}
	// MeleePatternAggressive: unblockable, unblockable, unparriable
	MeleePatternAggressive = MeleePattern{MeleeAttackUnblockable, MeleeAttackUnblockable, MeleeAttackUnparriable}
)

