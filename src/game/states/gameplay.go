package states

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/utils"
	"path"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/director"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/mapset"
	"github.com/memmaker/terminal-assassin/ui"
)

type GameStateGameplay struct {
	engine                services.Engine
	Ui                    GameplayUIState
	MouseDown             bool
	MousePositionOnScreen geometry.Point
	MousePositionInWorld  geometry.Point
	topLabel              *ui.FixedLabel
	midLabel              *ui.FixedLabel
	Pager                 *ui.Pager
	TargetLoS             []geometry.Point
	FocusedActor          *core.Actor
	targetMovePath        []geometry.Point
	ActionMap             map[geometry.Point]services.ContextAction
	contextActionsHelp    string
	lastPlayerMovementAt  time.Time
	MoveTimer             *time.Timer

	initActors              bool
	DebugModeActive         bool
	DebugShowAllVisionCones bool
	isDirty                 bool
	aimScrollTimer          int
	flashPlayerHurt         int
	dialogueLabels          map[*core.Actor]*ui.MovableLabel
	clearHalfWidth          bool

	runningScripts []*director.Script
	mapDialogues   map[string]*director.DialogueInfo
}

func (g *GameStateGameplay) Print(text string) {
	g.midLabel.SetText(text)
	g.midLabel.SetDirty()
}

func (g *GameStateGameplay) PrintStyled(text core.StyledText) {
	g.midLabel.SetStyledText(text)
	g.midLabel.SetDirty()
}

func (g *GameStateGameplay) UpdateHUD() {
	g.UpdateContextActions()
}

type GameplayUIState struct {
	LeftClick func()
	LeftHeld  func()
	LeftDrag  func()
	MouseMove func()
	Draw      func(con console.CellInterface)
}

var defaultUIState, aimingUIState, examineUIState GameplayUIState

func (g *GameStateGameplay) ToNormalUIState() {
	g.Pager = nil
	currentMap := g.engine.GetGame().GetMap()
	currentMap.Player.Status = core.ActorStatusOnSchedule
	currentMap.Player.FovMode = gridmap.FoVModeNormal
	currentMap.Player.FovShiftForPeeking = geometry.PointZero
	g.Ui = defaultUIState
	currentMap.UpdateFieldOfView(currentMap.Player)
}

func (g *GameStateGameplay) Init(engine services.Engine) {
	g.engine = engine
	game := g.engine.GetGame()
	currentMap := game.GetMap()
	engine.ResetForGameplay()

	defaultUIState = GameplayUIState{
		Draw: g.drawMousePos,
		LeftClick: func() {
			if !currentMap.IsActorAt(g.MousePositionInWorld) || !g.DebugModeActive {
				return
			}
			g.SetFocusedActor(currentMap.ActorAt(g.MousePositionInWorld))
		},
	}

	aimingUIState = GameplayUIState{
		LeftClick: g.AimedItemUse,
		LeftDrag: func() {
			if !currentMap.Player.HasBurstWeaponEquipped() {
				return
			}
			g.AdjustPlayerAim()
			g.AimedItemUse()
		},
		LeftHeld: func() {
			if !currentMap.Player.HasBurstWeaponEquipped() {
				return
			}
			g.AimedItemUse()
		},
		MouseMove: g.AdjustPlayerAim,
		Draw:      g.drawTargetPath,
	}

	g.Ui = defaultUIState
	gridHeight := g.engine.ScreenGridHeight()
	gridWidth := g.engine.ScreenGridWidth()
	g.topLabel = ui.NewSquareLabelWithWidth("", geometry.Point{X: 0, Y: gridHeight - 3}, gridWidth)
	g.midLabel = ui.NewHalfLabelWithWidth("", geometry.Point{X: 0, Y: gridHeight - 2}, gridWidth)
	g.midLabel.SetAutoClearAndAnimations(true)
	g.Pager = nil
	g.ActionMap = make(map[geometry.Point]services.ContextAction)
	g.dialogueLabels = make(map[*core.Actor]*ui.MovableLabel)
	g.SpawnPlayer()
	g.initCamera()

	currentMap.UpdateBakedLights()
	currentMap.UpdateDynamicLights()
	g.UpdateHUD()
	g.isDirty = true
	g.clearHalfWidth = true
	audio := g.engine.GetAudio()

	if currentMap.AmbienceSoundCue != "" {
		audio.StartLoop(currentMap.AmbienceSoundCue)
	}
	audio.PreLoadCuesIntoMemory([]string{
		"slaughter_loop",
		"blade_loop",
		"pistol_shot",
		"shotgun_shot",
		"smg_shot",
		"sniper_shot",
		"throw",
		"vomiting",
		"small_explosion",
		"silenced_pistol",
		"silenced_rifle",
		"bullet_hit_head",
		"eating",
		"get-dressed",
		"flesh_hit",
	})

	stats := engine.GetGame().GetStats()
	stats.StartMission()

	userInterface := g.engine.GetUI()
	toolTipFunc := func(origin geometry.Point, stringLength int) geometry.Rect { // from screen to half screen

		finalScreenHalfPos := userInterface.CalculateLabelPlacement(origin, stringLength)
		labelBounds := ui.NewBoundsForText(finalScreenHalfPos, stringLength)
		topOverlap := false
		for _, actorLabel := range g.dialogueLabels {
			if actorLabel.ScreenBounds().Overlaps(labelBounds) {
				topOverlap = true
			}
		}
		if topOverlap {
			labelBounds = labelBounds.Add(geometry.Point{X: 0, Y: 2})
			for _, actorLabel := range g.dialogueLabels {
				if actorLabel.ScreenBounds().Overlaps(labelBounds) {
					return geometry.Rect{}
				}
			}
		}
		return labelBounds
	}
	userInterface.InitTooltip(toolTipFunc)

	g.engine.SubscribeToEvents(services.NewFilter(func(t services.ItemPickedUpEvent) bool {
		g.Print(fmt.Sprintf("%s: picked up %s.", t.Actor.Name, t.Item.Name))
		return true
	}))

	// load dialogues
	g.parseMapDialogues(currentMap)
	// load scripts, parse them and run them
	g.parseMapScripts(currentMap)

	println(fmt.Sprintf("MISSION LOADING COMPLETE - Player at %v", currentMap.Player.Pos()))
}

