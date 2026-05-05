package objects

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

type Lamp struct {
	position geometry.Point
	Name     string
	state    DeviceState
	engine   services.Engine
}

func CreateLamp(engine services.Engine, description string) *Lamp {
	lamp := &Lamp{
		Name:   description,
		state:  DeviceStateOn,
		engine: engine,
	}
	return lamp
}

func NewLamp(description string) *Lamp {
	return &Lamp{
		Name:  description,
		state: DeviceStateOn,
	}
}

func (l *Lamp) EncodeAsString() string {
	return l.Name
}

func (l *Lamp) Description() string {
	return l.Name
}

func (l *Lamp) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {
	switch stim.Type() {
	case stimuli.StimulusPiercingDamage, stimuli.StimulusBluntDamage, stimuli.StimulusFire, stimuli.StimulusWater:
		if l.state != DeviceStateBroken {
			l.state = DeviceStateBroken
			l.removeLightFromMap(m)
		}
	}
}

func (l *Lamp) Icon() rune {
	return core.GlyphLamp
}

func (l *Lamp) Style(st common.Style) common.Style {
	st = common.Style{Foreground: core.CurrentTheme.ObjectForeground, Background: st.Background}
	if l.state == DeviceStateOn {
		st = st.WithFg(core.CurrentTheme.DeviceOnForeground)
	} else if l.state == DeviceStateBroken {
		st = st.WithFg(core.CurrentTheme.DeviceBrokenForeground)
	}
	return st
}

func (l *Lamp) Action(m services.Engine, person *core.Actor) {
	if l.state == DeviceStateBroken {
		return
	}
	if l.state == DeviceStateOff {
		l.state = DeviceStateOn
		l.addLightToMap(m)
	} else {
		l.state = DeviceStateOff
		l.removeLightFromMap(m)
	}
	m.GetGame().PrintMessage("You toggled the lamp.")
}

func (l *Lamp) IsActionAllowed(m services.Engine, person *core.Actor) bool {
	return true
}

func (l *Lamp) IsWalkable(*core.Actor) bool {
	return false
}

func (l *Lamp) IsPassableForProjectile() bool {
	return false
}

func (l *Lamp) IsTransparent() bool {
	return true
}

func (l *Lamp) Pos() geometry.Point {
	return l.position
}

func (l *Lamp) SetPos(pos geometry.Point) {
	l.position = pos
	if l.state == DeviceStateOn && l.engine != nil {
		l.addLightToMap(l.engine)
	}
}

func (l *Lamp) addLightToMap(m services.Engine) {
	currentMap := m.GetGame().GetMap()
	lightSource := &gridmap.LightSource{
		Pos:          l.position,
		Radius:       7,
		Color:        common.RGBAColor{R: 1.0, G: 0.9, B: 0.7, A: 1.0},
		MaxIntensity: 1.0,
	}
	currentMap.AddDynamicLightSource(l.position, lightSource)
	currentMap.UpdateDynamicLights()
}

func (l *Lamp) removeLightFromMap(m services.Engine) {
	currentMap := m.GetGame().GetMap()
	currentMap.RemoveDynamicLightAt(l.position)
}
