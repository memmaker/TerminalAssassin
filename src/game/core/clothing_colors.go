package core

import "github.com/memmaker/terminal-assassin/common"

// ClothingColor is one of the named entries in ClothingPalette.
// Clothing may only use these colors.
type ClothingColor string

const (
    ClothingColorBlack   ClothingColor = "black"
    ClothingColorRed     ClothingColor = "red"
    ClothingColorOrange  ClothingColor = "orange"
    ClothingColorYellow  ClothingColor = "yellow"
    ClothingColorGreen   ClothingColor = "green"
    ClothingColorCyan    ClothingColor = "cyan"
    ClothingColorBlue    ClothingColor = "blue"
    ClothingColorViolet  ClothingColor = "violet"
    ClothingColorMagenta ClothingColor = "magenta"
)

// ClothingPalette is the single source of truth for all clothing colors.
var ClothingPalette = map[ClothingColor]common.HSVColor{
    ClothingColorBlack:   common.NewHSVColorFromRGBBytes(0, 0, 0),
    ClothingColorRed:     common.NewHSVColor(0.0/360.0, 0.8, 1.0),
    ClothingColorOrange:  common.NewHSVColor(30.0/360.0, 0.8, 1.0),
    ClothingColorYellow:  common.NewHSVColor(60.0/360.0, 0.8, 1.0),
    ClothingColorGreen:   common.NewHSVColor(120.0/360.0, 0.8, 1.0),
    ClothingColorCyan:    common.NewHSVColor(180.0/360.0, 0.8, 1.0),
    ClothingColorBlue:    common.NewHSVColor(240.0/360.0, 0.8, 1.0),
    ClothingColorViolet:  common.NewHSVColor(270.0/360.0, 0.8, 1.0),
    ClothingColorMagenta: common.NewHSVColor(300.0/360.0, 0.8, 1.0),
}