func (g *GameStateGameplay) parseMapDialogues(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object]) {
	g.mapDialogues = make(map[string]*director.DialogueInfo)
	files := g.engine.GetFiles()
	mapFolder := currentMap.MapFileName()
	scriptsPath := path.Join(mapFolder, "dialogues")
	dialogueFiles := files.GetFilesInPath(scriptsPath)
	if len(dialogueFiles) == 0 {
		return
	}
	parser := director.NewDialogueParser(currentMap.Player)
	g.registerPredicateAndAssignmentFunctions(parser)
	for _, dialogueFilename := range dialogueFiles {
		dialogueFile, err := files.Open(dialogueFilename)
		if err != nil {
			println("Error opening dialogue file: " + dialogueFilename)
			continue
		}
		parsedDialogue, parseError := parser.DialogueFromFile(dialogueFile)
		if parseError != nil {
			println("Error parsing dialogue file: " + dialogueFilename)
			continue
		}
		g.mapDialogues[parsedDialogue.Name] = parsedDialogue
		parser.LoadDialogForActors(parsedDialogue)
	}
}

func (g *GameStateGameplay) parseMapScripts(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object]) {
	g.runningScripts = make([]*director.Script, 0)
	files := g.engine.GetFiles()
	mapFolder := currentMap.MapFileName()
	scriptsPath := path.Join(mapFolder, "scripts")
	scriptFiles := files.GetFilesInPath(scriptsPath)
	if len(scriptFiles) == 0 {
		return
	}
	parser := director.NewScriptParser(currentMap.Player)
	g.registerPredicateAndAssignmentFunctions(parser)
	g.registerActionFunctions(parser)
	config := g.engine.GetGame().GetConfig()
	for _, scriptFilename := range scriptFiles {
		baseFilename := path.Base(scriptFilename)
		if strings.HasPrefix(baseFilename, "hint_") && !config.ShowHints {
			continue
		}
		scriptFile, err := files.Open(scriptFilename)
		if err == nil {
			parsedScript, parseError := parser.ScriptFromFile(scriptFile)
			if parseError != nil {
				println("Error parsing script file: " + scriptFilename)
				continue
			} else {
				g.runningScripts = append(g.runningScripts, parsedScript)
			}
		} else {
			println("Error opening script file: " + scriptFilename)
			continue
		}
	}
}

func (g *GameStateGameplay) initCamera() {
	game := g.engine.GetGame()
	currentMap := game.GetMap()
	playerPos := currentMap.Player.Pos()
	mapWidth, mapHeight := game.GetMap().MapWidth, game.GetMap().MapHeight
	game.GetCamera().CenterOn(playerPos, mapWidth, mapHeight)
}

func (g *GameStateGameplay) SpawnPlayer() {
	planning := g.engine.GetGame().GetMissionPlan()
	data := g.engine.GetData()
	career := g.engine.GetCareer()
	clothes := data.DefaultPlayerClothing()
	startLocation, _ := planning.Location()
	if planning.Clothes() != nil {
		clothes = *planning.Clothes()
	}
	visionRange := 10
	player := &core.Actor{
		Name:           career.PlayerName,
		AutoMoveSpeed:  3,
		MaxVisionRange: visionRange,
		Clothes:        clothes,
		FoVinDegrees:   90,
		Health:         3,
		MovementMode:   core.MovementModeWalking,
		Status:         core.ActorStatusIdle,
	}
	player.Fov = geometry.NewFOV(geometry.NewRect(-visionRange, -visionRange, visionRange+1, visionRange+1))
	player.Inventory = &core.InventoryComponent{Items: make([]*core.Item, 0)}
	player.Dialogue = &core.DialogueComponent{Conversations: make(map[string]*core.Conversation), SpokenSpeech: mapset.NewSet[string](), HeardSpeech: mapset.NewSet[string]()}

	weapon := planning.Weapon()
	if weapon != nil {
		player.Inventory.AddItem(weapon)
		weapon.HeldBy = player
	}
	gearOne := planning.GearOne()
	if gearOne != nil {
		player.Inventory.AddItem(gearOne)
		gearOne.HeldBy = player
	}
	gearTwo := planning.GearTwo()
	if gearTwo != nil {
		player.Inventory.AddItem(gearTwo)
		gearTwo.HeldBy = player
	}

	currentMap := g.engine.GetGame().GetMap()
	currentMap.Player = player
	currentMap.AddActor(player, startLocation)
	currentMap.UpdateFieldOfView(player)
}
func (g *GameStateGameplay) drawTargetPath(con console.CellInterface) {
	cam := g.engine.GetGame().GetCamera()
	//g.ensureWorldPosInView(g.MousePositionOnScreen, 4)
	mapWidth, mapHeight := g.engine.MapWindowWidth(), g.engine.MapWindowHeight()
	for _, p := range g.TargetLoS {
		screenPos := cam.WorldToScreen(p)
		if screenPos.Y < 0 || screenPos.Y >= mapHeight || screenPos.X < 0 || screenPos.X >= mapWidth {
			continue
		}
		c := con.AtSquare(screenPos)
		con.SetSquare(screenPos, c.WithStyle(c.Style.WithBg(common.IllegalActionRed)))
	}
}

