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
    return &Door{uniqueName: name, State: DoorStateClosed, DamageThreshold: damageThreshold, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

func NewLockedDoorAt(name string, key string, damageThreshold int) *Door {
    return &Door{uniqueName: name, KeyString: key, State: DoorStateLocked, DamageThreshold: damageThreshold, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

func NewLockedElectronicDoorAt(name string, key string, damageThreshold int) *Door {
    return &Door{uniqueName: name, KeyString: key, State: DoorStateLocked, Type: DoorTypeElectronic, DamageThreshold: damageThreshold, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

func NewKeypadDoorAt(name string, code string, damageThreshold int) *Door {
    return &Door{uniqueName: name, KeyString: code, State: DoorStateLocked, Type: DoorTypeElectronic, Keypad: true, DamageThreshold: damageThreshold, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

func NewOpenDoorAt(name string, damageThreshold int) *Door {
    return &Door{uniqueName: name, State: DoorStateOpen, DamageThreshold: damageThreshold, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
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

type DoorType bool

const (
    DoorTypeMechanic   DoorType = false
    DoorTypeElectronic DoorType = true
)

type Door struct {
    State           DoorState
    KeyString       string
    position        geometry.Point
    IsBurnable      bool
    Type            DoorType
    Keypad          bool
    DamageThreshold int
    Difficulty      core.LockDifficulty
    uniqueName      string
    definedStyle    common.Style
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

func (d *Door) GetStyle() common.Style {
    return d.definedStyle
}

func (d *Door) SetStyle(style common.Style) {
    d.definedStyle = style
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
    return
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
    animator := m.GetAnimator()
    game := m.GetGame()
    aic := m.GetAI()
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
        needed := d.Difficulty.PickCount()
        hasPicks := person.CountItemTypeInInventory(core.ItemTypeMechanicalLockpick) >= needed
        hasCrowbar := d.Difficulty != core.LockDifficultyHard && person.CountItemTypeInInventory(core.ItemTypeCrowbar) >= 1
        if hasPicks {
            pickTime := d.Difficulty.PickTime()
            animationCompleted := false
            until := func() bool { return animationCompleted }
            onLockPicked := func() {
                animationCompleted = true
                person.ConsumeItemsFromInventory(core.ItemTypeMechanicalLockpick, needed)
                d.State = DoorStateClosed
            }
            aic.SetEngaged(person, core.ActorStatusEngagedIllegal, until)
            game.IllegalActionAt(d.Pos(), core.ObservationIllegalAction)
            animator.ActorEngagedAnimation(person, core.GlyphLockpick, d.Pos(), pickTime, onLockPicked)
        } else if hasCrowbar {
            crowbarTime := d.Difficulty.PickTime() * 3
            game.SoundEventAt(d.Pos(), core.ObservationMeleeNoises, 20)
            game.IllegalActionAt(d.Pos(), core.ObservationIllegalAction)
            animationCompleted := false
            until := func() bool { return animationCompleted }
            onPried := func() {
                animationCompleted = true
                person.ConsumeItemsFromInventory(core.ItemTypeCrowbar, 1)
                d.State = DoorStateClosed
                game.UpdateAllFoVsFrom(d.Pos())
            }
            aic.SetEngaged(person, core.ActorStatusEngagedIllegal, until)
            animator.ActorEngagedAnimation(person, core.GlyphCrowbar, d.Pos(), crowbarTime, onPried)
        } else {
            if d.Difficulty == core.LockDifficultyHard {
                game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s).", d.Difficulty.ToString(), needed))
            } else {
                game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s) or a crowbar.", d.Difficulty.ToString(), needed))
            }
        }
    case d.State == DoorStateLocked && d.Type == DoorTypeElectronic:
        needed := d.Difficulty.PickCount()
        if person.CountItemTypeInInventory(core.ItemTypeElectronicalLockpick) < needed {
            game.PrintMessage(fmt.Sprintf("This %s lock needs %d pick(s).", d.Difficulty.ToString(), needed))
            return
        }
        pickTime := d.Difficulty.PickTime()
        animationCompleted := false
        until := func() bool { return animationCompleted }
        onLockPicked := func() {
            animationCompleted = true
            person.ConsumeItemsFromInventory(core.ItemTypeElectronicalLockpick, needed)
            d.State = DoorStateClosed
        }
        aic.SetEngaged(person, core.ActorStatusEngagedIllegal, until)
        animator.ActorEngagedAnimation(person, core.GlyphLockpickElectronic, d.Pos(), pickTime, onLockPicked)
    case d.State == DoorStateOpen:
        d.State = DoorStateClosed
        game.UpdateAllFoVsFrom(d.Pos())
    case d.State == DoorStateClosed:
        d.State = DoorStateOpen
        game.UpdateAllFoVsFrom(d.Pos())
    }
    return
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
    return d.definedStyle
}

func (d *Door) IsUnlockableWithKeyFrom(person *core.Actor) bool {
    switch {
    case person == nil:
        return false
    case d.Type == DoorTypeMechanic && person.HasKeyInInventory(d.KeyString):
        return true
    case d.Type == DoorTypeElectronic && person.HasKeyCardInInventory(d.KeyString):
        return true
    }
    return false
}

func (d *Door) IsUnlockableWithPickFrom(person *core.Actor) bool {
    needed := d.Difficulty.PickCount()
    switch {
    case person == nil:
        return false
    case d.Type == DoorTypeMechanic:
        return person.CountItemTypeInInventory(core.ItemTypeMechanicalLockpick) >= needed
    case d.Type == DoorTypeElectronic:
        return person.CountItemTypeInInventory(core.ItemTypeElectronicalLockpick) >= needed
    }
    return false
}
