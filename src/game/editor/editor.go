package editor

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/ui"
)

type GameStateEditor struct {
	engine                 services.Engine
	gridIsDirty            bool
	clearHalfWidth         bool
	selectionTool          Brush
	selectedWorldPositions []geometry.Point
	menuBar                *ui.MenuBar
	placeThingIcon         rune
	currentBackgroundColor common.Color
	currentForegroundColor common.Color
	clearHUD               bool

	CurrentRune rune
	MouseDown   bool
	textInput   *ui.TextInput

	LastSelectedPos       geometry.Point
	SelectedActor         *core.Actor
	SelectedLightSource   *gridmap.LightSource
	SelectedZone          *gridmap.ZoneInfo
	selectedItem          *core.Item
	selectedObject        services.Object
	selectedNamedLocation string
	SelectedTaskIndex     int
	handler               UIHandler
	topStatusLineLabel    *ui.FixedLabel
	bottomMessageLabel    *ui.FixedLabel
	boxStart              geometry.Point
	boxDrag               bool
	boxEnd                geometry.Point
	MousePositionOnScreen geometry.Point
	MousePositionInWorld  geometry.Point
	currentPrefab         *gridmap.Prefab[*core.Actor, *core.Item, services.Object]
}

func (g *GameStateEditor) ClearOverlay() {
	g.clearHalfWidth = true
}

type UIHandler struct {
	KeyPressed    map[core.Key]func()
	ContextMenu   []services.MenuItem
	CellsSelected func()
	MouseMoved    func()
	TextReceived  func(content string)
	Name          string
}

func (h UIHandler) WithTextHandler(textReceivedHandler func(text string)) UIHandler {
	h.TextReceived = textReceivedHandler
	return h
}

var placePrefabUI, setColorUI, createPrefabUI, editLightsUI, editNamedLocationUI, addObjectsUI, quickAddActorsUI, editMapUI, addStimuliUI, addZonesUI, addTasksUI, addActorsUI, editActorUI, editTaskUI, addClothesUI, addItemsUI UIHandler

var globalKeyPresses map[core.Key]func()