func (g *GameStateGameplay) drawMousePos(con console.CellInterface) {
	// TODO: re-enable this
	/*
		m := g.engine.GetGame()
		c := con.AtSquare(g.MousePositionInWorld)
		con.SetSquare(g.MousePositionInWorld, c.WithStyle(c.Style

	*/
}

func (g *GameStateGameplay) UpdateContextActions() {
	game := g.engine.GetGame()
	currentMap := game.GetMap()
	player := currentMap.Player

	if player == nil {
		return
	}

	defer g.UpdateStatusLine()

	neighbors := currentMap.GetAllCardinalNeighbors(player.Pos())
	neighbors = append(neighbors, player.Pos())
	for _, absoluteMapPos := range neighbors {
		if !currentMap.Contains(absoluteMapPos) {
			continue
		}
		relativeNeighborPos := absoluteMapPos.Sub(player.Pos())
		contextAction := game.GetContextActionAt(absoluteMapPos)
		if contextAction == nil {
			delete(g.ActionMap, relativeNeighborPos)
		} else {
			g.ActionMap[relativeNeighborPos] = contextAction
		}
	}

	// shifted pickups (pickup has highest priority)
	shiftedPlayerPos := player.Pos().Add(player.FovShiftForPeeking)
	if currentMap.IsItemAt(shiftedPlayerPos) {
		itemAt := currentMap.ItemAt(shiftedPlayerPos)
		g.ActionMap[geometry.PointZero] = game.CreatePickupAction(itemAt)
	}

	// dialogue has lowest priority
	if _, ok := g.ActionMap[geometry.PointZero]; ok {
		return
	}
	// dialogue
	// check the cardinal positions at distance 2
	distTwoNeighbors := []geometry.Point{
		player.Pos().Add(geometry.Point{X: 2, Y: 0}),
		player.Pos().Add(geometry.Point{X: -2, Y: 0}),
		player.Pos().Add(geometry.Point{X: 0, Y: 2}),
		player.Pos().Add(geometry.Point{X: 0, Y: -2}),
	}
	for _, absoluteMapPos := range distTwoNeighbors {
		if !currentMap.Contains(absoluteMapPos) {
			continue
		}
		if currentMap.IsActorAt(absoluteMapPos) {
			actorAt := currentMap.ActorAt(absoluteMapPos)
			conversationName := actorAt.Dialogue.HasDialogueFor(player)
			if conversationName != "" {
				dialogueInfo := g.mapDialogues[conversationName]
				dialogueAction := game.StartDialogueAction(dialogueInfo.InitialSpeaker, dialogueInfo.Name)
				if dialogueAction.IsActionPossible(g.engine, player, absoluteMapPos) {
					g.ActionMap[geometry.PointZero] = dialogueAction
				}
			}
		}
	}
}

func (g *GameStateGameplay) playerMoved(oldPosition geometry.Point, newPosition geometry.Point) {
	g.Ui = defaultUIState

	defer g.UpdateContextActions()

	g.updatePlayerMovementMode(newPosition)

	g.printTileContentsMessage(newPosition)

	borderSize := 10

	g.ensureWorldPosInView(newPosition, borderSize)
}

func (g *GameStateGameplay) ensureWorldPosInView(worldPosition geometry.Point, borderSize int) {
	screenMapWidth := g.engine.MapWindowWidth()
	screenMapHeight := g.engine.MapWindowHeight()
	game := g.engine.GetGame()
	camera := game.GetCamera()
	mapWidth := game.GetMap().MapWidth
	mapHeight := game.GetMap().MapHeight
	newPositionOnScreen := camera.WorldToScreen(worldPosition)

	moveDelta := geometry.Point{X: 0, Y: 0}
	if newPositionOnScreen.X < borderSize {
		moveDelta.X = newPositionOnScreen.X - borderSize
	} else if newPositionOnScreen.X >= screenMapWidth-borderSize {
		moveDelta.X = newPositionOnScreen.X - (screenMapWidth - borderSize)
	}
	if newPositionOnScreen.Y < borderSize {
		moveDelta.Y = newPositionOnScreen.Y - borderSize
	} else if newPositionOnScreen.Y >= screenMapHeight-borderSize {
		moveDelta.Y = newPositionOnScreen.Y - (screenMapHeight - borderSize)
	}

	if moveDelta.X != 0 || moveDelta.Y != 0 {
		camera.MoveBy(moveDelta, mapWidth, mapHeight)
	}

}
func (g *GameStateGameplay) resetPlayerState() {
	player := g.engine.GetGame().GetMap().Player
	player.FovMode = gridmap.FoVModeNormal
	player.FovShiftForPeeking = geometry.PointZero
	g.Ui = defaultUIState
}

func (g *GameStateGameplay) updatePlayerMovementMode(newPosition geometry.Point) {
	timeNow := time.Now()

	timespanSinceLastMove := timeNow.Sub(g.lastPlayerMovementAt)
	msSinceLastMove := timespanSinceLastMove.Milliseconds()
	game := g.engine.GetGame()

	player := game.GetMap().Player

	if msSinceLastMove > 600 {
		player.MovementMode = core.MovementModeSneaking
	} else if msSinceLastMove > 300 {
		player.MovementMode = core.MovementModeWalking
		g.engine.Schedule(0.300, func() { g.downgradeMovementMode(newPosition) })
	} else {
		player.MovementMode = core.MovementModeRunning
		g.engine.Schedule(0.300, func() { g.downgradeMovementMode(newPosition) })
	}
	g.lastPlayerMovementAt = timeNow
}

