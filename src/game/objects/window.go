package objects

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

type WindowState uint8

const (
	WindowStateClosed WindowState = iota
	WindowStateOpen
	WindowStateBroken
)

type Window struct {
	State            WindowState
	position         geometry.Point
	DamageThreshold  int
	uniqueIdentifier string
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
	return common.Style{Foreground: core.CurrentTheme.ObjectForeground, Background: st.Background}
}

func (w *Window) Action(m services.Engine, person *core.Actor) {
	switch w.State {
	case WindowStateOpen:
		w.State = WindowStateClosed
	case WindowStateClosed:
		w.State = WindowStateOpen
	}
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
	zone := m.GetGame().GetMap().ZoneAt(person.Pos())
	return zone == nil || zone.IsPublic()
}

func NewClosedWindowAt(identifier string, damageThreshold int) *Window {
	return &Window{State: WindowStateClosed, DamageThreshold: damageThreshold, uniqueIdentifier: identifier}
}

func NewOpenWindowAt(identifier string, damageThreshold int) *Window {
	return &Window{State: WindowStateOpen, DamageThreshold: damageThreshold, uniqueIdentifier: identifier}
}

func NewBrokenWindowAt(identifier string) *Window {
	return &Window{State: WindowStateBroken, uniqueIdentifier: identifier}
}