func (g *GameStateEditor) ResizeAndClearMap(newWidth, newHeight int) {
	game := g.engine.GetGame()
	game.ClearMap(newWidth, newHeight)
	currentMap := game.GetMap()
	currentMap.Apply(func(cell gridmap.MapCell[*core.Actor, *core.Item, services.Object]) gridmap.MapCell[*core.Actor, *core.Item, services.Object] {
		cell.IsExplored = true
		return cell
	})
	currentMap.ApplyAmbientLight()
	currentMap.UpdateBakedLights()
	currentMap.UpdateDynamicLights()
	game.GetCamera().CenterOn(geometry.Point{}, g.engine.MapWindowWidth(), g.engine.MapWindowHeight())
	g.gridIsDirty = true
	g.clearHalfWidth = true
}
func (g *GameStateEditor) Init(engine services.Engine) {
	g.engine = engine
	g.currentBackgroundColor = common.Black
	g.currentForegroundColor = common.White
	//game := g.engine.GetGame()
	//game.ClearMap(g.engine.MapWindowWidth(), g.engine.MapWindowWidth())
	//currentMap := game.GetMap()
	editMapUI = UIHandler{
		Name: "edit map",
		KeyPressed: map[core.Key]func(){
			core.KeySpace: g.openTileMenu,
		},
		CellsSelected: g.selectAtMousePos,
	}
	addZonesUI = UIHandler{
		Name: "add zones",
		KeyPressed: map[core.Key]func(){
			core.KeySpace: g.openZoneMenu,
		},
		ContextMenu: []services.MenuItem{
			{
				Label:    "Add Zone",
				Handler:  g.addZone,
				Icon:     'a',
				QuickKey: "a",
			},
			{
				Label:    "Toggle Type",
				Handler:  g.toggleZoneType,
				Icon:     'p',
				QuickKey: "p",
			},
			{
				Label:    "Open Clothes Menu",
				Handler:  g.openClothesMenuForZone,
				Icon:     'c',
				QuickKey: "c",
			},
		},
		CellsSelected: g.selectAtMousePos,
	}
	addStimuliUI = UIHandler{
		Name: "add stimuli",
		KeyPressed: map[core.Key]func(){
			core.KeySpace: g.openStimuliMenu,
		},
		CellsSelected: g.selectAtMousePos,
	}
	createPrefabUI = UIHandler{
		Name:          "create prefab",
		CellsSelected: g.selectAtMousePos,
	}
	placePrefabUI = UIHandler{
		Name:          "place prefab",
		CellsSelected: g.placePrefab,
		ContextMenu: []services.MenuItem{
			{
				Label: "Rotate Prefab",
				Handler: func() {
					if g.currentPrefab != nil {
						g.currentPrefab.RotateCW()
						g.gridIsDirty = true
					}
				},
				Icon:     'e',
				QuickKey: "e",
			},
		},
	}
	setColorUI = UIHandler{
		Name: "set color",
		KeyPressed: map[core.Key]func(){
			"o": g.changeBackgroundColor,
		},
		CellsSelected: g.selectAtMousePos,
	}
	editLightsUI = UIHandler{
		Name: "edit lights",
		ContextMenu: []services.MenuItem{
			{
				Label:    "Decrease Light Radius",
				Handler:  g.decreaseLightRadius,
				Icon:     's',
				QuickKey: "s",
			},
			{
				Label:    "Increase Light Radius",
				Handler:  g.increaseLightRadius,
				Icon:     'd',
				QuickKey: "d",
			},
			{
				Label:    "Change Light Color",
				Handler:  g.changeSelectedLightColor,
				Icon:     'f',
				QuickKey: "f",
			},
			{
				Label:    "Change Ambient Light",
				Handler:  g.changeAmbientLight,
				Icon:     'o',
				QuickKey: "o",
			},
			{
				Label:    "Update All Lights",
				Handler:  g.updateAllLights,
				Icon:     'i',
				QuickKey: "i",
			},
		},
		CellsSelected: g.selectAtMousePos,
	}
	addTasksUI = UIHandler{
		Name:          "add tasks",
		KeyPressed:    map[core.Key]func(){},
		CellsSelected: g.addTask,
	}
	editTaskUI = UIHandler{
		Name: "edit task",
		ContextMenu: []services.MenuItem{
			{
				Label:    "Add Task",
				Handler:  g.changeUIStateFunc(addTasksUI),
				Icon:     'a',
				QuickKey: "a",
			},
			{
				Label:    "Decrease Task Time",
				Handler:  g.decreaseTaskTime,
				Icon:     '-',
				QuickKey: "j",
			},
			{
				Label:    "Increase Task Time",
				Handler:  g.increaseTaskTime,
				Icon:     '+',
				QuickKey: "k",
			},
			{
				Label:    "Delete Task",
				Handler:  g.deleteTask,
				Icon:     'x',
				QuickKey: core.KeyBackspace,
			},
		},
		CellsSelected: g.selectAtMousePos,
	}
	addActorsUI = UIHandler{
		Name:          "add actors",
		KeyPressed:    map[core.Key]func(){},
		CellsSelected: g.addActor,
	}
	addClothesUI = UIHandler{
		Name: "add clothes",
		KeyPressed: map[core.Key]func(){
			core.KeySpace: g.openClothesMenu,
		},
		CellsSelected: g.selectAtMousePos,
	}
	quickAddActorsUI = UIHandler{
		Name:          "quick add actors",
		KeyPressed:    map[core.Key]func(){},
		CellsSelected: g.quickAddActor,
	}
	editActorUI = UIHandler{
		Name: "edit actor",
		ContextMenu: []services.MenuItem{
			{
				Label:    "Add Actor",
				Handler:  g.changeUIStateFunc(addActorsUI),
				Icon:     'a',
				QuickKey: "a",
			},
			{
				Label:    "Quick Add Actor",
				Handler:  g.changeUIStateFunc(quickAddActorsUI),
				Icon:     'q',
				QuickKey: "q",
			},
			{
				Label:    "Toggle Actor Type",
				Handler:  g.toggleActorType,
				Icon:     't',
				QuickKey: "t",
			},
			{
				Label:    "Select Leader",
				Handler:  g.selectLeaderForActor,
				Icon:     'f',
				QuickKey: "f",
			},
			{
				Label:    "Rename Actor",
				Handler:  g.renameSelectedActor,
				Icon:     'r',
				QuickKey: "r",
			},
			{
				Label:    "Move Actor",
				Handler:  g.moveSelectedActor,
				Icon:     'w',
				QuickKey: "w",
			},
			{
				Label:    "Open inventory",
				Handler:  g.openInventoryForSelectedActor,
				Icon:     'i',
				QuickKey: "i",
			},
			{
				Label:    "Adjust Look Direction",
				Handler:  g.adjustLookDirectionForSelectedActor,
				Icon:     'd',
				QuickKey: "d",
			},
			{
				Label:    "Delete Actor",
				Handler:  g.deleteActor,
				Icon:     'x',
				QuickKey: core.KeyBackspace,
			},
		},
		CellsSelected: g.selectAtMousePos,
	}

	addItemsUI = UIHandler{
		Name: "add / remove items",
		KeyPressed: map[core.Key]func(){
			core.KeySpace: g.openItemMenu,
		},
		ContextMenu: []services.MenuItem{
			{
				Label:    "Set Key",
				Handler:  g.setKeyOfSelectedItem,
				Icon:     'k',
				QuickKey: "k",
			},
			{
				Label:    "Delete Item",
				Handler:  g.removeSelectedItem,
				Icon:     'x',
				QuickKey: core.KeyBackspace,
			},
		},
		CellsSelected: g.selectAtMousePos,
	}

	addObjectsUI = UIHandler{
		Name: "add / remove objects",
		KeyPressed: map[core.Key]func(){
			core.KeySpace: g.openObjectsMenu,
		},
		ContextMenu: []services.MenuItem{
			{
				Label:    "Set Key",
				Handler:  g.setKeyOfSelectedObject,
				Icon:     'k',
				QuickKey: "k",
			},
			{
				Label:    "Delete Object",
				Handler:  g.removeSelectedObject,
				Icon:     'x',
				QuickKey: core.KeyBackspace,
			},
		},
		CellsSelected: g.selectAtMousePos,
	}
	editNamedLocationUI = UIHandler{
		Name:          "edit named locations",
		CellsSelected: g.selectAtMousePos,
		ContextMenu: []services.MenuItem{
			{
				Label:    "Add Location",
				Handler:  g.setBrushHandler(editNamedLocationUI, 'ç', g.newNamedLocation),
				Icon:     'q',
				QuickKey: "q",
			},
			{
				Label:    "Rename Location",
				Handler:  g.renameSelectedNamedLocation,
				Icon:     'r',
				QuickKey: "r",
			},
			{
				Label:    "Move Location",
				Handler:  g.moveSelectedNamedLocation,
				Icon:     'w',
				QuickKey: "w",
			},
			{
				Label:    "Delete Location",
				Handler:  g.deleteSelectedNamedLocation,
				Icon:     'x',
				QuickKey: core.KeyBackspace,
			},
		},
	}

	emptyTile := g.engine.GetData().GroundTile()
	globalKeyPresses = map[core.Key]func(){
		"n": func() {
			//g.engine.GetGame().GetMap().ClearKeyCards()
			g.backInTime()
			return
		},
		"m": func() {
			//g.engine.GetGame().GetMap().ClearKeys()
			g.forwardInTime()
			return
		},
		core.KeyLeftArrow: func() {
			if ebiten.IsKeyPressed(ebiten.KeyAlt) {
				g.engine.GetGame().GetMap().ShiftMapBy(geometry.Point{X: -1, Y: 0})
			} else if ebiten.IsKeyPressed(ebiten.KeyShift) {
				g.engine.GetGame().GetMap().Resize(g.engine.GetGame().GetMap().MapWidth-1, g.engine.GetGame().GetMap().MapHeight, emptyTile)
			} else {
				g.moveCameraOnMap(geometry.Point{X: -1, Y: 0})
			}
			g.gridIsDirty = true
		},
		core.KeyRightArrow: func() {
			if ebiten.IsKeyPressed(ebiten.KeyAlt) {
				g.engine.GetGame().GetMap().ShiftMapBy(geometry.Point{X: 1, Y: 0})
			} else if ebiten.IsKeyPressed(ebiten.KeyShift) {
				g.engine.GetGame().GetMap().Resize(g.engine.GetGame().GetMap().MapWidth+1, g.engine.GetGame().GetMap().MapHeight, emptyTile)
			} else {
				g.moveCameraOnMap(geometry.Point{X: 1, Y: 0})
			}
			g.gridIsDirty = true
		},
		core.KeyUpArrow: func() {
			if ebiten.IsKeyPressed(ebiten.KeyAlt) {
				g.engine.GetGame().GetMap().ShiftMapBy(geometry.Point{X: 0, Y: -1})
			} else if ebiten.IsKeyPressed(ebiten.KeyShift) {
				g.engine.GetGame().GetMap().Resize(g.engine.GetGame().GetMap().MapWidth, g.engine.GetGame().GetMap().MapHeight-1, emptyTile)
			} else {
				g.moveCameraOnMap(geometry.Point{X: 0, Y: -1})
			}
			g.gridIsDirty = true
		},
		core.KeyDownArrow: func() {
			if ebiten.IsKeyPressed(ebiten.KeyAlt) {
				g.engine.GetGame().GetMap().ShiftMapBy(geometry.Point{X: 0, Y: 1})
			} else if ebiten.IsKeyPressed(ebiten.KeyShift) {
				g.engine.GetGame().GetMap().Resize(g.engine.GetGame().GetMap().MapWidth, g.engine.GetGame().GetMap().MapHeight+1, emptyTile)
			} else {
				g.moveCameraOnMap(geometry.Point{X: 0, Y: 1})
			}
			g.gridIsDirty = true
		},
		core.KeyEscape: g.resetSelectionAndSwitchToDefaultState,
	}
	gridHeight := g.engine.ScreenGridHeight()
	gridWidth := g.engine.ScreenGridWidth()
	g.CurrentRune = '.'
	g.bottomMessageLabel = ui.NewSquareLabelWithWidth("", geometry.Point{X: 0, Y: gridHeight - 1}, gridWidth)
	g.topStatusLineLabel = ui.NewSquareLabelWithWidth("", geometry.Point{X: 0, Y: gridHeight - 3}, gridWidth)
	userInterface := g.engine.GetUI()
	userInterface.AddToScene(g.bottomMessageLabel)
	userInterface.AddToScene(g.topStatusLineLabel)

	g.createMenuBar(gridWidth, gridHeight)

	g.changeUIStateTo(editMapUI)
	g.ResizeAndClearMap(g.engine.MapWindowWidth(), g.engine.MapWindowHeight())
	g.selectionTool = NewPencil()
	g.updateStatusLine()

	toolTipFunc := func(origin geometry.Point, stringLength int) geometry.Rect { // from screen to half screen
		finalScreenHalfPos := userInterface.CalculateLabelPlacement(origin, stringLength)
		labelBounds := ui.NewBoundsForText(finalScreenHalfPos, stringLength)
		return labelBounds
	}
	userInterface.InitTooltip(toolTipFunc)
}
func (g *GameStateEditor) pickTileFromSelection() {
	if g.selectedWorldPositions == nil || len(g.selectedWorldPositions) == 0 {
		return
	}
	pos := g.selectedWorldPositions[0]
	cell := g.engine.GetGame().GetMap().CellAt(pos)
	tileCopy := cell.TileType
	g.currentBackgroundColor = tileCopy.DefinedStyle.Background
	g.currentForegroundColor = tileCopy.DefinedStyle.Foreground
	g.updateStatusLine()

	g.setBrushHandlerWithLightUpdate(editMapUI, tileCopy.Icon(), func(pos geometry.Point) {
		g.placeTileAtPos(tileCopy, pos)
	})()
	/*
		g.setBrushHandlerWithLightUpdate(setColorUI, core.GlyphPalette, func(point geometry.Point) {
			g.setBackgroundColorOfTileAt(point, g.currentBackgroundColor)
			g.setForegroundColorOfTileAt(point, g.currentForegroundColor)
		})()
	*/
}