func (g *GameStateGameplay) downgradeMovementMode(position geometry.Point) {
	player := g.engine.GetGame().GetMap().Player
	if player.Pos() != position {
		return
	}
	defer g.UpdateStatusLine()
	switch player.MovementMode {
	case core.MovementModeRunning:
		player.MovementMode = core.MovementModeWalking
		g.engine.Schedule(0.300, func() { g.downgradeMovementMode(position) })
	case core.MovementModeWalking:
		player.MovementMode = core.MovementModeSneaking
	}
}

func (g *GameStateGameplay) SetDirty() {
	g.isDirty = true
}

func (g *GameStateGameplay) Update(input services.InputInterface) {
	//g.startMission()
	game := g.engine.GetGame()
	aic := g.engine.GetAI()
	aic.Update()
	commands := input.PollGameCommands()
	noPointerCmd := false
	for _, command := range commands {
		switch typedCommand := command.(type) {
		case core.KeyCommand:
			g.handleKeyCommands(typedCommand)
			noPointerCmd = true
		case core.GameCommand:
			g.handleCommand(typedCommand)
			noPointerCmd = true
		case core.DirectionalGameCommand:
			g.handleDirectionalInput(typedCommand)
			noPointerCmd = true
		case core.PointerCommand:
			g.handleMouse(typedCommand)
		}
	}
	g.isDirty = true

	if len(commands) > 0 {
		if noPointerCmd {
			g.clearTooltip()
		}
	}

	g.midLabel.Update(input)

	g.scrollCameraForScope()
	g.updateScripts()
	g.updateDialogue()
	actions := game.GetActions()
	actions.Update()

	currentMap := game.GetMap()
	if currentMap.DynamicLightsChanged && game.GetConfig().LightSources {
		currentMap.UpdateDynamicLights()
	}
}

func (g *GameStateGameplay) handleKeyCommands(command core.KeyCommand) {
	training := NewTrainingHelper(g.engine)
	switch command.Key {
	case "F1":
		g.ToggleShowCompleteMap()
	case "F2":
		g.ShowFocusedActorPager()
	case "F3":
		//alertContextAction(g.engine)
		training.alertContextAction()
	}
}

func (g *GameStateGameplay) clearTooltip() {
	userInterface := g.engine.GetUI()
	userInterface.ClearTooltip()
}

func (g *GameStateGameplay) scrollCameraForScope() {
	game := g.engine.GetGame()
	player := game.GetMap().Player
	if player != nil && player.FovMode == gridmap.FoVModeScoped {
		if g.aimScrollTimer == 0 {
			g.ensureWorldPosInView(game.GetCamera().ScreenToWorld(g.MousePositionOnScreen), 4)
			g.AdjustPlayerAim()
			g.aimScrollTimer = 10
		} else {
			g.aimScrollTimer--
		}
	}
}

func (g *GameStateGameplay) handleMouse(command core.PointerCommand) {
	g.MousePositionOnScreen = command.Pos
	g.MousePositionInWorld = g.engine.GetGame().GetCamera().ScreenToWorld(command.Pos)
	switch command.Action {
	case core.MouseLeft:

		g.MouseDown = true
		if g.Ui.LeftClick != nil {
			g.Ui.LeftClick()
		}
	case core.MouseLeftHeld:
		g.MouseDown = true
		if g.Ui.LeftHeld != nil {
			g.Ui.LeftHeld()
		}
	case core.MouseLeftReleased:
		g.MouseDown = false

	case core.MouseMoved:
		if g.MouseDown && g.Ui.LeftDrag != nil {
			g.Ui.LeftDrag()
		} else if g.Ui.MouseMove != nil {
			g.Ui.MouseMove()
		}
		g.updateMouseOver()
	}
	return
}

const (
	OwnTileInput core.Key = "r"
	SouthInput   core.Key = "s"
	NorthInput   core.Key = "w"
	WestInput    core.Key = "a"
	EastInput    core.Key = "d"
)

func (g *GameStateGameplay) handleMovementInput(inputKey core.Key) {
	pdelta := geometry.Point{}
	switch inputKey {
	case SouthInput:
		pdelta = pdelta.Shift(0, 1)
	case WestInput:
		pdelta = pdelta.Shift(-1, 0)
	case EastInput:
		pdelta = pdelta.Shift(1, 0)
	case NorthInput:
		pdelta = pdelta.Shift(0, -1)
	}
	playerMovementFromInput(g, pdelta)

	pdelta = geometry.Point{}
	switch inputKey {
	case "g":
		pdelta = pdelta.Shift(0, 1)
	case "f":
		pdelta = pdelta.Shift(-1, 0)
	case "h":
		pdelta = pdelta.Shift(1, 0)
	case "t":
		pdelta = pdelta.Shift(0, -1)
	}
	playerPeekingFromInput(g, pdelta)
}

func playerPeekingFromInput(g *GameStateGameplay, pdelta geometry.Point) {
	game := g.engine.GetGame()
	currentMap := game.GetMap()
	player := currentMap.Player
	if (pdelta.X != 0 || pdelta.Y != 0) && player.CanMove() && g.canPeekFrom(player.Pos().Add(pdelta)) {
		player.FovShiftForPeeking = pdelta

	} else {
		player.FovShiftForPeeking = geometry.PointZero
	}
	currentMap.UpdateFieldOfView(player)
	g.AdjustPlayerAim()
	g.UpdateHUD()
}

func (g *GameStateGameplay) canPeekFrom(worldPos geometry.Point) bool {
	game := g.engine.GetGame()
	currentMap := game.GetMap()
	return currentMap.IsCurrentlyPassable(worldPos) || game.IsDoorAt(worldPos)
}

