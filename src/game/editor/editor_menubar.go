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
			Label: "Schedule",
			Handler: func() {
				if g.SelectedActor == nil {
					g.PrintAsMessage("ERR: select an Actor first")
					return
				}
				g.changeUIStateTo(editTaskUI)
			},
			Icon:      'S',
			Highlight: g.isState(editTaskUI),
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
		{
			Label: "Tile Background Color",
			Handler: func() {
				g.changeBackgroundColor()
			},
			Icon: core.GlyphPalette,
		},
		{
			Label: "Tile Foreground Color",
			Handler: func() {
				g.changeForegroundColor()
			},
			Icon: core.GlyphPalette,
		},
		{
			Label: "Eye dropper (tiles/color)",
			Handler: func() {
				g.placeThingIcon = 'e'
				g.handler.CellsSelected = g.pickTileFromSelection
				g.updateStatusLine()
			},
			ShiftHandler: func() {
				g.placeThingIcon = 'e'
				g.handler.CellsSelected = g.pickColorFromSelection
				g.updateStatusLine()
			},
			Icon:     'e',
			QuickKey: "e",
		},
		{
			Label:     "Edit Named Locations",
			Handler:   g.setBrushHandler(editNamedLocationUI, 'ç', g.selectNamedLocation),
			Icon:      'n',
			Highlight: g.isState(editNamedLocationUI),
		},
		{
			Label:   "Ambient Light",
			Handler: g.changeAmbientLight,
			Icon:    'L',
		},
	}
	menuBarRect := geometry.NewRect(0, gridHeight-3, gridWidth, gridHeight-1)
	g.menuBar = ui.NewMenuBar("Editor", editorMainMenuItems, menuBarRect)
	userInterface.AddToScene(g.menuBar)
}
