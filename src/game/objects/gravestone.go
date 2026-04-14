package objects

import (
    "strings"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/game/stimuli"
    "github.com/memmaker/terminal-assassin/geometry"
)

const gravestonePrefix = "gravestone|"

func NewGravestone(inscription string) *Gravestone {
    return &Gravestone{
        inscription: inscription,
    }
}

func GravestoneFromEncoded(encoded string) *Gravestone {
    inscription := strings.TrimPrefix(encoded, gravestonePrefix)
    return NewGravestone(inscription)
}

type Gravestone struct {
    inscription string
    position    geometry.Point
}

// EncodeAsString serialises the gravestone so it survives a save/load cycle.
// The inscription is embedded after the type prefix.
func (g *Gravestone) EncodeAsString() string {
    return gravestonePrefix + g.inscription
}

// Description is shown in the mouseover tooltip.
func (g *Gravestone) Description() string {
    if g.inscription == "" {
        return "A weathered gravestone."
    }
    return g.inscription
}

// GetText implements services.Textable.
func (g *Gravestone) GetText() string {
    return g.inscription
}

// SetText implements services.Textable — updates the inscription in place.
func (g *Gravestone) SetText(text string) {
    g.inscription = text
}

func (g *Gravestone) Icon() rune {
    return core.GlyphGravestone
}

func (g *Gravestone) Pos() geometry.Point {
    return g.position
}

func (g *Gravestone) SetPos(pos geometry.Point) {
    g.position = pos
}

func (g *Gravestone) Style(st common.Style) common.Style {
    return common.Style{Foreground: core.CurrentTheme.ObjectForeground, Background: st.Background}
}

// IsWalkable returns false — gravestones block movement.
func (g *Gravestone) IsWalkable(_ *core.Actor) bool {
    return false
}

// IsTransparent returns true — you can see past a gravestone.
func (g *Gravestone) IsTransparent() bool {
    return true
}

func (g *Gravestone) IsPassableForProjectile() bool {
    return false
}

// Action does nothing — gravestones are read-only scenery.
func (g *Gravestone) Action(_ services.Engine, _ *core.Actor) {}

func (g *Gravestone) IsActionAllowed(_ services.Engine, _ *core.Actor) bool {
    return false
}

func (g *Gravestone) ApplyStimulus(_ services.Engine, _ stimuli.Stimulus) {}