func playerMovementFromInput(g *GameStateGameplay, pdelta geometry.Point) {
	m := g.engine.GetGame()
	currentMap := m.GetMap()
	player := currentMap.Player
	if (pdelta.X != 0 || pdelta.Y != 0) && player.CanMove() {
		oldPos := player.Pos()
		np := player.Pos().Add(pdelta)
		if np != player.Pos() && currentMap.CurrentlyPassableForActor(player)(np) {
			g.resetPlayerState()
			m.MoveActor(player, np)
			g.playerMoved(oldPos, np)
		}
	}
	return
}

func (g *GameStateGameplay) updatePager() {
	g.Pager.Update(g.engine.GetInput())
	if g.Pager.Action() == ui.PagerQuit {
		g.Pager = nil
	}
}

// OUR FPS DROP SEEMS TO BE CAUSED BY THIS FUNCTION
func (g *GameStateGameplay) Draw(con console.CellInterface) {
	m := g.engine.GetGame()
	if ebiten.ActualFPS() < 55 && g.engine.CurrentTick()%60 == 0 {
		println(fmt.Sprintf("Low FPS: %.2f", ebiten.ActualFPS()))
		defer utils.TimeTrack(time.Now(), "GameStateGameplay->Draw()")
	}

	if g.Pager != nil {
		g.Pager.Draw(con)
		return
	}

	if g.clearHalfWidth {
		con.HalfWidthTransparent()
		g.clearHalfWidth = false
	}

	if !g.isDirty {
		return
	}

	player := m.GetMap().Player
	//ambientLightness := m.GetMap().AmbientLight.RelativeLuminance()
	defaultCellStyle := common.DefaultStyle
	if g.flashPlayerHurt != len(player.DamageTaken) {
		g.flashPlayerHurt++
		defaultCellStyle = defaultCellStyle.WithBg(core.ColorFromCode(core.ColorBlood))
	}
	m.GetMap().IterWindow(m.GetCamera().ViewPort, func(p geometry.Point, c gridmap.MapCell[*core.Actor, *core.Item, services.Object]) {
		icon := ' '
		style := defaultCellStyle
		var cellIsVisibleToPlayer bool

		if c.IsExplored || g.DebugModeActive {
			cellIsVisibleToPlayer = player.CanSee(p) || g.DebugModeActive
			if cellIsVisibleToPlayer {
				icon, style = m.DrawWorldAtPosition(p, c) // >75% of cell draw execution time
				style = m.ApplyLighting(p, c, style)      // TODO: this should be precaclulated..
			} else {
				icon, style = m.DrawMapAtPosition(p, c)
				style = style.Desaturate()
				style = style.Darken(0.2)
				//style = style.Darken(ambientLightness)
			}
			posRelativeToPlayer := p.Sub(player.Pos())
			if action, isActionAtPos := g.ActionMap[posRelativeToPlayer]; isActionAtPos {
				_, actionStyle := action.Description(g.engine, player, p)
				style = style.WithBg(actionStyle.Background)
			}
		}
		drawPos := m.GetCamera().WorldToScreen(p)
		con.SetSquare(drawPos, common.Cell{Rune: icon, Style: style})
	})

	emptyPoint := geometry.Point{}
	if m.GetMap().Player.FovShiftForPeeking != emptyPoint {
		fovSourcePos := m.GetMap().Player.Pos().Add(m.GetMap().Player.FovShiftForPeeking)
		screenFovSource := m.GetCamera().WorldToScreen(fovSourcePos)
		currentCell := con.AtSquare(screenFovSource)
		con.SetSquare(screenFovSource, currentCell.WithStyle(currentCell.Style.WithBg(core.ColorFromCode(core.ColorFoVSource))))
	}
	for _, actor := range m.GetMap().Actors() {
		if !actor.IsPlayer() && actor.AI.SuspicionCounter > 0 {
			m.DrawVisionCone(con, actor)
		}
	}

	actions := g.engine.GetGame().GetActions()
	actions.Draw(con)

	for _, label := range g.dialogueLabels {
		label.Draw(con)
	}
	g.topLabel.Draw(con)
	g.midLabel.Draw(con)

	if g.Ui.Draw != nil {

		g.Ui.Draw(con)
	}
	g.drawContextHUD(con)

	g.isDirty = false
}

