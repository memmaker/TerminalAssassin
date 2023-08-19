package objects

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

func NewTriggerObject(description string, symbol rune) *TriggerObject {
	return &TriggerObject{
		symbol:       symbol,
		Name:         description,
		definedStyle: common.DefaultStyle.WithBg(common.Transparent),
	}
}

// TriggerObject
// has on/off state
// can be turned on/off by hand
// will emit trigger event, if turned on or off
type TriggerObject struct {
	position     geometry.Point
	symbol       rune
	Name         string
	definedStyle common.Style
	state        DeviceState
	triggerKey   string
}

func (t *TriggerObject) GetKey() string {
	return t.triggerKey
}

func (t *TriggerObject) SetKey(key string) {
	t.triggerKey = key
}

func (t *TriggerObject) GetStyle() common.Style {
	return t.definedStyle
}

func (t *TriggerObject) SetStyle(style common.Style) {
	t.definedStyle = style
}

func (t *TriggerObject) EncodeAsString() string {
	return fmt.Sprintf("Trigger: %s", t.Name)
}

func (t *TriggerObject) Description() string {
	return t.Name
}

func (t *TriggerObject) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {
	switch stim.Type() {
	case stimuli.StimulusPiercingDamage:
		t.state = DeviceStateBroken
	case stimuli.StimulusBluntDamage:
		t.state = DeviceStateBroken
	case stimuli.StimulusFire:
		t.state = DeviceStateBroken
	case stimuli.StimulusWater:
		t.state = DeviceStateBroken
	}
}

func (t *TriggerObject) Icon() rune {
	return t.symbol
}

func (t *TriggerObject) Style(st common.Style) common.Style {
	st = t.definedStyle.WithBg(st.Background)
	if t.state == DeviceStateOn {
		st = st.WithFg(common.Green)
	} else if t.state == DeviceStateBroken {
		st = st.WithFg(common.Red)
	}
	return st
}

func (t *TriggerObject) Action(m services.Engine, person *core.Actor) {
	t.toggleState(m)
}

func (t *TriggerObject) toggleState(m services.Engine) {
	if t.state == DeviceStateOff {
		t.state = DeviceStateOn
	} else if t.state == DeviceStateOn {
		t.state = DeviceStateOff
	}
	m.PublishEvent(services.TriggerEvent{Key: t.triggerKey})
}

func (t *TriggerObject) IsActionAllowed(m services.Engine, person *core.Actor) bool {
	return t.state != DeviceStateBroken
}

func (t *TriggerObject) IsWalkable(*core.Actor) bool {
	return false
}
func (t *TriggerObject) IsPassableForProjectile() bool {
	return false
}

func (t *TriggerObject) IsTransparent() bool {
	return true
}

func (t *TriggerObject) Pos() geometry.Point {
	return t.position
}

func (t *TriggerObject) SetPos(pos geometry.Point) {
	t.position = pos
}
