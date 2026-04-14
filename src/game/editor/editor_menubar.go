package editor

import (
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/ui"
)

func (g *GameStateEditor) createMenuBar(gridWidth int, gridHeight int) {
    userInterface := g.engine.GetUI()
    editorMainMenuItems := []services.MenuItem{
        {
            Label:    "Brush",
            Handler:  g.openBrushMenu,
            Icon:     core.GlyphPencil,
            QuickKey: core.KeyTab,
        },
        {
            Label:     "Tiles",
            Handler:   g.openTileMenu,
            Icon:      core.GlyphWall,
            Highlight: g.isState(editMapUI),
            QuickKey:  "F1",
        },
        {
            Label:     "Items",
            Handler:   g.openItemMenu,
            Icon:      core.GlyphWrench,
            Highlight: g.isState(addItemsUI),
            QuickKey:  "F2",
        },
        {
            Label:     "Objects",
            Handler:   g.openObjectsMenu,
            Icon:      core.GlyphClosedDoor,
            Highlight: g.isState(addObjectsUI),
            QuickKey:  "F3",
        },
        {
            Label:     "Actors",
            Handler:   g.changeUIStateFunc(editActorUI),
            Icon:      '@',
            Highlight: g.isState(editActorUI),
            QuickKey:  "F4",
        },
        {
            Label:     "Schedule",
            Handler:   g.openScheduleLibraryMenu,
            Icon:      'S',
            Highlight: g.isState(editScheduleUI),
            QuickKey:  "F5",
        },
        {
            Label:     "Clothes",
            Handler:   g.openClothesMenu,
            Icon:      core.GlyphClothing,
            Highlight: g.isState(addClothesUI),
            QuickKey:  "F6",
        },
        {
            Label:     "Zones",
            Handler:   g.openZoneMenu,
            Icon:      'ç',
            Highlight: g.isState(addZonesUI),
            QuickKey:  "F7",
        },
        {
            Label:     "Edit Named Locations",
            Handler:   g.setBrushHandler(editNamedLocationUI, 'ç', g.selectNamedLocation),
            Icon:      'n',
            Highlight: g.isState(editNamedLocationUI),
        },
        {
            Label:     "Stimuli",
            Handler:   g.openStimuliMenu,
            Icon:      core.GlyphWater,
            Highlight: g.isState(addStimuliUI),
            QuickKey:  "F8",
        },
        {
            Label: "Lights",
            Handler: g.setBrushHandlerWithLightUpdate(editLightsUI, core.GlyphStreetLight, func(point geometry.Point) {
                g.placeLightAt(point)
            }),
            Icon:      core.GlyphStreetLight,
            Highlight: g.isState(editLightsUI),
            QuickKey:  "F9",
        },
        {
            Label:    "Color",
            Handler:  g.openColorMenu,
            Icon:     core.GlyphPalette,
            QuickKey: "F12",
        },
        {
            Label: "Eye dropper (tiles)",
            Handler: func() {
                g.placeThingIcon = 'e'
                g.handler.CellsSelected = g.pickTileFromSelection
                g.updateStatusLine()
            },
            Icon:     'e',
            QuickKey: "e",
        },
        {
            Label: "Prefabs",
            Handler: func() {
                g.changeUIStateTo(createPrefabUI)
                g.placeThingIcon = 'p'
                g.selectionTool = NewFilledRectangleBrush()
                g.handler.CellsSelected = g.prefabFromSelection
            },
            Icon:      'p',
            Highlight: g.isState(createPrefabUI),
            QuickKey:  "F10",
        },
        {
            Label:    "Global",
            Handler:  g.openGlobalMenu,
            Icon:     'g',
            QuickKey: "F11",
        },
    }
    menuBarRect := geometry.NewRect(0, gridHeight-3, gridWidth, gridHeight-1)
    g.menuBar = ui.NewMenuBar("Editor", editorMainMenuItems, menuBarRect)
    userInterface.AddToScene(g.menuBar)
}
