package objects

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Boulder struct {
	position     geometry.Point
	symbol       rune
	Name         string
	definedStyle common.Style
	hasFallen    bool
	key          string
}

func (b *Boulder) Style(st common.Style) common.Style {
	return b.definedStyle
}

func (b *Boulder) Action(m services.Engine, person *core.Actor) {
	b.Activate(m)
}

func (b *Boulder) IsActionAllowed(m services.Engine, person *core.Actor) bool {
	return false //!b.hasFallen
}

func (b *Boulder) IsWalkable(actor *core.Actor) bool {
	return !b.hasFallen
}

func (b *Boulder) IsTransparent() bool {
	return !b.hasFallen
}

func (b *Boulder) IsPassableForProjectile() bool {
	return false
}

func (b *Boulder) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {

}

func (b *Boulder) Description() string {
	return b.Name
}

func (b *Boulder) Pos() geometry.Point {
	return b.position
}

func (b *Boulder) Icon() rune {
	if b.hasFallen {
		return b.symbol
	}
	return 'X'
}

func (b *Boulder) SetPos(point geometry.Point) {
	b.position = point
}
func (b *Boulder) EncodeAsString() string {
	return fmt.Sprintf("Boulder: %s", b.Name)
}

func (b *Boulder) SetStyle(style common.Style) {
	b.definedStyle = style
}

func (b *Boulder) GetStyle() common.Style {
	return b.definedStyle
}

func (b *Boulder) GetKey() string {
	return b.key
}

func (b *Boulder) SetKey(key string) {
	b.key = key
}

func (b *Boulder) Activate(engine services.Engine) {
	if !b.hasFallen {
		game := engine.GetGame()
		game.Apply(b.Pos(), core.NewEffectSourceFromObject(b), stimuli.StimEffect{Stimuli: []stimuli.Stimulus{
			stimuli.Stim{StimType: stimuli.StimulusBluntDamage, StimForce: 100},
		}})
		b.hasFallen = true
	}
}
func NewBoulder(description string, symbol rune) *Boulder {
	boulder := &Boulder{
		symbol:       symbol,
		Name:         description,
		definedStyle: common.DefaultStyle.WithBg(common.Transparent),
	}
	return boulder
}
