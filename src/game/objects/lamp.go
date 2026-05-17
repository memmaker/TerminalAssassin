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
	position    geometry.Point
	Name        string
	state       DeviceState
	engine      services.Engine
	lightSource *gridmap.LightSource
}

func CreateLamp(engine services.Engine, description string) *Lamp {
	lamp := &Lamp{
		Name:   description,
		state:  DeviceStateOn,
		engine: engine,
	}
	// Note: Initialize() must be called after SetPos sets the position
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
	// Lazy initialization: add light if not yet initialized
	if l.lightSource == nil && l.state == DeviceStateOn && l.engine != nil && l.position.X != 0 && l.position.Y != 0 {
		l.addLightToMap(l.engine)
	}
	return core.GlyphStreetLight
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
	return l.state != DeviceStateBroken
}

func (l *Lamp) IsWalkable(*core.Actor) bool {
	return l.state == DeviceStateBroken
}

func (l *Lamp) IsPassableForProjectile() bool {
	return l.state == DeviceStateBroken
}

func (l *Lamp) IsTransparent() bool {
	return true
}

func (l *Lamp) Pos() geometry.Point {
	return l.position
}

func (l *Lamp) SetPos(pos geometry.Point) {
	l.position = pos
}

// OnRemoved implements services.Removable — cleans up the dynamic light when
// the lamp object is deleted from the map (e.g. in the editor).
func (l *Lamp) OnRemoved(m services.Engine) {
	l.removeLightFromMap(m)
}

// Initialize should be called after the lamp is placed and SetPos has been called
func (l *Lamp) Initialize() {
	if l.state == DeviceStateOn && l.engine != nil && l.lightSource == nil {
		l.addLightToMap(l.engine)
	}
}

func (l *Lamp) addLightToMap(m services.Engine) {
	if l.lightSource != nil {
		return // light already exists
	}
	currentMap := m.GetGame().GetMap()
	// Adopt a pre-existing light at this position (e.g. loaded from dynamic_lights.txt)
	// so that removeLightFromMap can remove it by pointer comparison.
	if existing, ok := currentMap.DynamicLights[l.position]; ok {
		l.lightSource = existing
		return
	}
	l.lightSource = &gridmap.LightSource{
		Pos:          l.position,
		Radius:       7,
		Color:        common.RGBAColor{R: 1.0, G: 0.9, B: 0.7, A: 1.0},
		MaxIntensity: 1.0,
	}
	currentMap.AddDynamicLightSource(l.position, l.lightSource)
	currentMap.UpdateDynamicLights()
}

func (l *Lamp) removeLightFromMap(m services.Engine) {
	if l.lightSource == nil {
		return // no light to remove
	}
	currentMap := m.GetGame().GetMap()
	// Only remove if the light at this position is our own light source
	if existingLight, exists := currentMap.DynamicLights[l.position]; exists && existingLight == l.lightSource {
		currentMap.RemoveDynamicLightAt(l.position)
	}
	l.lightSource = nil
}
