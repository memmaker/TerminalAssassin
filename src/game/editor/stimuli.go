package editor

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
)

func (g *GameStateEditor) openStimuliMenu() {
	menuItems := []services.MenuItem{
		{
			Label: "Clear stimuli",
			Icon:  'X',
			Handler: g.setBrushHandler(addStimuliUI, 'X', func(pos geometry.Point) {
				g.engine.GetGame().GetMap().RemoveAllStimuliFromTile(pos)
			}),
		},
		{
			Label: "Fire",
			Icon:  core.GlyphFireOne,
			Handler: g.setBrushHandler(addStimuliUI, core.GlyphFireOne, func(pos geometry.Point) {
				g.engine.GetGame().GetMap().AddStimulusToTile(pos, stimuli.Stim{
					StimType:  stimuli.StimulusFire,
					StimForce: 80,
				})
			}),
		},
		{
			Label: "Water",
			Icon:  core.GlyphWater,
			Handler: g.setBrushHandler(addStimuliUI, core.GlyphWater, func(pos geometry.Point) {
				g.engine.GetGame().GetMap().AddStimulusToTile(pos, stimuli.Stim{
					StimType:  stimuli.StimulusWater,
					StimForce: 80,
				})
			}),
		},
		{
			Label: "Blood",
			Icon:  core.GlyphBlood,
			Handler: g.setBrushHandler(addStimuliUI, core.GlyphBlood, func(pos geometry.Point) {
				g.engine.GetGame().GetMap().AddStimulusToTile(pos, stimuli.Stim{
					StimType:  stimuli.StimulusBlood,
					StimForce: 10,
				})
			}),
		},
		{
			Label: "Burnable",
			Icon:  core.GlyphWater,
			Handler: g.setBrushHandler(addStimuliUI, core.GlyphWater, func(pos geometry.Point) {
				g.engine.GetGame().GetMap().AddStimulusToTile(pos, stimuli.Stim{
					StimType:  stimuli.StimulusBurnableLiquid,
					StimForce: 80,
				})
			}),
		},
	}
	g.OpenMenuBarDropDown("Stimuli", (2*7)-2, menuItems)
}
