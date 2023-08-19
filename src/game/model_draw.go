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
	st := c.TileType.Style()
	tileIcon := c.TileType.Icon()

	if c.Stimuli != nil && len(c.Stimuli) > 0 {
		icon, style := StimDrawInfo(c, st)
		st = style
		if tileIcon == core.GlyphGround {
			tileIcon = icon
		}
	}

	switch {
	case !c.IsExplored:
		return ' ', common.DefaultStyle
	case m.GetMap().IsObjectAt(p):
		visibleObject := m.GetMap().ObjectAt(p)
		return visibleObject.Icon(), visibleObject.Style(st)
	}
	return tileIcon, c.TileType.Style()
}
func (m *Model) DrawWorldAtPosition(p geometry.Point, c gridmap.MapCell[*core.Actor, *core.Item, services.Object]) (rune, common.Style) {
	st := c.TileType.Style()
	tileIcon := c.TileType.Icon()

	if c.Stimuli != nil && len(c.Stimuli) > 0 {
		icon, style := StimDrawInfo(c, st) // this is the most expensive function call in this function
		st = style
		if tileIcon == core.GlyphGround {
			tileIcon = icon
		}
	}
	switch {
	case !c.IsExplored:
		return ' ', common.DefaultStyle
	case m.gridMap.Player != nil && p == m.gridMap.Player.Pos() && m.gridMap.Player.IsVisible():
		return '@', m.gridMap.Player.Style(st)
	case c.Actor != nil && (*c.Actor).IsVisible():
		return (*c.Actor).Symbol(), (*c.Actor).Style(st)
	case c.DownedActor != nil && (*c.DownedActor).IsVisible():
		return (*c.DownedActor).Symbol(), (*c.DownedActor).Style(st)
	case m.GetMap().IsItemAt(p):
		visibleItem := m.GetMap().ItemAt(p)
		return visibleItem.Icon(), visibleItem.Style(st)
	case c.IsExplored && m.GetMap().IsObjectAt(p):
		visibleObject := m.GetMap().ObjectAt(p)
		return visibleObject.Icon(), visibleObject.Style(st)
	}
	return tileIcon, st
}

func StimDrawInfo(c gridmap.MapCell[*core.Actor, *core.Item, services.Object], tileStyle common.Style) (rune, common.Style) {
	// fire should always determine the background color, if it's present
	// electricity should always define foreground color, if it's present
	// if not electricity, poison should always define foreground color, if it's present
	// burnable liquid should only define background color, if no water and no fire is present
	stimStyle := tileStyle
	icon := c.TileType.Icon()
	if c.HasStim(stimuli.StimulusBlood) && c.ForceOfStim(stimuli.StimulusBlood) > 15 {
		stimStyle = stimStyle.WithFg(core.ColorFromCode(core.ColorBrightRed)).WithBg(core.ColorFromCode(core.ColorMapRed))
		icon = runeFromStimType(stimuli.StimulusBlood)
	} else if c.HasStim(stimuli.StimulusFire) {
		stimStyle = styleFromStimType(stimuli.StimulusFire, stimStyle)
		icon = runeFromStimType(stimuli.StimulusFire)
	} else if c.HasStim(stimuli.StimulusWater) {
		stimStyle = styleFromStimType(stimuli.StimulusWater, stimStyle)
		icon = runeFromStimType(stimuli.StimulusWater)
	} else if c.HasStim(stimuli.StimulusBurnableLiquid) {
		stimStyle = styleFromStimType(stimuli.StimulusBurnableLiquid, stimStyle)
		icon = runeFromStimType(stimuli.StimulusBurnableLiquid)
	}

	if c.HasStim(stimuli.StimulusHighVoltage) {
		stimStyle = styleFromStimType(stimuli.StimulusHighVoltage, stimStyle)
		icon = runeFromStimType(stimuli.StimulusHighVoltage)
	} else if c.HasStim(stimuli.StimulusLethalPoison) {
		stimStyle = styleFromStimType(stimuli.StimulusLethalPoison, stimStyle)
		icon = runeFromStimType(stimuli.StimulusLethalPoison)
	} else if c.HasStim(stimuli.StimulusEmeticPoison) {
		stimStyle = styleFromStimType(stimuli.StimulusEmeticPoison, stimStyle)
		icon = runeFromStimType(stimuli.StimulusEmeticPoison)
	} else if c.HasStim(stimuli.StimulusBlood) && c.ForceOfStim(stimuli.StimulusBlood) <= 15 {
		stimStyle = stimStyle.WithFg(core.ColorFromCode(core.ColorBlood))
		icon = core.GlyphBlood
	}
	return icon, stimStyle
}

func styleFromStimType(stimulusType stimuli.StimulusType, st common.Style) common.Style {
	switch stimulusType {
	case stimuli.StimulusWater:
		return st.WithFg(core.ColorFromCode(core.ColorWater)).WithBg(core.ColorFromCode(core.ColorWaterBackground))
	case stimuli.StimulusBurnableLiquid:
		return st.WithFg(core.ColorFromCode(core.ColorBurnableForeground)).WithBg(core.ColorFromCode(core.ColorBurnableBackground))
	case stimuli.StimulusFire:
		return st.WithFg(core.ColorFromCode(core.ColorExplosionDark)).WithBg(core.ColorFromCode(core.ColorExplosionLight))
	case stimuli.StimulusLethalPoison:
		return st.WithFg(core.ColorFromCode(core.ColorPoisonLethal))
	case stimuli.StimulusEmeticPoison:
		return st.WithFg(core.ColorFromCode(core.ColorPoisonEmetic))
	case stimuli.StimulusHighVoltage:
		return st.WithFg(core.ColorFromCode(core.ColorElectricForeground)) //.WithBg(ColorFromCode(ColorElectricBackground))
	}
	return st
}

func runeFromStimType(stimulusType stimuli.StimulusType) rune {
	switch stimulusType {
	case stimuli.StimulusBlood:
		fallthrough
	case stimuli.StimulusBurnableLiquid:
		fallthrough
	case stimuli.StimulusWater:
		return core.GlyphWater
	case stimuli.StimulusFire:
		return core.GlyphFireOne
	case stimuli.StimulusHighVoltage:
		return core.GlyphElectric
	case stimuli.StimulusLethalPoison:
		fallthrough
	case stimuli.StimulusEmeticPoison:
		return core.GlyphFog
	}
	return core.GlyphGround
}
