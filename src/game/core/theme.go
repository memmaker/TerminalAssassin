package core

import (
    "io/fs"
    "os"
    "strings"

    "github.com/memmaker/terminal-assassin/common"
    rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

// ColorTheme defines every color used during gameplay.
// Swap CurrentTheme at runtime for an instant full re-skin.
//
// Naming convention: every field ends in Foreground or Background to indicate
// the only drawing layer it may be used for. The few exceptions are the
// Clothing palette (always foreground) and MenuHighlightForeground (foreground tint).
type ColorTheme struct {
    Name string

    // ── Map / World ──────────────────────────────────────────────────────
    MapBackground      common.Color // bg: default ground fill
    MapBackgroundLight common.Color // bg: lighter ground variant / named-location highlight
    MapForeground      common.Color // fg: default ground glyph ink

    WallForeground common.Color // fg: wall glyph ink
    WallBackground common.Color // bg: wall fill

    // ── Hazards / Stimuli ────────────────────────────────────────────────
    // Each hazard pair is used exclusively as (fg, bg) when drawing the stimulus.
    FireForeground        common.Color
    FireBackground        common.Color
    ExplosionForeground   common.Color
    ExplosionBackground   common.Color
    BloodForeground       common.Color // fg: blood glyph ink
    BloodBackground       common.Color // bg: blood tile fill
    WaterForeground       common.Color
    WaterBackground       common.Color
    OilForeground         common.Color
    OilBackground         common.Color
    ElectricityForeground common.Color
    ElectricityBackground common.Color

    // ── Poisons (foreground only) ────────────────────────────────────────
    // Poison colors tint the glyph foreground of poisoned items and stimuli.
    SleepPoisonForeground  common.Color
    EmeticPoisonForeground common.Color
    FrenzyPoisonForeground common.Color
    LethalPoisonForeground common.Color

    // ── Generic semantic colors (foreground only) ────────────────────────
    // DangerForeground is a generic "alert red" for foreground use in the HUD,
    // animations, and UI text whenever a red tint is needed that is NOT blood.
    DangerForeground common.Color
    // SuccessForeground / FailureForeground are used for result screens and
    // career text where green = good, red = bad.
    SuccessForeground common.Color
    FailureForeground common.Color
    // DeviceOnForeground / DeviceBrokenForeground color the glyphs of interactive
    // devices (distractors, triggers). Off uses the default style, On is active,
    // Broken means the device was destroyed (e.g. by damage).
    DeviceOnForeground     common.Color
    DeviceBrokenForeground common.Color

    // ── Interaction / Actions (background only) ──────────────────────────
    AimingBackground        common.Color
    LegalActionBackground   common.Color
    IllegalActionBackground common.Color
    ExitHighlightBackground common.Color

    // ── UI / HUD ─────────────────────────────────────────────────────────
    MenuBackground         common.Color
    MenuForeground         common.Color
    MenuHighlightForeground common.Color // fg: highlighted menu item text
    TooltipBackground      common.Color
    TooltipForeground      common.Color
    // HUD traffic-light colors are split into fg and bg variants so that each
    // field is used for exactly one drawing layer.
    HUDGoodForeground    common.Color // fg: healthy HP bar, success badge text
    HUDGoodBackground    common.Color // bg: success badge fill
    HUDWarningForeground common.Color // fg: medium HP bar text
    HUDWarningBackground common.Color // bg: trespassing zone, editor path
    HUDDangerBackground  common.Color // bg: hostile zone, failure badge fill

    // ── Vision / Awareness (background only) ─────────────────────────────
    LOSBackground        common.Color // bg: line-of-sight / hiding indicator
    VisionConeBackground common.Color // bg: NPC vision cone overlay

    SuspicionLowBackground    common.Color // bg: vision cone at suspicion 1
    SuspicionMediumBackground common.Color // bg: vision cone at suspicion 2
    SuspicionHighBackground   common.Color // bg: vision cone at suspicion 3
    MarkedBackground          common.Color // bg: editor selection highlight
    SelectionBackground       common.Color // bg: editor task-number cell

    // ── Editor Zone Highlight Colors (background only) ───────────────────
    ZoneDropOffBackground      common.Color
    ZoneHighSecurityBackground common.Color
    ZonePublicBackground       common.Color
    ZonePrivateBackground      common.Color

    // ── Editor Overlays (background only) ────────────────────────────────
    EditorSpawnBackground      common.Color // bg: player spawn marker
    EditorVisionConeBackground common.Color // bg: vision cone / task preview overlay
    EditorBuriedItemBackground common.Color // bg: buried item highlight
    EditorTaskNumberForeground common.Color // fg: task-index digit on schedule tiles

    // ── Animation (background only) ──────────────────────────────────────
    EngagedInTaskBackground common.Color // bg: NPC task-animation flash

    // ── Objects & Items (foreground only) ─────────────────────────────────
    // These are available for data-file tile/item definitions.
    ObjectForeground        common.Color
    ItemForeground          common.Color
    ObviousWeaponForeground common.Color
    TreeForeground          common.Color
    // Container state backgrounds — indicate the contents of a CorpseContainer.
    ContainerHidingBackground common.Color // bg: player is hiding inside
    ContainerBodyBackground   common.Color // bg: a body has been stashed

    // ── Clothing palette (foreground only) ───────────────────────────────
    ClothingColors map[ClothingColor]common.HSVColor
}

// CurrentTheme is the active color theme.
// Replace it at any time; the next rendered frame picks it up immediately.
var CurrentTheme *ColorTheme = DefaultTheme()

// SetTheme replaces the active theme.
func SetTheme(t *ColorTheme) {
    CurrentTheme = t
}

// DefaultTheme returns the original hardcoded color palette as a ColorTheme.
func DefaultTheme() *ColorTheme {
    return &ColorTheme{
        Name:               "default",
        MapBackground:      common.NewHSVColorFromRGBBytes(200, 200, 200),
        MapBackgroundLight: common.NewHSVColorFromRGBBytes(220, 220, 220),
        MapForeground:      common.NewHSVColor(0, 0, 0.65),
        WallForeground:     common.Black,
        WallBackground:     common.RGBAColor{R: 0.9, G: 0.9, B: 0.9, A: 1.0},

        FireForeground:        common.NewHSVColorFromRGBBytes(117, 20, 0),
        FireBackground:        common.NewHSVColorFromRGBBytes(255, 85, 0),
        ExplosionForeground:   common.NewHSVColorFromRGBBytes(117, 20, 0),
        ExplosionBackground:   common.NewHSVColorFromRGBBytes(255, 85, 0),
        BloodForeground:       common.NewRGBColorFromBytes(255, 0, 0),
        BloodBackground:       common.NewHSVColorFromRGBBytes(255, 10, 10),
        WaterForeground:       common.NewRGBColorFromBytes(49, 95, 204),
        WaterBackground:       common.NewHSVColorFromRGBBytes(23, 13, 87),
        OilForeground:         common.NewHSVColorFromRGBBytes(63, 52, 33),
        OilBackground:         common.NewHSVColorFromRGBBytes(130, 108, 69),
        ElectricityForeground: common.NewHSVColor(57.0/360.0, 1, 0.55),
        ElectricityBackground: common.NewHSVColor(57.0/360.0, 1, 0.15),

        SleepPoisonForeground:  common.NewHSVColorFromRGBBytes(30, 140, 120),
        EmeticPoisonForeground: common.NewHSVColorFromRGBBytes(50, 160, 80),
        FrenzyPoisonForeground: common.NewRGBColorFromBytes(255, 128, 0),
        LethalPoisonForeground: common.NewHSVColorFromRGBBytes(165, 10, 223),

        DangerForeground:    common.NewHSVColorFromRGBBytes(255, 10, 10),
        SuccessForeground:   common.RGBAColor{R: 0.0, G: 1.0, B: 0.0, A: 1.0},
        FailureForeground:   common.RGBAColor{R: 1.0, G: 0.0, B: 0.0, A: 1.0},
        DeviceOnForeground:     common.RGBAColor{R: 0.0, G: 1.0, B: 0.0, A: 1.0},
        DeviceBrokenForeground: common.RGBAColor{R: 1.0, G: 0.0, B: 0.0, A: 1.0},

        AimingBackground:        common.RGBAColor{R: 191.0 / 255.0, G: 53.0 / 255.0, B: 29.0 / 255.0, A: 1.0},
        LegalActionBackground:   common.RGBAColor{R: 51.0 / 255.0, G: 255.0 / 255.0, B: 93.0 / 255.0, A: 1.0},
        IllegalActionBackground: common.RGBAColor{R: 191.0 / 255.0, G: 53.0 / 255.0, B: 29.0 / 255.0, A: 1.0},
        ExitHighlightBackground: common.NewHSVColor(137.0/360.0, 1, 0.7),

        MenuBackground:          common.RGBAColor{R: 13.0 / 255.0, G: 8.0 / 255.0, B: 1.0 / 255.0, A: 1.0},
        MenuForeground:          common.RGBAColor{R: 1.0 * 3, G: 0.635294117647059 * 3, B: (1.0 / 255.0) * 3, A: 1.0},
        MenuHighlightForeground: common.NewHSVColorFromRGBBytes(191, 191, 191),
        TooltipBackground:       common.RGBAColor{R: 4.0, G: 4.0, B: 4.0, A: 1.0},
        TooltipForeground:       common.Black,
        HUDGoodForeground:       common.NewHSVColorFromRGBBytes(51, 255, 93),
        HUDGoodBackground:       common.NewHSVColorFromRGBBytes(51, 255, 93),
        HUDWarningForeground:    common.NewRGBColorFromBytes(255, 203, 51),
        HUDWarningBackground:    common.NewRGBColorFromBytes(255, 203, 51),
        HUDDangerBackground:     common.NewHSVColorFromRGBBytes(255, 10, 10),

        LOSBackground:        common.NewHSVColor(0, 0, 0.80),
        VisionConeBackground: common.NewHSVColorFromRGBBytes(80, 0, 102),
        SuspicionLowBackground:    common.NewHSVColor(137.0/360.0, 1, 0.7),
        SuspicionMediumBackground: common.NewHSVColor(59.0/360.0, 1, 0.7),
        SuspicionHighBackground:   common.NewHSVColor(8.0/360.0, 1, 0.7),
        MarkedBackground:          common.NewHSVColorFromRGBBytes(98, 182, 176),
        SelectionBackground:       common.NewHSVColorFromRGBBytes(191, 191, 191),

        ObjectForeground:        common.NewHSVColor(0, 0, 0.95),
        ItemForeground:          common.NewHSVColor(0, 0, 0.95),
        ObviousWeaponForeground: common.NewHSVColor(10.0/360.0, 1.0, 1.0),
        TreeForeground:          common.NewRGBColorFromBytes(28, 109, 57),
        ContainerHidingBackground: common.NewHSVColor(0, 0, 0.80),
        ContainerBodyBackground:   common.NewRGBColorFromBytes(255, 203, 51),

        ZoneDropOffBackground:      common.NewRGBColorFromBytes(49, 95, 204),
        ZoneHighSecurityBackground: common.NewRGBColorFromBytes(255, 0, 0),
        ZonePublicBackground:       common.NewHSVColor(137.0/360.0, 1, 0.7),
        ZonePrivateBackground:      common.NewHSVColor(300.0/360.0, 0.8, 0.8),

        EditorSpawnBackground:      common.RGBAColor{R: 0.0, G: 1.0, B: 0.0, A: 1.0},
        EditorVisionConeBackground: common.RGBAColor{R: 0.0, G: 1.0, B: 0.0, A: 1.0},
        EditorBuriedItemBackground: common.NewRGBColorFromBytes(101, 67, 33),
        EditorTaskNumberForeground: common.NewHSVColorFromRGBBytes(255, 10, 10),

        EngagedInTaskBackground: common.NewHSVColorFromRGBBytes(255, 203, 51),

        ClothingColors: map[ClothingColor]common.HSVColor{
            ClothingColorBlack:   common.NewHSVColorFromRGBBytes(0, 0, 0),
            ClothingColorRed:     common.NewHSVColor(0.0/360.0, 0.8, 1.0),
            ClothingColorOrange:  common.NewHSVColor(30.0/360.0, 0.8, 1.0),
            ClothingColorYellow:  common.NewHSVColor(60.0/360.0, 0.8, 1.0),
            ClothingColorGreen:   common.NewHSVColor(120.0/360.0, 0.8, 1.0),
            ClothingColorCyan:    common.NewHSVColor(180.0/360.0, 0.8, 1.0),
            ClothingColorBlue:    common.NewHSVColor(240.0/360.0, 0.8, 1.0),
            ClothingColorViolet:  common.NewHSVColor(270.0/360.0, 0.8, 1.0),
            ClothingColorMagenta: common.NewHSVColor(300.0/360.0, 0.8, 1.0),
        },
    }
}

// clothingPrefix is the rec-file field prefix for clothing colors.
const clothingPrefix = "Clothing_"

// ToRecord serializes the theme to a single rec_files record.
func (t *ColorTheme) ToRecord() rec_files.Record {
    fields := rec_files.Record{
        {Name: "Name", Value: t.Name},
        {Name: "MapBackground", Value: t.MapBackground.EncodeAsString()},
        {Name: "MapBackgroundLight", Value: t.MapBackgroundLight.EncodeAsString()},
        {Name: "MapForeground", Value: t.MapForeground.EncodeAsString()},
        {Name: "WallForeground", Value: t.WallForeground.EncodeAsString()},
        {Name: "WallBackground", Value: t.WallBackground.EncodeAsString()},
        {Name: "FireForeground", Value: t.FireForeground.EncodeAsString()},
        {Name: "FireBackground", Value: t.FireBackground.EncodeAsString()},
        {Name: "ExplosionForeground", Value: t.ExplosionForeground.EncodeAsString()},
        {Name: "ExplosionBackground", Value: t.ExplosionBackground.EncodeAsString()},
        {Name: "BloodForeground", Value: t.BloodForeground.EncodeAsString()},
        {Name: "BloodBackground", Value: t.BloodBackground.EncodeAsString()},
        {Name: "WaterForeground", Value: t.WaterForeground.EncodeAsString()},
        {Name: "WaterBackground", Value: t.WaterBackground.EncodeAsString()},
        {Name: "OilForeground", Value: t.OilForeground.EncodeAsString()},
        {Name: "OilBackground", Value: t.OilBackground.EncodeAsString()},
        {Name: "ElectricityForeground", Value: t.ElectricityForeground.EncodeAsString()},
        {Name: "ElectricityBackground", Value: t.ElectricityBackground.EncodeAsString()},
        {Name: "SleepPoisonForeground", Value: t.SleepPoisonForeground.EncodeAsString()},
        {Name: "EmeticPoisonForeground", Value: t.EmeticPoisonForeground.EncodeAsString()},
        {Name: "FrenzyPoisonForeground", Value: t.FrenzyPoisonForeground.EncodeAsString()},
        {Name: "LethalPoisonForeground", Value: t.LethalPoisonForeground.EncodeAsString()},
        {Name: "DangerForeground", Value: t.DangerForeground.EncodeAsString()},
        {Name: "SuccessForeground", Value: t.SuccessForeground.EncodeAsString()},
        {Name: "FailureForeground", Value: t.FailureForeground.EncodeAsString()},
        {Name: "DeviceOnForeground", Value: t.DeviceOnForeground.EncodeAsString()},
        {Name: "DeviceBrokenForeground", Value: t.DeviceBrokenForeground.EncodeAsString()},
        {Name: "AimingBackground", Value: t.AimingBackground.EncodeAsString()},
        {Name: "LegalActionBackground", Value: t.LegalActionBackground.EncodeAsString()},
        {Name: "IllegalActionBackground", Value: t.IllegalActionBackground.EncodeAsString()},
        {Name: "ExitHighlightBackground", Value: t.ExitHighlightBackground.EncodeAsString()},
        {Name: "MenuBackground", Value: t.MenuBackground.EncodeAsString()},
        {Name: "MenuForeground", Value: t.MenuForeground.EncodeAsString()},
        {Name: "MenuHighlightForeground", Value: t.MenuHighlightForeground.EncodeAsString()},
        {Name: "TooltipBackground", Value: t.TooltipBackground.EncodeAsString()},
        {Name: "TooltipForeground", Value: t.TooltipForeground.EncodeAsString()},
        {Name: "HUDGoodForeground", Value: t.HUDGoodForeground.EncodeAsString()},
        {Name: "HUDGoodBackground", Value: t.HUDGoodBackground.EncodeAsString()},
        {Name: "HUDWarningForeground", Value: t.HUDWarningForeground.EncodeAsString()},
        {Name: "HUDWarningBackground", Value: t.HUDWarningBackground.EncodeAsString()},
        {Name: "HUDDangerBackground", Value: t.HUDDangerBackground.EncodeAsString()},
        {Name: "LOSBackground", Value: t.LOSBackground.EncodeAsString()},
        {Name: "VisionConeBackground", Value: t.VisionConeBackground.EncodeAsString()},
        {Name: "SuspicionLowBackground", Value: t.SuspicionLowBackground.EncodeAsString()},
        {Name: "SuspicionMediumBackground", Value: t.SuspicionMediumBackground.EncodeAsString()},
        {Name: "SuspicionHighBackground", Value: t.SuspicionHighBackground.EncodeAsString()},
        {Name: "MarkedBackground", Value: t.MarkedBackground.EncodeAsString()},
        {Name: "SelectionBackground", Value: t.SelectionBackground.EncodeAsString()},
        {Name: "ObjectForeground", Value: t.ObjectForeground.EncodeAsString()},
        {Name: "ItemForeground", Value: t.ItemForeground.EncodeAsString()},
        {Name: "ObviousWeaponForeground", Value: t.ObviousWeaponForeground.EncodeAsString()},
        {Name: "TreeForeground", Value: t.TreeForeground.EncodeAsString()},
        {Name: "ContainerHidingBackground", Value: t.ContainerHidingBackground.EncodeAsString()},
        {Name: "ContainerBodyBackground", Value: t.ContainerBodyBackground.EncodeAsString()},
        {Name: "ZoneDropOffBackground", Value: t.ZoneDropOffBackground.EncodeAsString()},
        {Name: "ZoneHighSecurityBackground", Value: t.ZoneHighSecurityBackground.EncodeAsString()},
        {Name: "ZonePublicBackground", Value: t.ZonePublicBackground.EncodeAsString()},
        {Name: "ZonePrivateBackground", Value: t.ZonePrivateBackground.EncodeAsString()},
        {Name: "EditorSpawnBackground", Value: t.EditorSpawnBackground.EncodeAsString()},
        {Name: "EditorVisionConeBackground", Value: t.EditorVisionConeBackground.EncodeAsString()},
        {Name: "EditorBuriedItemBackground", Value: t.EditorBuriedItemBackground.EncodeAsString()},
        {Name: "EditorTaskNumberForeground", Value: t.EditorTaskNumberForeground.EncodeAsString()},
        {Name: "EngagedInTaskBackground", Value: t.EngagedInTaskBackground.EncodeAsString()},
    }
    for colorName, hsv := range t.ClothingColors {
        fields = append(fields, rec_files.Field{
            Name:  clothingPrefix + string(colorName),
            Value: hsv.EncodeAsString(),
        })
    }
    return fields
}

// ThemeFromRecord deserializes a ColorTheme from a rec_files record.
// Missing fields fall back to DefaultTheme values.
func ThemeFromRecord(record rec_files.Record) *ColorTheme {
    t := DefaultTheme()
    for _, field := range record {
        switch field.Name {
        case "Name":
            t.Name = field.Value
        case "MapBackground":
            t.MapBackground = common.NewColorFromString(field.Value)
        case "MapBackgroundLight":
            t.MapBackgroundLight = common.NewColorFromString(field.Value)
        case "MapForeground":
            t.MapForeground = common.NewColorFromString(field.Value)
        case "WallForeground":
            t.WallForeground = common.NewColorFromString(field.Value)
        case "WallBackground":
            t.WallBackground = common.NewColorFromString(field.Value)
        case "FireForeground":
            t.FireForeground = common.NewColorFromString(field.Value)
        case "FireBackground":
            t.FireBackground = common.NewColorFromString(field.Value)
        case "ExplosionForeground":
            t.ExplosionForeground = common.NewColorFromString(field.Value)
        case "ExplosionBackground":
            t.ExplosionBackground = common.NewColorFromString(field.Value)
        case "BloodForeground":
            t.BloodForeground = common.NewColorFromString(field.Value)
        case "BloodBackground":
            t.BloodBackground = common.NewColorFromString(field.Value)
        case "WaterForeground":
            t.WaterForeground = common.NewColorFromString(field.Value)
        case "WaterBackground":
            t.WaterBackground = common.NewColorFromString(field.Value)
        case "OilForeground":
            t.OilForeground = common.NewColorFromString(field.Value)
        case "OilBackground":
            t.OilBackground = common.NewColorFromString(field.Value)
        case "ElectricityForeground":
            t.ElectricityForeground = common.NewColorFromString(field.Value)
        case "ElectricityBackground":
            t.ElectricityBackground = common.NewColorFromString(field.Value)
        // Accept both old and new poison field names for backwards compat.
        case "SleepPoisonForeground", "SleepPoisonColor":
            t.SleepPoisonForeground = common.NewColorFromString(field.Value)
        case "EmeticPoisonForeground", "EmeticPoisonColor":
            t.EmeticPoisonForeground = common.NewColorFromString(field.Value)
        case "FrenzyPoisonForeground", "FrenzyPoisonColor":
            t.FrenzyPoisonForeground = common.NewColorFromString(field.Value)
        case "LethalPoisonForeground", "LethalPoisonColor":
            t.LethalPoisonForeground = common.NewColorFromString(field.Value)
        case "DangerForeground":
            t.DangerForeground = common.NewColorFromString(field.Value)
        case "SuccessForeground":
            t.SuccessForeground = common.NewColorFromString(field.Value)
        case "FailureForeground":
            t.FailureForeground = common.NewColorFromString(field.Value)
        case "DeviceOnForeground":
            t.DeviceOnForeground = common.NewColorFromString(field.Value)
        case "DeviceBrokenForeground", "DeviceOffForeground":
            t.DeviceBrokenForeground = common.NewColorFromString(field.Value)
        case "AimingBackground":
            t.AimingBackground = common.NewColorFromString(field.Value)
        case "LegalActionBackground":
            t.LegalActionBackground = common.NewColorFromString(field.Value)
        case "IllegalActionBackground":
            t.IllegalActionBackground = common.NewColorFromString(field.Value)
        case "ExitHighlightBackground", "ExitHighlightColor":
            t.ExitHighlightBackground = common.NewColorFromString(field.Value)
        case "MenuBackground":
            t.MenuBackground = common.NewColorFromString(field.Value)
        case "MenuForeground":
            t.MenuForeground = common.NewColorFromString(field.Value)
        case "MenuHighlightForeground", "MenuHighlight":
            t.MenuHighlightForeground = common.NewColorFromString(field.Value)
        case "TooltipBackground":
            t.TooltipBackground = common.NewColorFromString(field.Value)
        case "TooltipForeground":
            t.TooltipForeground = common.NewColorFromString(field.Value)
        // Accept both old and new HUD field names for backwards compat.
        case "HUDGoodForeground":
            t.HUDGoodForeground = common.NewColorFromString(field.Value)
        case "HUDGoodBackground":
            t.HUDGoodBackground = common.NewColorFromString(field.Value)
        case "HUDGoodColor":
            c := common.NewColorFromString(field.Value)
            t.HUDGoodForeground = c
            t.HUDGoodBackground = c
        case "HUDWarningForeground":
            t.HUDWarningForeground = common.NewColorFromString(field.Value)
        case "HUDWarningBackground":
            t.HUDWarningBackground = common.NewColorFromString(field.Value)
        case "HUDWarningColor":
            c := common.NewColorFromString(field.Value)
            t.HUDWarningForeground = c
            t.HUDWarningBackground = c
        case "HUDDangerBackground", "HUDDangerColor":
            t.HUDDangerBackground = common.NewColorFromString(field.Value)
        case "LOSBackground", "LOSColor":
            t.LOSBackground = common.NewColorFromString(field.Value)
        case "VisionConeBackground", "VisionConeColor":
            t.VisionConeBackground = common.NewColorFromString(field.Value)
        case "SuspicionLowBackground", "SuspicionLow":
            t.SuspicionLowBackground = common.NewColorFromString(field.Value)
        case "SuspicionMediumBackground", "SuspicionMedium":
            t.SuspicionMediumBackground = common.NewColorFromString(field.Value)
        case "SuspicionHighBackground", "SuspicionHigh":
            t.SuspicionHighBackground = common.NewColorFromString(field.Value)
        case "MarkedBackground", "MarkedColor":
            t.MarkedBackground = common.NewColorFromString(field.Value)
        case "SelectionBackground", "SelectionColor":
            t.SelectionBackground = common.NewColorFromString(field.Value)
        case "ObjectForeground":
            t.ObjectForeground = common.NewColorFromString(field.Value)
        case "ItemForeground":
            t.ItemForeground = common.NewColorFromString(field.Value)
        case "ObviousWeaponForeground", "ObviousWeaponColor":
            t.ObviousWeaponForeground = common.NewColorFromString(field.Value)
        case "TreeForeground", "TreeColor":
            t.TreeForeground = common.NewColorFromString(field.Value)
        case "ContainerHidingBackground":
            t.ContainerHidingBackground = common.NewColorFromString(field.Value)
        case "ContainerBodyBackground":
            t.ContainerBodyBackground = common.NewColorFromString(field.Value)
        case "ZoneDropOffBackground", "ZoneDropOffColor":
            t.ZoneDropOffBackground = common.NewColorFromString(field.Value)
        case "ZoneHighSecurityBackground", "ZoneHighSecurityColor":
            t.ZoneHighSecurityBackground = common.NewColorFromString(field.Value)
        case "ZonePublicBackground", "ZonePublicColor":
            t.ZonePublicBackground = common.NewColorFromString(field.Value)
        case "ZonePrivateBackground", "ZonePrivateColor":
            t.ZonePrivateBackground = common.NewColorFromString(field.Value)
        case "EditorSpawnBackground":
            t.EditorSpawnBackground = common.NewColorFromString(field.Value)
        case "EditorVisionConeBackground":
            t.EditorVisionConeBackground = common.NewColorFromString(field.Value)
        case "EditorBuriedItemBackground":
            t.EditorBuriedItemBackground = common.NewColorFromString(field.Value)
        case "EditorTaskNumberForeground":
            t.EditorTaskNumberForeground = common.NewColorFromString(field.Value)
        case "EngagedInTaskBackground", "EngagedInTaskColor":
            t.EngagedInTaskBackground = common.NewColorFromString(field.Value)
        default:
            if strings.HasPrefix(field.Name, clothingPrefix) {
                colorName := ClothingColor(strings.TrimPrefix(field.Name, clothingPrefix))
                t.ClothingColors[colorName] = common.NewColorFromString(field.Value).ToHSV()
            }
        }
    }
    return t
}

// LoadThemeFromFile reads a ColorTheme from a rec-format file.
// Falls back to DefaultTheme if the file is empty.
func LoadThemeFromFile(file fs.File) (*ColorTheme, error) {
    records := rec_files.Read(file)
    if len(records) == 0 {
        return DefaultTheme(), nil
    }
    return ThemeFromRecord(records[0]), nil
}

// SaveThemeToFile writes a ColorTheme to a file in rec format.
func SaveThemeToFile(file *os.File, t *ColorTheme) error {
    return rec_files.Write(file, []rec_files.Record{t.ToRecord()})
}
