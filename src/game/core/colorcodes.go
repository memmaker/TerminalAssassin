package core

import "github.com/memmaker/terminal-assassin/common"

type ColorCode uint16

const (
	ColorPlayer ColorCode = iota
	ColorMapBackground
	ColorMapBackgroundLight
	ColorExplosionDark
	ColorExplosionLight
	ColorVisionCone
	ColorRedIndicator
	ColorMapForeground
	ColorLOS
	ColorActor
	ColorCivilian
	ColorGuard
	ColorEnforcer
	ColorTarget
	ColorDead
	ColorBlood
	ColorBrightRed
	ColorGood
	ColorTree
	ColorWarning
	ColorFoVSource
	ColorBlackBackground
	ColorWater
	ColorPoisonEmetic
	ColorPoisonLethal
	ColorSelected
	ColorElectricForeground
	ColorElectricBackground
	ColorBurnableForeground
	ColorBurnableBackground
	ColorWaterBackground
	ColorMarked
	ColorGray
	ColorWeaponObvious
	ColorItems
	ColorMapGreen
	ColorMapYellow
	ColorMapRed
)

func ColorFromCode(color ColorCode) common.Color {
	switch color {
	case ColorLOS:
		return common.NewHSVColor(0, 0, 0.80)
	case ColorItems:
		return common.NewHSVColor(0, 0, 0.95)
	case ColorCivilian:
		return common.NewHSVColorFromRGBBytes(255, 255, 255)
	case ColorWater:
		return common.NewRGBColorFromBytes(49, 95, 204)
	case ColorGuard:
		return common.NewHSVColorFromRGBBytes(51, 255, 93)
	case ColorEnforcer:
		return common.NewHSVColorFromRGBBytes(255, 203, 51)
	case ColorTarget:
		return common.NewHSVColorFromRGBBytes(255, 51, 51)
	case ColorActor:
		return common.NewHSVColor(170/360.0, 0.5, 0.5)
	case ColorDead:
		return common.NewHSVColorFromRGBBytes(255, 51, 51)
	case ColorGood:
		return common.NewHSVColorFromRGBBytes(51, 255, 93)
	case ColorWarning:
		return common.NewRGBColorFromBytes(255, 203, 51)
	case ColorBrightRed:
		return common.NewRGBColorFromBytes(255, 0, 0)
	case ColorTree:
		return common.NewRGBColorFromBytes(28, 109, 57)
	case ColorMapForeground:
		return common.NewHSVColor(0, 0, 0.65)
	case ColorPoisonLethal:
		return common.NewHSVColorFromRGBBytes(165, 10, 223)
	case ColorPoisonEmetic:
		return common.NewHSVColorFromRGBBytes(121, 186, 141)
	case ColorExplosionLight:
		return common.NewHSVColorFromRGBBytes(255, 85, 0)
	case ColorExplosionDark:
		return common.NewHSVColorFromRGBBytes(117, 20, 0)
	case ColorWeaponObvious:
		return common.NewHSVColor(10.0/360.0, 1.0, 1.0)
	case ColorMapBackground:
		return common.NewHSVColorFromRGBBytes(0, 4, 52)
	case ColorBlackBackground:
		return common.NewHSVColorFromRGBBytes(0, 0, 0)
	case ColorFoVSource:
		return common.NewHSVColorFromRGBBytes(52, 200, 52)
	case ColorMapBackgroundLight:
		return common.NewHSVColorFromRGBBytes(0, 8, 102)
	case ColorMapGreen:
		return common.NewHSVColor(137/360.0, 1, 0.7)
	case ColorMapYellow:
		return common.NewHSVColor(59/360.0, 1, 0.7)
	case ColorMapRed:
		return common.NewHSVColor(8/360.0, 1, 0.7)
	case ColorVisionCone:
		return common.NewHSVColorFromRGBBytes(80, 0, 102)
	case ColorRedIndicator:
		return common.RGBAColor{R: 191 / 255.0, G: 53 / 255.0, B: 29 / 255.0, A: 1.0}
	case ColorBlood:
		return common.NewHSVColorFromRGBBytes(255, 10, 10)
	case ColorMarked:
		return common.NewHSVColorFromRGBBytes(98, 182, 176)
	case ColorSelected:
		return common.NewHSVColorFromRGBBytes(191, 191, 191)
	case ColorGray:
		return common.NewHSVColorFromRGBBytes(191, 191, 191)
	case ColorBurnableBackground:
		return common.NewHSVColorFromRGBBytes(130, 108, 69)
	case ColorBurnableForeground:
		return common.NewHSVColorFromRGBBytes(63, 52, 33)
	case ColorWaterBackground:
		return common.NewHSVColorFromRGBBytes(23, 13, 87)
	case ColorElectricForeground:
		return common.NewHSVColor(57/360.0, 1, 0.55)
	case ColorElectricBackground:
		return common.NewHSVColor(57/360.0, 1, 0.15)
	}
	return common.NewHSVColorFromRGBBytes(255, 255, 255)
}