func (g *GameStateGameplay) UpdateStatusLine() {
	m := g.engine.GetGame()
	itemSymbol := string(core.GlyphEmptyHand)
	itemStyle := common.DefaultStyle
	hpStyle := common.DefaultStyle
	clothingStyle := common.DefaultStyle
	detectionStyle := common.DefaultStyle
	currentMap := m.GetMap()
	player := currentMap.Player

	if player.EquippedItem != nil {
		itemSymbol = string(player.EquippedItem.Icon())
		if !player.EquippedItem.IsLegalForActor(player) {
			itemStyle = itemStyle.WithFg(core.ColorFromCode(core.ColorBlood))
		} else if player.EquippedItem.Type == core.ItemTypeEmeticPoison {
			itemStyle = itemStyle.WithFg(core.ColorFromCode(core.ColorPoisonEmetic))
		} else if player.EquippedItem.Type == core.ItemTypeLethalPoison {
			itemStyle = itemStyle.WithFg(core.ColorFromCode(core.ColorPoisonLethal))
		}
		if player.EquippedItem.Uses > 0 {
			itemSymbol += fmt.Sprintf(" (%d)", player.EquippedItem.Uses)
		}
	}
	if player.Health <= 1 {
		hpStyle = hpStyle.WithFg(core.ColorFromCode(core.ColorBlood))
	} else if player.Health == 2 {
		hpStyle = hpStyle.WithFg(core.ColorFromCode(core.ColorWarning))
	} else {
		hpStyle = hpStyle.WithFg(core.ColorFromCode(core.ColorGood))
	}

	clothingStyle = clothingStyle.WithFg(player.Clothes.FgColor) //.WithBg(player.Clothes.Color)

	healthString := createHealthBar(player.Health)

	zoneInformation := currentMap.ZoneAt(player.Pos()).Name
	if currentMap.IsInHostileZone(player) {
		zoneInformation = "hostile area"
		detectionStyle = detectionStyle.WithBg(core.ColorFromCode(core.ColorWarning)).WithFg(common.Black)
	} else if currentMap.IsTrespassing(player) {
		zoneInformation = "trespassing"
		detectionStyle = detectionStyle.WithBg(core.ColorFromCode(core.ColorWarning)).WithFg(common.Black)
	}

	challengeInformation := ""
	if g.DebugModeActive {
		challengeInformation = fmt.Sprintf("%.2f", ebiten.ActualFPS())
	} else {
		stats := m.GetStats()
		if stats.BeenSpotted {
			challengeInformation += "@rS@N"
		} else {
			challengeInformation += "@gS@N"
		}

		if stats.BodiesFound {
			challengeInformation += "@rB@N"
		} else {
			challengeInformation += "@gB@N"
		}

		if stats.DisguisesWorn.Cardinality() > 0 {
			challengeInformation += "@rC@N"
		} else {
			challengeInformation += "@gC@N"
		}
	}

	witnessCount := g.witnessCount()
	if witnessCount > 0 {
		challengeInformation += fmt.Sprintf("@r%d@N", witnessCount)
	} else {
		challengeInformation += "@gW@N"
	}

	redStyle := common.DefaultStyle.WithBg(core.ColorFromCode(core.ColorBlood))
	greenStyle := common.DefaultStyle.WithBg(core.ColorFromCode(core.ColorGood))
	g.topLabel.SetStyledText(core.Text(fmt.Sprintf("@c%s@N | @h%s@N | %s | @i%s@N | @d%s@N | %s", player.Clothes.Name, healthString, string(player.MovementMode), itemSymbol, zoneInformation, challengeInformation)).
		WithStyle(common.DefaultStyle).
		WithMarkup('c', clothingStyle).
		WithMarkup('i', itemStyle).
		WithMarkup('d', detectionStyle).
		WithMarkup('h', hpStyle).
		WithMarkup('r', redStyle).
		WithMarkup('g', greenStyle))

	g.isDirty = true
}
func (g *GameStateGameplay) witnessCount() int {
	witnessCount := 0
	currentMap := g.engine.GetGame().GetMap()
	for _, actor := range currentMap.Actors() {
		if actor.IsEyeWitness {
			witnessCount++
		}
	}
	for _, downedActor := range currentMap.DownedActors() {
		if downedActor.IsAlive() && downedActor.IsEyeWitness {
			witnessCount++
		}
	}
	return witnessCount
}
func (g *GameStateGameplay) drawContextHUD(con console.CellInterface) {
	gridWidth := g.engine.ScreenGridWidth()
	gridHeight := g.engine.ScreenGridHeight()
	userInterface := g.engine.GetUI()
	input := g.engine.GetInput()
	keyDefs := input.GetKeyDefinitions()
	hudHeight := userInterface.HUDHeight()

	yPosStart := gridHeight - hudHeight
	xPosStart := gridWidth - 6

	westPos := geometry.Point{X: xPosStart, Y: yPosStart + 1}
	northPos := geometry.Point{X: xPosStart + 2, Y: yPosStart}
	centerPos := geometry.Point{X: xPosStart + 2, Y: yPosStart + 1}
	southPos := geometry.Point{X: xPosStart + 2, Y: yPosStart + 2}
	eastPos := geometry.Point{X: xPosStart + 4, Y: yPosStart + 1}

	g.drawContextHUDIcon(con, geometry.RelativeNorth, northPos, rune(keyDefs.ActionKeys[0].String()[0]))
	g.drawContextHUDIcon(con, geometry.RelativeWest, westPos, rune(keyDefs.ActionKeys[1].String()[0]))
	g.drawContextHUDIcon(con, geometry.RelativeSouth, southPos, rune(keyDefs.ActionKeys[2].String()[0]))
	g.drawContextHUDIcon(con, geometry.RelativeEast, eastPos, rune(keyDefs.ActionKeys[3].String()[0]))
	g.drawContextHUDIcon(con, geometry.PointZero, centerPos, rune(keyDefs.SameTileActionKey.String()[0]))
}

func (g *GameStateGameplay) drawContextHUDIcon(con console.CellInterface, relativePos, hudIconPos geometry.Point, emptyRune rune) {
	contextAction := g.ActionMap[relativePos]
	if contextAction != nil {
		engine := g.engine
		player := engine.GetGame().GetMap().Player
		icon, style := contextAction.Description(engine, player, player.Pos().Add(relativePos))
		con.SetSquare(hudIconPos, common.Cell{Rune: icon, Style: style})
	} else {
		con.SetSquare(hudIconPos, common.Cell{Rune: emptyRune, Style: common.DefaultStyle})
	}
}
func createHealthBar(health int) string {
	if health < 0 {
		health = 0
	}
	fullGlyph := '£'
	emptyGlyph := '¡'
	hp := strings.Repeat(string(fullGlyph), health)
	hp = hp + strings.Repeat(string(emptyGlyph), 3-health)
	return hp
}

func (g *GameStateGameplay) drawMovePath(con console.CellInterface) {
	for _, p := range g.targetMovePath {
		c := con.AtSquare(p)
		//con.SetSquare(p, c.WithStyle(c.Style.WithAttrs(AttrReverse)))
		con.SetSquare(p, c.WithStyle(c.Style.WithBg(core.ColorFromCode(core.ColorMapBackgroundLight))))
	}
}

