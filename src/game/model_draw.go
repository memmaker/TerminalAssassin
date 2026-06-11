package game

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

func (m *Model) ApplyLighting(p geometry.Point, cell gridmap.MapCell[*core.Actor, *core.Item, services.Object], currentStyle common.Style) common.Style {
	lightScale := 0.75
	if !cell.TileType.IsTransparent {
		lightScale = 1.0
	}
	if currentStyle.Foreground == nil {
		currentStyle.Foreground = common.DefaultStyle.Foreground
	}
	if currentStyle.Background == nil {
		currentStyle.Background = common.DefaultStyle.Background
	}

	currentMap := m.gridMap
	lightAtCell := currentMap.LightAt(p)
	bgColor, fgColor := m.applyLightingToEnvironmentColors(lightScale, lightAtCell, currentStyle.Foreground, currentStyle.Background)
	return common.Style{Foreground: fgColor, Background: bgColor}
}

func (m *Model) applyLightingToEnvironmentColors(scale float64, light, fg, bg common.Color) (common.RGBAColor, common.RGBAColor) {
	bgColor := common.RGBAColor{
		R: bg.RValue() * (light.RValue() * scale),
		G: bg.GValue() * (light.GValue() * scale),
		B: bg.BValue() * (light.BValue() * scale),
		A: 1.0,
	}
	fgColor := common.RGBAColor{
		R: fg.RValue() * (light.RValue() * scale),
		G: fg.GValue() * (light.GValue() * scale),
		B: fg.BValue() * (light.BValue() * scale),
		A: 1.0,
	}
	return bgColor, fgColor
}
func (m *Model) DrawMapAtPosition(p geometry.Point, c gridmap.MapCell[*core.Actor, *core.Item, services.Object]) (rune, common.Style) {
	return m.drawAtPosition(p, c, false)
}

func (m *Model) DrawWorldAtPosition(p geometry.Point, c gridmap.MapCell[*core.Actor, *core.Item, services.Object]) (rune, common.Style) {
	return m.drawAtPosition(p, c, true)
}

func (m *Model) drawAtPosition(p geometry.Point, c gridmap.MapCell[*core.Actor, *core.Item, services.Object], showEntities bool) (rune, common.Style) {
	renderStyle := core.CurrentTheme.TileStyle(c.TileType)

	renderIcon := c.TileType.Icon()

	if len(c.Stimuli) > 0 {
		stimIcon, stimStyle := StimDrawInfo(c, renderStyle)
		renderStyle = stimStyle
		if renderIcon == core.GlyphGround {
			renderIcon = stimIcon
		}
	}

	switch {
	case !c.IsExplored:
		return ' ', common.DefaultStyle
	case showEntities && m.gridMap.Player != nil && p == m.gridMap.Player.Pos() && m.gridMap.Player.IsVisible():
		return '@', m.gridMap.Player.Style(renderStyle)
	case showEntities && c.Actor != nil && (*c.Actor).IsVisible():
		return (*c.Actor).Symbol(), (*c.Actor).Style(renderStyle)
	case showEntities && c.DownedActor != nil && (*c.DownedActor).IsVisible():
		return (*c.DownedActor).Symbol(), (*c.DownedActor).Style(renderStyle)
	case showEntities && c.Item != nil && !(*c.Item).Buried:
		return (*c.Item).Icon(), (*c.Item).Style(renderStyle)
	case m.GetMap().IsObjectAt(p):
		visibleObject := m.GetMap().ObjectAt(p)
		return visibleObject.Icon(), visibleObject.Style(renderStyle)
	}
	return renderIcon, renderStyle
}

func StimDrawInfo(c gridmap.MapCell[*core.Actor, *core.Item, services.Object], tileStyle common.Style) (rune, common.Style) {
	st := tileStyle
	icon := c.TileType.Icon()
	for _, s := range []stimuli.StimulusType{
		stimuli.StimulusFire,
		stimuli.StimulusHighVoltage,
		stimuli.StimulusSmoke,
		stimuli.StimulusSleep,
		stimuli.StimulusEmetic,
		stimuli.StimulusLethal,
		stimuli.StimulusBurnable,
		stimuli.StimulusWater,
		stimuli.StimulusBlood,
	} {
		if c.HasStim(s) {
			return stimStyleAndRune(s, st)
		}
	}
	return icon, st
}

func stimStyleAndRune(s stimuli.StimulusType, st common.Style) (rune, common.Style) {
	switch s {
	case stimuli.StimulusBlood:
		return core.GlyphBlood, st.WithFg(core.CurrentTheme.BloodForeground)
	case stimuli.StimulusWater:
		return core.GlyphWater, st.WithFg(core.CurrentTheme.WaterForeground).WithBg(core.CurrentTheme.WaterBackground)
	case stimuli.StimulusBurnable:
		return core.GlyphWater, st.WithFg(core.CurrentTheme.OilForeground).WithBg(core.CurrentTheme.OilBackground)
	case stimuli.StimulusSmoke:
		return core.GlyphFog, st.WithFg(common.NewRGBColorFromBytes(180, 180, 180)).WithBg(common.NewRGBColorFromBytes(80, 80, 80))
	case stimuli.StimulusFire:
		return core.GlyphFireOne, st.WithFg(core.CurrentTheme.FireForeground).WithBg(core.CurrentTheme.FireBackground)
	case stimuli.StimulusLethal:
		return core.GlyphFog, st.WithFg(core.CurrentTheme.LethalPoisonForeground)
	case stimuli.StimulusEmetic:
		return core.GlyphFog, st.WithFg(core.CurrentTheme.EmeticPoisonForeground)
	case stimuli.StimulusSleep:
		return core.GlyphFog, st.WithFg(core.CurrentTheme.SleepPoisonForeground)
	case stimuli.StimulusHighVoltage:
		return core.GlyphElectric, st.WithFg(core.CurrentTheme.ElectricityForeground)
	}
	return core.GlyphGround, st
}
