package objects

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Cage struct {
	position geometry.Point
	isOpen   bool
}

func NewCage() *Cage {
	return &Cage{}
}

func (c *Cage) Style(st common.Style) common.Style {
	return common.Style{Foreground: core.CurrentTheme.ObjectForeground, Background: st.Background}
}

func (c *Cage) Icon() rune {
	if c.isOpen {
		return core.GlyphCageOpen
	}
	return core.GlyphCageClosed
}

func (c *Cage) Description() string {
	if c.isOpen {
		return "an open cage"
	}
	return "a closed cage"
}

func (c *Cage) IsWalkable(*core.Actor) bool  { return c.isOpen }
func (c *Cage) IsTransparent() bool          { return c.isOpen }
func (c *Cage) IsPassableForProjectile() bool { return c.isOpen }

func (c *Cage) Pos() geometry.Point         { return c.position }
func (c *Cage) SetPos(p geometry.Point)     { c.position = p }
func (c *Cage) EncodeAsString() string      { return "cage" }

func (c *Cage) IsActionAllowed(_ services.Engine, _ *core.Actor) bool { return !c.isOpen }

func (c *Cage) ActionDescription() string { return "open cage" }

func (c *Cage) Action(m services.Engine, _ *core.Actor) {
	c.open(m)
}

func (c *Cage) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {
	if c.isOpen {
		return
	}
	isPiercing := stim.Type() == stimuli.StimulusPiercingDamage
	isBlunt := stim.Type() == stimuli.StimulusBluntDamage
	if (isPiercing || isBlunt) && stim.Force() > 5 {
		c.open(m)
	}
}

func (c *Cage) open(m services.Engine) {
	if c.isOpen {
		return
	}
	c.isOpen = true

	currentMap := m.GetGame().GetMap()
	// find a free adjacent tile
	spawnPos, found := findFreeAdjacentTile(m, c.position)
	if !found {
		return
	}

	predator := core.NewActor("Predator")
	predator.Type = core.ActorTypePredator
	predator.Team = "Predator"
	predator.MapPos = spawnPos
	predator.LastPos = spawnPos

	currentMap.AddActor(predator, spawnPos)
	m.GetGame().InitActor(predator)
}

func findFreeAdjacentTile(m services.Engine, pos geometry.Point) (geometry.Point, bool) {
	currentMap := m.GetGame().GetMap()
	nb := geometry.Neighbors{}
	candidates := nb.All(pos, func(p geometry.Point) bool {
		return currentMap.Contains(p) && !currentMap.IsActorAt(p) && currentMap.IsWalkable(p)
	})
	if len(candidates) == 0 {
		return geometry.Point{}, false
	}
	return candidates[0], true
}