func (g *GameStateGameplay) handleCommand(command core.GameCommand) {
	model := g.engine.GetGame()
	input := g.engine.GetInput()
	userInterface := g.engine.GetUI()
	player := model.GetMap().Player
	switch command {
	case core.StartSneaking:
		input.SetMovementDelayForSneaking()
	case core.StopSneaking:
		input.SetMovementDelayForWalkingAndRunning()
	case core.NextItem:
		g.EquipNextInventoryItem(player)
	case core.OpenInventory:
		userInterface.OpenItemRingMenu(player.EquippedItem, player.Inventory.Items, func(item *core.Item) {
			//player := m.engine.GetGame().GetMap().Player
			if player.EquippedItem != nil && player.EquippedItem != item && player.EquippedItem.IsBig {
				g.engine.GetGame().DropEquippedItem(player)
			}
			player.EquippedItem = item
			g.UpdateHUD()
		}, func() {
			g.UpdateHUD()
		})
	case core.DropItem:
		model.DropEquippedItem(player)
		g.UpdateHUD()
	case core.PickUpItem:
		model.PickUpItem(player)
		g.UpdateHUD()
	case core.PutItemAway:
		player.PutItemAway()
		g.UpdateHUD()
	case core.UseEquippedItem:
		g.AimedItemUse()
		g.UpdateHUD()
	case core.UseItem:
		g.BeginAimOrUseItem()
	case core.Cancel:
		g.OpenPauseMenu()
	}
}

func (g *GameStateGameplay) handleDirectionalInput(directionalCommand core.DirectionalGameCommand) {
	switch directionalCommand.Command {
	case core.MovementDirection:
		playerMovementFromInput(g, toIntDirection(directionalCommand.XAxis, directionalCommand.YAxis))
	case core.PeekingDirection:
		playerPeekingFromInput(g, toIntDirection(directionalCommand.XAxis, directionalCommand.YAxis))
	case core.ActionDirection:
		g.playerDirectionalActionFromInput(toIntDirection(directionalCommand.XAxis, directionalCommand.YAxis))
	case core.AimingDirection:
		g.AdjustPlayerAimFromPad(directionalCommand.XAxis, directionalCommand.YAxis)
	}
	return
}

func toIntDirection(xAxis float64, yAxis float64) geometry.Point {
	x := 0
	y := 0
	if xAxis > 0.5 {
		x = 1
	} else if xAxis < -0.5 {
		x = -1
	}
	if yAxis > 0.5 {
		y = 1
	} else if yAxis < -0.5 {
		y = -1
	}
	return geometry.Point{X: x, Y: y}
}

type LightSource struct {
	Radius int
	Power  float64
}

type MapLighter struct {
	CurrentMap   *gridmap.GridMap[*core.Actor, *core.Item, services.Object]
	LightSources map[geometry.Point]LightSource
}

func (m MapLighter) Cost(src geometry.Point, from geometry.Point, to geometry.Point) int {
	return geometry.DistanceManhattan(from, to)
	//return DistanceSquared(from, to)
}

func (m MapLighter) MaxCost(src geometry.Point) int {
	light := m.LightSources[src]
	return light.Radius
}

func (g *GameStateGameplay) playerDirectionalActionFromInput(direction geometry.Point) {
	player := g.engine.GetGame().GetMap().Player
	if !player.CanUseActions() {
		return
	}
	if directionalAction, valid := g.ActionMap[direction]; valid {
		directionalAction.Action(g.engine, player, player.Pos().Add(direction))
		g.UpdateHUD()
	}
	return
}
func (g *GameStateGameplay) printTileContentsMessage(position geometry.Point) {
	eng := g.engine
	m := eng.GetGame()
	tileType := m.GetMap().CellAt(position).TileType
	if m.GetMap().IsItemAt(position) {
		itemHere := m.GetMap().ItemAt(position)
		g.Print(fmt.Sprintf("There is %s here.", itemHere.Name))
	} else if tileType.Special != gridmap.SpecialTileDefaultFloor {
		tileName := tileType.Description()
		g.Print(fmt.Sprintf("There is %s here.", tileName))
	}
}

func (g *GameStateGameplay) updateMouseOver() {
	model := g.engine.GetGame()
	currentMap := model.GetMap()
	mouseInWorld := g.MousePositionInWorld
	userInterface := g.engine.GetUI()

	g.isDirty = true

	if !currentMap.Contains(mouseInWorld) || !currentMap.Player.CanSee(mouseInWorld) {
		if userInterface.TooltipShown() {
			g.clearTooltip()
		}
		return
	}
	mapCell := currentMap.CellAt(mouseInWorld)
	if currentMap.IsActorAt(mouseInWorld) {
		actorAtMouse := currentMap.ActorAt(mouseInWorld)
		g.showActorTooltip(actorAtMouse)
	} else if currentMap.IsDownedActorAt(mouseInWorld) {
		actorAtMouse := currentMap.DownedActorAt(mouseInWorld)
		g.showActorTooltip(actorAtMouse)
	} else if currentMap.IsItemAt(mouseInWorld) {
		itemHere := currentMap.ItemAt(mouseInWorld)
		g.showItemTooltip(itemHere)
	} else if currentMap.IsObjectAt(mouseInWorld) {
		objectHere := currentMap.ObjectAt(mouseInWorld)
		g.showObjectTooltip(objectHere)
	} else if mapCell.TileType.Special != gridmap.SpecialTileDefaultFloor {
		g.showCellTooltip(mouseInWorld, mapCell)
	} else if userInterface.TooltipShown() {
		userInterface.ClearTooltip()
	}
}

