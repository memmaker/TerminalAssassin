package objects

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

// LiquidLeaker
// on puncture stimulus: spill liquid
// when liquid is spilled, on firestarter stimulus: explode
// this will also represent any other object that cannot be moved and containsPlayer a burnable liquid
// eg. vehicles, oil drums, etc.
type LiquidLeaker struct {
	position            geometry.Point
	symbol              rune
	Name                string
	HasLeaked           bool
	CanBeLeakedByHand   bool
	LeakingStimulusType stimuli.StimulusType
	definedStyle        common.Style
}

func (l *LiquidLeaker) GetStyle() common.Style {
	return l.definedStyle
}

func (l *LiquidLeaker) SetStyle(style common.Style) {
	l.definedStyle = style
}

func (l *LiquidLeaker) EncodeAsString() string {
	return l.Name
}

func (l *LiquidLeaker) Description() string {
	return l.Name
}

func (l *LiquidLeaker) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {
	switch {
	case stim.Type() == stimuli.StimulusPiercingDamage && !l.HasLeaked:
		l.StartLeak(m)
	}
	return
}

func (l *LiquidLeaker) StartLeak(m services.Engine) {
	l.HasLeaked = true
	game := m.GetGame()
	effectLeak := stimuli.EffectLeak(l.LeakingStimulusType, 50, 6)
	game.ApplyDelayed(l.Pos(), core.NewEffectSourceFromObject(l), effectLeak, 0.5)
}

func (l *LiquidLeaker) Icon() rune {
	return l.symbol
}

func (l *LiquidLeaker) Style(st common.Style) common.Style {
	st = l.definedStyle.WithBg(st.Background)
	switch l.LeakingStimulusType {
	case stimuli.StimulusBurnableLiquid:
		st = st.WithFg(core.ColorFromCode(core.ColorBurnableForeground))
	case stimuli.StimulusWater:
		st = st.WithFg(core.ColorFromCode(core.ColorWater))
	case stimuli.StimulusLethalPoison:
		st = st.WithFg(core.ColorFromCode(core.ColorPoisonLethal))
	case stimuli.StimulusEmeticPoison:
		st = st.WithFg(core.ColorFromCode(core.ColorPoisonEmetic))
	}
	return st
}

func (l *LiquidLeaker) Action(m services.Engine, person *core.Actor) {
	if !l.HasLeaked {
		l.StartLeak(m)
	}
}

func (l *LiquidLeaker) IsActionAllowed(m services.Engine, person *core.Actor) bool {
	return l.CanBeLeakedByHand && !l.HasLeaked
}

func (l *LiquidLeaker) IsWalkable(*core.Actor) bool {
	return false
}

func (l *LiquidLeaker) IsTransparent() bool {
	return true
}
func (l *LiquidLeaker) IsPassableForProjectile() bool {
	return false
}
func (l *LiquidLeaker) Pos() geometry.Point {
	return l.position
}

func (l *LiquidLeaker) SetPos(pos geometry.Point) {
	l.position = pos
}

func NewLiquidContainer(description string, symbol rune, leakedStim stimuli.StimulusType) *LiquidLeaker {
	return &LiquidLeaker{
		symbol:              symbol,
		Name:                description,
		LeakingStimulusType: leakedStim,
		definedStyle:        common.DefaultStyle.WithBg(common.Transparent),
	}
}

func NewLiquidFaucet(description string, symbol rune, leakedStim stimuli.StimulusType) *LiquidLeaker {
	return &LiquidLeaker{
		symbol:              symbol,
		Name:                description,
		LeakingStimulusType: leakedStim,
		CanBeLeakedByHand:   true,
		definedStyle:        common.DefaultStyle.WithBg(common.Transparent),
	}
}
