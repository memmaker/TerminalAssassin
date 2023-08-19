package objects

import (
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
	DamageThreshold int
	uniqueName      string
	definedStyle    common.Style
}

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
	InteractionTime := 4.0
	animator := m.GetAnimator()
	game := m.GetGame()
	aic := m.GetAI()
	switch {
	case d.State == DoorStateLocked && person.HasKeyInInventory(d.KeyString) && d.Type == DoorTypeMechanic:
		d.State = DoorStateClosed
	case d.State == DoorStateLocked && person.HasKeyCardInInventory(d.KeyString) && d.Type == DoorTypeElectronic:
		d.State = DoorStateClosed
	case d.State == DoorStateLocked && person.HasToolEquipped(core.ItemTypeMechanicalLockpick) && d.Type == DoorTypeMechanic:
		animationCompleted := false
		until := func() bool {
			return animationCompleted
		}
		onLockPicked := func() {
			animationCompleted = true
			d.State = DoorStateClosed
		}
		aic.SetEngaged(person, core.ActorStatusEngagedIllegal, until)
		animator.ActorEngagedAnimation(person, core.GlyphLockpick, d.Pos(), InteractionTime, onLockPicked)
	case d.State == DoorStateLocked && person.HasToolEquipped(core.ItemTypeElectronicalLockpick) && d.Type == DoorTypeElectronic:
		animationCompleted := false
		until := func() bool {
			return animationCompleted
		}
		onLockPicked := func() {
			animationCompleted = true
			d.State = DoorStateClosed
		}
		aic.SetEngaged(person, core.ActorStatusEngagedIllegal, until)
		animator.ActorEngagedAnimation(person, core.GlyphLockpickElectronic, d.Pos(), InteractionTime, onLockPicked)
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
	if d.State == DoorStateLocked && !d.IsUnlockableWithKeyFrom(person) && !d.IsUnlockableWithPickFrom(person) {
		return false
	}
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
	equippedItem := person.EquippedItem
	switch {
	case person == nil || equippedItem == nil:
		return false
	case d.Type == DoorTypeMechanic && equippedItem.Type == core.ItemTypeMechanicalLockpick:
		return true
	case d.Type == DoorTypeElectronic && equippedItem.Type == core.ItemTypeElectronicalLockpick:
		return true
	}
	return false
}

type WindowState uint8

const (
	WindowStateClosed WindowState = iota
	WindowStateOpen
	WindowStateBroken
)

type Window struct {
	State            WindowState
	position         geometry.Point
	OutsideOffset    geometry.Point // offset from the window to the outside
	DamageThreshold  int
	uniqueIdentifier string
	definedStyle     common.Style
}

func (w *Window) GetStyle() common.Style {
	return w.definedStyle
}

func (w *Window) SetStyle(style common.Style) {
	w.definedStyle = style
}

func (w *Window) Description() string {
	switch w.State {
	case WindowStateOpen:
		return "an open window"
	case WindowStateClosed:
		return "a closed window"
	case WindowStateBroken:
		return "a broken window"
	}
	return "a window"
}

func (w *Window) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {
	isRelevantDamageType := stim.Type() == stimuli.StimulusPiercingDamage || stim.Type() == stimuli.StimulusBluntDamage || stim.Type() == stimuli.StimulusExplosionDamage || stim.Type() == stimuli.StimulusFire

	isClosed := w.State == WindowStateClosed
	if isRelevantDamageType && isClosed && stim.Force() > w.DamageThreshold {
		w.State = WindowStateBroken
	}
	return
}

func (w *Window) Pos() geometry.Point {
	return w.position
}

func (w *Window) SetPos(pos geometry.Point) {
	w.position = pos
}

func (w *Window) Icon() rune {
	switch w.State {
	case WindowStateOpen:
		return core.GlyphOpenWindow
	case WindowStateBroken:
		return core.GlyphBrokenWindow
	default:
		return core.GlyphClosedWindow
	}
}

func (w *Window) Style(st common.Style) common.Style {
	return w.definedStyle
}

func (w *Window) Action(m services.Engine, person *core.Actor) {
	switch w.State {
	case WindowStateOpen:
		w.State = WindowStateClosed
	case WindowStateClosed:
		w.State = WindowStateOpen
	}
	return
}

func (w *Window) IsActionAllowed(m services.Engine, person *core.Actor) bool {
	if w.State == WindowStateBroken || w.IsPersonOutside(m, person) {
		return false
	}
	return true
}

func (w *Window) ActionDescription() string {
	if w.State == WindowStateOpen {
		return string(core.GlyphClosedWindow)
	}
	return string(core.GlyphOpenWindow)
}

func (w *Window) IsWalkable(*core.Actor) bool {
	return w.State != WindowStateClosed
}

func (w *Window) IsTransparent() bool {
	return true
}
func (w *Window) IsPassableForProjectile() bool {
	return true
}
func (w *Window) EncodeAsString() string {
	return w.uniqueIdentifier
}
func (w *Window) IsPersonOutside(m services.Engine, person *core.Actor) bool {
	return person.Pos() == w.Pos().Add(w.OutsideOffset)
}

func NewClosedWindowAt(identifier string, damageThreshold int) *Window {
	return &Window{State: WindowStateClosed, DamageThreshold: damageThreshold, uniqueIdentifier: identifier, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

func NewOpenWindowAt(identifier string, damageThreshold int) *Window {
	return &Window{State: WindowStateOpen, DamageThreshold: damageThreshold, uniqueIdentifier: identifier, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}

func NewBrokenWindowAt(identifier string) *Window {
	return &Window{State: WindowStateBroken, uniqueIdentifier: identifier, definedStyle: common.DefaultStyle.WithBg(common.Transparent)}
}