func (g *GameStateEditor) pickColorFromSelection() {
	if g.selectedWorldPositions == nil || len(g.selectedWorldPositions) == 0 {
		return
	}
	pos := g.selectedWorldPositions[0]
	cell := g.engine.GetGame().GetMap().CellAt(pos)
	tileCopy := cell.TileType
	g.currentBackgroundColor = tileCopy.DefinedStyle.Background
	g.currentForegroundColor = tileCopy.DefinedStyle.Foreground
	g.updateStatusLine()

	g.setBrushHandlerWithLightUpdate(setColorUI, core.GlyphPalette, func(point geometry.Point) {
		g.setBackgroundColorOfTileAt(point, g.currentBackgroundColor)
		g.setForegroundColorOfTileAt(point, g.currentForegroundColor)
	})()
}

func (g *GameStateEditor) setKeyOfSelectedObject() {
	if _, isKeyed := g.selectedObject.(services.KeyBound); !isKeyed || g.selectedObject == nil {
		return
	}
	g.changeUIStateTo(addObjectsUI.WithTextHandler(func(text string) {
		keyedObj, _ := g.selectedObject.(services.KeyBound)
		keyedObj.SetKey(text)
		g.PrintAsMessage(fmt.Sprintf("Key of %s set to %s", g.selectedObject.Description(), text))
		g.SetDirty()
	}))
	g.showTextInput("Set Object Key: ", "")
}

func (g *GameStateEditor) setKeyOfSelectedItem() {
	if g.selectedItem == nil {
		return
	}
	g.changeUIStateTo(addItemsUI.WithTextHandler(func(text string) {
		g.selectedItem.SetKey(text)
		g.PrintAsMessage(fmt.Sprintf("Key of %s set to %s", g.selectedItem.Name, text))
		g.SetDirty()
	}))
	g.showTextInput("Set Item Key: ", "")
}

func (g *GameStateEditor) resetSelectionAndSwitchToDefaultState() {
	g.currentPrefab = nil
	g.SelectedZone = nil
	g.selectedObject = nil
	g.selectedItem = nil
	g.selectedWorldPositions = nil
	g.selectedNamedLocation = ""
	g.SelectedActor = nil
	g.SelectedLightSource = nil
	g.SelectedTaskIndex = 0
	g.changeUIStateTo(editMapUI)
}
