package objects

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

type AlarmState int

const (
	AlarmStateActive AlarmState = iota
	AlarmStateBroken
	AlarmStateTriggered
)

// AlarmObject is placed by map designers. Guards in AlarmRun will activate it
// when they detect a dangerous sighting and no easier alert is available.
type AlarmObject struct {
	position geometry.Point
	state    AlarmState
	Name     string
}

func NewAlarmObject(name string) *AlarmObject {
	return &AlarmObject{Name: name, state: AlarmStateActive}
}

func (a *AlarmObject) IsActiveAlarm() bool { return a.state == AlarmStateActive }

func (a *AlarmObject) TriggerAlarm(engine services.Engine, sightingLocation geometry.Point) {
	if a.state != AlarmStateActive {
		return
	}
	a.state = AlarmStateTriggered
	println("ALARM TRIGGERED:", a.Name, "sighting at", sightingLocation.String())
	engine.PublishEvent(services.AlarmTriggeredEvent{SightingLocation: sightingLocation})

	// silence all remaining active alarms — only one can fire per mission
	for _, obj := range engine.GetGame().GetMap().AllObjects {
		if dev, ok := obj.(services.AlarmDevice); ok && dev.IsActiveAlarm() {
			dev.SilenceAlarm()
		}
	}
}

func (a *AlarmObject) SilenceAlarm() {
	a.state = AlarmStateTriggered
}

func (a *AlarmObject) Action(engine services.Engine, person *core.Actor) {
	// Player-triggered — same as guard trigger but without a known sighting location
	a.TriggerAlarm(engine, person.Pos())
}

func (a *AlarmObject) IsActionAllowed(_ services.Engine, _ *core.Actor) bool {
	return a.state == AlarmStateActive
}

func (a *AlarmObject) ApplyStimulus(_ services.Engine, stim stimuli.Stimulus) {
	switch stim.Type() {
	case stimuli.StimulusPiercingDamage, stimuli.StimulusBluntDamage,
		stimuli.StimulusFire, stimuli.StimulusWater, stimuli.StimulusExplosionDamage:
		a.state = AlarmStateBroken
	}
}

func (a *AlarmObject) Style(st common.Style) common.Style {
	fg := core.CurrentTheme.ObjectForeground
	switch a.state {
	case AlarmStateTriggered:
		fg = core.CurrentTheme.DeviceBrokenForeground
	case AlarmStateBroken:
		fg = core.CurrentTheme.DeviceBrokenForeground
	case AlarmStateActive:
		fg = core.CurrentTheme.DeviceOnForeground
	}
	return common.Style{Foreground: fg, Background: st.Background}
}

func (a *AlarmObject) Icon() rune                    { return core.GlyphAlarm }
func (a *AlarmObject) Pos() geometry.Point           { return a.position }
func (a *AlarmObject) SetPos(p geometry.Point)       { a.position = p }
func (a *AlarmObject) Description() string           { return a.Name }
func (a *AlarmObject) EncodeAsString() string        { return a.Name }
func (a *AlarmObject) IsWalkable(*core.Actor) bool   { return false }
func (a *AlarmObject) IsTransparent() bool           { return true }
func (a *AlarmObject) IsPassableForProjectile() bool { return false }
