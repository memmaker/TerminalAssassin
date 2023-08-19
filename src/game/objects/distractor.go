package objects

import (
	"fmt"
	"math"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

type DeviceState int

const (
	DeviceStateOff DeviceState = iota
	DeviceStateOn
	DeviceStateBroken
)

func NewZoneDistractor(description string, symbol rune) *ZoneDistractor {
	return &ZoneDistractor{
		symbol:       symbol,
		Name:         description,
		definedStyle: common.DefaultStyle.WithBg(common.Transparent),
	}
}

// ZoneDistractor
// has on/off state
// can be turned on/off by hand
// will emit sound regularly, if turned on
type ZoneDistractor struct {
	position     geometry.Point
	symbol       rune
	Name         string
	definedStyle common.Style
	state        DeviceState
}

func (r *ZoneDistractor) GetStyle() common.Style {
	return r.definedStyle
}

func (r *ZoneDistractor) SetStyle(style common.Style) {
	r.definedStyle = style
}

func (r *ZoneDistractor) EncodeAsString() string {
	return r.Name
}

func (r *ZoneDistractor) Description() string {
	return r.Name
}

func (r *ZoneDistractor) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {
	switch stim.Type() {
	case stimuli.StimulusPiercingDamage:
		r.state = DeviceStateBroken
	case stimuli.StimulusBluntDamage:
		r.state = DeviceStateBroken
	case stimuli.StimulusFire:
		r.state = DeviceStateBroken
	case stimuli.StimulusWater:
		r.state = DeviceStateBroken
	}
}

func (r *ZoneDistractor) Icon() rune {
	return r.symbol
}

func (r *ZoneDistractor) Style(st common.Style) common.Style {
	st = r.definedStyle.WithBg(st.Background)
	if r.state == DeviceStateOn {
		st = st.WithFg(common.Green)
	} else if r.state == DeviceStateBroken {
		st = st.WithFg(common.Red)
	}
	return st
}

func (r *ZoneDistractor) Action(m services.Engine, person *core.Actor) {
	r.toggleDistraction(m)
}

func (r *ZoneDistractor) toggleDistraction(m services.Engine) {
	if r.state == DeviceStateOff {
		r.state = DeviceStateOn
	} else if r.state == DeviceStateOn {
		r.state = DeviceStateOff
	}
	r.tryToDistract(m)
}

func (r *ZoneDistractor) tryToDistract(m services.Engine) {
	if r.state == DeviceStateOn {
		target := r.findActorToDistract(m)
		if target == nil {
			return
		}
		aic := m.GetAI()

		report := aic.ReportIncident(target, r.Pos(), core.ObservationDeviceDistraction)
		if aic.TryRegisterHandler(target, report) {
			report.RegisteredHandler = target
			aic.SwitchToInvestigation(target, report)
			println(fmt.Sprintf("ZoneDistractor is attracting %s", target.DebugDisplayName()))
		}

		m.Schedule(60*5, func() {
			r.tryToDistract(m)
		})
	}
}

func (r *ZoneDistractor) findActorToDistract(m services.Engine) *core.Actor {
	currentMap := m.GetGame().GetMap()
	zone := currentMap.ZoneAt(r.Pos())
	minDist := math.MaxInt
	var nearestActor *core.Actor

	for _, actor := range currentMap.Actors() {
		if !actor.CanBeDistracted() {
			continue
		}
		actorStartZone := currentMap.ZoneAt(actor.AI.StartPosition)
		actorCurrentZone := currentMap.ZoneAt(actor.Pos())
		if actorStartZone == zone && actorCurrentZone == zone {
			dist := geometry.DistanceManhattan(actor.Pos(), r.Pos())
			if dist < minDist {
				minDist = dist
				nearestActor = actor
			}
		}
	}
	return nearestActor
}

func (r *ZoneDistractor) IsActionAllowed(m services.Engine, person *core.Actor) bool {
	return r.state != DeviceStateBroken
}

func (r *ZoneDistractor) IsWalkable(*core.Actor) bool {
	return false
}
func (r *ZoneDistractor) IsPassableForProjectile() bool {
	return false
}

func (r *ZoneDistractor) IsTransparent() bool {
	return true
}

func (r *ZoneDistractor) Pos() geometry.Point {
	return r.position
}

func (r *ZoneDistractor) SetPos(pos geometry.Point) {
	r.position = pos
}