func (g *GameStateGameplay) showTooltipAt(screenPos geometry.Point, styledText core.StyledText) {
	userInterface := g.engine.GetUI()
	userInterface.ShowTooltipAt(screenPos, styledText)
}

func (g *GameStateGameplay) showActorTooltip(person *core.Actor) {
	infoString := fmt.Sprintf("%s / %s", person.Name, person.NameOfClothing())
	if person.Status != "" {
		infoString = fmt.Sprintf("%s / %s / %s", person.Name, person.NameOfClothing(), person.Status)
	}
	tooltipFontColor := common.Black
	if person.Type == core.ActorTypeTarget {
		tooltipFontColor = common.NewRGBColorFromBytes(255, 51, 51)
	}
	styledText := core.Text(infoString).WithStyle(common.Style{Foreground: tooltipFontColor, Background: common.FourWhite})
	screenPos := g.engine.GetGame().GetCamera().WorldToScreen(person.Pos())
	g.showTooltipAt(screenPos, styledText)
}

func (g *GameStateGameplay) showItemTooltip(item *core.Item) {
	infoString := fmt.Sprintf("%s", item.Name)
	styledText := core.Text(infoString).WithStyle(common.Style{Foreground: common.Black, Background: common.FourWhite})
	screenPos := g.engine.GetGame().GetCamera().WorldToScreen(item.Pos())
	g.showTooltipAt(screenPos, styledText)
}

func (g *GameStateGameplay) showObjectTooltip(object services.Object) {
	infoString := fmt.Sprintf("%s", object.Description())
	styledText := core.Text(infoString).WithStyle(common.Style{Foreground: common.Black, Background: common.FourWhite})
	screenPos := g.engine.GetGame().GetCamera().WorldToScreen(object.Pos())
	g.showTooltipAt(screenPos, styledText)
}
func (g *GameStateGameplay) showCellTooltip(pos geometry.Point, cell gridmap.MapCell[*core.Actor, *core.Item, services.Object]) {
	infoString := fmt.Sprintf("%s", cell.TileType.Description())
	styledText := core.Text(infoString).WithStyle(common.Style{Foreground: common.Black, Background: common.FourWhite})
	screenPos := g.engine.GetGame().GetCamera().WorldToScreen(pos)
	g.showTooltipAt(screenPos, styledText)
}

func (g *GameStateGameplay) updateDialogue() {
	for actor, label := range g.dialogueLabels {
		if actor.IsDowned() || label.GetText() == "" {
			label.Clear()
		} else {
			label.Update(nil)
		}
	}
	//camera := g.engine.GetGame().GetCamera()
	for _, actor := range g.engine.GetGame().GetMap().Actors() {
		if actor.IsDowned() {
			continue
		}

		if actor.HasLineToSpeak(g.engine.CurrentTick()) {
			g.showNextUtteranceFor(actor)
		} else if !actor.Dialogue.Active(g.engine.CurrentTick()) && actor.Dialogue.CurrentDialogue != "" {
			println(fmt.Sprintf("%s ended dialogue (%s)", actor.DebugDisplayName(), actor.Dialogue.CurrentDialogue))
			actor.Dialogue.CurrentDialogue = ""
			actor.Dialogue.Situation = nil
		}
	}

}

func (g *GameStateGameplay) showNextUtteranceFor(actor *core.Actor) {
	ok := false
	var actorLabel *ui.MovableLabel
	if actorLabel, ok = g.dialogueLabels[actor]; !ok {
		actorLabelBoundsFunc := func(origin geometry.Point, stringLength int) geometry.Rect {
			return g.engine.GetUI().BoundsForWorldLabel(origin, stringLength)
		}
		actorLabel = ui.NewMovableLabel(actorLabelBoundsFunc)
		g.dialogueLabels[actor] = actorLabel
	}
	g.isDirty = true
	soundObservation := core.Observation(actor.Dialogue.NextUtterance.EventCode)

	g.engine.Schedule(3, func() {
		if actor.IsDowned() {
			g.engine.GetGame().SoundEventAt(actor.Pos(), core.ObservationDownedSpeaker, 10)
		} else {
			g.engine.GetGame().SoundEventAt(actor.Pos(), soundObservation, 10)
		}
		actorLabel.Clear()
		actor.Dialogue.IsCurrentlySpeaking = false
		g.isDirty = true
	})

	actor.Dialogue.IsCurrentlySpeaking = true
	player := g.engine.GetGame().GetMap().Player
	if actor == player || player.CanSeeActor(actor) {

		userInterface := g.engine.GetUI()
		dialogueStyle := actor.ChatStyle()
		dialogueLength := actor.Dialogue.NextUtterance.Line.Size().X
		actorLabelBoundsFunc := func(origin geometry.Point, stringLength int) geometry.Rect {
			return userInterface.BoundsForWorldLabel(origin, stringLength)
		}
		actorLabel.Set(actor.Pos(), actor.Dialogue.NextUtterance.Line.WithStyle(dialogueStyle))
		screenLabelRect := actorLabelBoundsFunc(actor.Pos(), dialogueLength)
		if userInterface.IntersectsTooltip(screenLabelRect) {
			userInterface.ClearTooltip()
		}
	}

	actor.Dialogue.DidSpeak(actor, g.engine.CurrentTick())
}

func (g *GameStateGameplay) updateScripts() {
	for i := len(g.runningScripts) - 1; i >= 0; i-- {
		script := g.runningScripts[i]
		if script.IsFinished() {
			g.runningScripts = append(g.runningScripts[:i], g.runningScripts[i+1:]...)
		}
	}
}

func (g *GameStateGameplay) ClearOverlay() {
	g.clearHalfWidth = true
}
