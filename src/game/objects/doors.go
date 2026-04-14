package objects

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

func NewClosedDoorAt(name string, damageThreshold int) *Door {
	return &Door{uniqueName: name, State: DoorStateClosed, DamageThreshold: damageThreshold}
}

func NewLockedDoorAt(name string, key string, damageThreshold int) *Door {
	return &Door{uniqueName: name, KeyString: key, State: DoorStateLocked, DamageThreshold: damageThreshold}
}

func NewLockedElectronicDoorAt(name string, key string, damageThreshold int) *Door {
	return &Door{uniqueName: name, KeyString: key, State: DoorStateLocked, Type: DoorTypeElectronic, DamageThreshold: damageThreshold}
}

func NewKeypadDoorAt(name string, code string, damageThreshold int) *Door {
	return &Door{uniqueName: name, KeyString: code, State: DoorStateLocked, Type: DoorTypeElectronic, Keypad: true, DamageThreshold: damageThreshold}
}

func NewOpenDoorAt(name string, damageThreshold int) *Door {
	return &Door{uniqueName: name, State: DoorStateOpen, DamageThreshold: damageThreshold}
}

type DoorState uint8

const (
	DoorStateClosed DoorState = iota
	DoorStateOpen
	DoorStateLocked
)

func (d DoorState) String() string {
	switch d {
	case DoorStateClosed:
		return "closed"
	case DoorStateOpen:
		return "open"
	case DoorStateLocked:
		return "locked"
	}
	return "unknown"
}

func StateFromString(s string) DoorState {
	switch s {
	case "closed":
		return DoorStateClosed
	case "open":
		return DoorStateOpen
	case "locked":
		return DoorStateLocked
	}
	return DoorStateClosed
}

const (
	DoorTypeMechanic   = core.LockTypeMechanical
	DoorTypeElectronic = core.LockTypeElectronic
)

type Door struct {
	State           DoorState
	KeyString       string
	position        geometry.Point
	IsBurnable      bool
	Type            core.LockType
	Keypad          bool
	DamageThreshold int
	Difficulty      core.LockDifficulty
	uniqueName      string
}

// ---- services.LockDifficultyHolder ----

func (d *Door) GetLockDifficulty() core.LockDifficulty     { return d.Difficulty }
func (d *Door) SetLockDifficulty(diff core.LockDifficulty) { d.Difficulty = diff }

func (d *Door) GetKey() string {
	return d.KeyString
}

func (d *Door) SetKey(key string) {
	d.KeyString = key
}

func (d *Door) EncodeAsString() string {
	return d.uniqueName
}

func (d *Door) Description() string {
	switch d.State {
	case DoorStateClosed:
		return "a closed door"
	case DoorStateOpen:
		return "an open door"
	case DoorStateLocked:
		if d.Keypad {
			return "a keypad door (locked)"
		}
		if d.KeyString != "" {
			return fmt.Sprintf("a locked door (%s)", d.KeyString)
		}
		return "a locked door"
	}
	return "a door"
}

func (d *Door) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {
	isRelevantDamageType := stim.Type() == stimuli.StimulusPiercingDamage || stim.Type() == stimuli.StimulusBluntDamage || stim.Type() == stimuli.StimulusExplosionDamage || stim.Type() == stimuli.StimulusFire

	isClosed := d.State == DoorStateClosed || d.State == DoorStateLocked
	if isRelevantDamageType && isClosed && stim.Force() > d.DamageThreshold {
		d.State = DoorStateOpen
	}
}

func (d *Door) Pos() geometry.Point {
	return d.position
}

func (d *Door) SetPos(pos geometry.Point) {
	d.position = pos
}

func (d *Door) IsWalkable(person *core.Actor) bool {
	if d.State == DoorStateLocked && d.IsUnlockableWithKeyFrom(person) {
		return true
	}
	return d.State != DoorStateLocked
}

func (d *Door) IsTransparent() bool {
	return d.State == DoorStateOpen
}
func (d *Door) IsPassableForProjectile() bool {
	return true
}
func (d *Door) ActionDescription() string {

	return "n/a"
}

func (d *Door) Action(m services.Engine, person *core.Actor) {
	game := m.GetGame()
	unlockDoor := func() { d.State = DoorStateClosed }
	breakDoor := func() { d.State = DoorStateOpen; game.UpdateAllFoVsFrom(d.Pos()) }
	switch {
	case d.State == DoorStateLocked && d.Keypad:
		m.GetUI().ShowTextInput("Enter code: ", "", func(code string) {
			if code == d.KeyString {
				d.State = DoorStateClosed
			} else {
				game.PrintMessage("Wrong code.")
			}
		}, func() {})
	case d.State == DoorStateLocked && person.HasKeyInInventory(d.KeyString) && d.Type == DoorTypeMechanic:
		d.State = DoorStateClosed
	case d.State == DoorStateLocked && person.HasKeyCardInInventory(d.KeyString) && d.Type == DoorTypeElectronic:
		d.State = DoorStateClosed
	case d.State == DoorStateLocked && d.Type == DoorTypeMechanic:
		performMechanicalPickLock(m, person, d.Pos(), d.Difficulty, unlockDoor, breakDoor)
	case d.State == DoorStateLocked && d.Type == DoorTypeElectronic:
		performElectronicPickLock(m, person, d.Pos(), d.Difficulty, unlockDoor)
	case d.State == DoorStateOpen:
		d.State = DoorStateClosed
		game.UpdateAllFoVsFrom(d.Pos())
	case d.State == DoorStateClosed:
		d.State = DoorStateOpen
		game.UpdateAllFoVsFrom(d.Pos())
	}
}

func (d *Door) IsActionAllowed(m services.Engine, person *core.Actor) bool {
	return true
}

func (d *Door) Icon() rune {
	switch {
	case d.State == DoorStateOpen:
		return core.GlyphOpenDoor
	case d.State == DoorStateClosed:
		return core.GlyphClosedDoor
	case d.State == DoorStateLocked && d.Type == DoorTypeMechanic:
		return core.GlyphLockedDoor
	case d.State == DoorStateLocked && d.Type == DoorTypeElectronic:
		return core.GlyphLockedDoorElectronic
	}
	return core.GlyphClosedDoor
}

func (d *Door) Style(st common.Style) common.Style {
	return common.Style{Foreground: core.CurrentTheme.ObjectForeground, Background: st.Background}
}

func (d *Door) IsUnlockableWithKeyFrom(person *core.Actor) bool {
	return isUnlockableWithKeyFrom(d.Type, d.KeyString, person)
}

func (d *Door) IsUnlockableWithPickFrom(person *core.Actor) bool {
	return isUnlockableWithPickFrom(d.Type, d.Difficulty, person)
}
