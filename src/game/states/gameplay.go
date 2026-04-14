package states

import (
    "fmt"
    "github.com/memmaker/terminal-assassin/utils"
    "math"
    "math/rand"
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

    MoveTimer *time.Timer

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

    flashlightLight *gridmap.LightSource // dynamic light that tracks the flashlight aim point

    padAimPos    geometry.PointF // accumulated gamepad aim cursor position
    padAimActive bool            // true while aiming via gamepad (suppresses mouse-based scope scroll)

    lookModeActive bool            // true while free look cursor is active (Select toggle)
    lookCursorPos  geometry.PointF // accumulated look cursor position in world space

    // timeAccumulator counts Update ticks since the last in-game minute was
    // advanced.  At 60 TPS one real second equals one in-game minute, so a full
    // day/night cycle takes 24 real minutes.
    timeAccumulator int

    assassinationTargets map[*core.Actor]struct{} // NPCs highlighted for assassination
}

func (g *GameStateGameplay) Print(text string) {
    g.midLabel.SetText(text)
    g.midLabel.SetDirty()
}

func (g *GameStateGameplay) PrintStyled(text core.StyledText) {
    g.midLabel.SetStyledText(text)
    g.midLabel.SetDirty()
}

type GameplayUIState struct {
    ID        int8
    LeftClick func()
    LeftHeld  func()
    LeftDrag  func()
    MouseMove func()
    Draw      func(con console.CellInterface)
}

var defaultUIState, aimingUIState, examineUIState, lookUIState GameplayUIState

func (g *GameStateGameplay) ToNormalUIState() {
    g.Pager = nil
    currentMap := g.engine.GetGame().GetMap()
    currentMap.Player.Status = core.ActorStatusOnSchedule
    currentMap.Player.FovMode = gridmap.FoVModeNormal
    currentMap.Player.FoVShift = geometry.PointZero
    currentMap.Player.InteractionShift = geometry.PointZero
    g.Ui = defaultUIState
    currentMap.UpdateFieldOfView(currentMap.Player)
}

func (g *GameStateGameplay) Init(engine services.Engine) {
    g.engine = engine
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    engine.ResetForGameplay()

    defaultUIState = GameplayUIState{
        ID:   0,
        Draw: g.drawMousePos,
        LeftClick: func() {
            if !currentMap.IsActorAt(g.MousePositionInWorld) || !g.DebugModeActive {
                return
            }
            g.SetFocusedActor(currentMap.ActorAt(g.MousePositionInWorld))
        },
    }

    aimingUIState = GameplayUIState{
        ID:        1,
        LeftClick: g.UseRangedItemInLoS,
        LeftDrag: func() {
            if !currentMap.Player.HasBurstWeaponEquipped() {
                return
            }
            g.AdjustPlayerAimFromMouse()
            g.UseRangedItemInLoS()
        },
        LeftHeld: func() {
            if !currentMap.Player.HasBurstWeaponEquipped() {
                return
            }
            g.UseRangedItemInLoS()
        },
        MouseMove: g.AdjustPlayerAimFromMouse,
        Draw:      g.drawTargetPath,
    }

    examineUIState = GameplayUIState{
        ID: 2,
    }

    lookUIState = GameplayUIState{
        ID:   3,
        Draw: g.drawLookCursor,
    }

    g.Ui = defaultUIState
    gridHeight := g.engine.ScreenGridHeight()
    gridWidth := g.engine.ScreenGridWidth()
    g.topLabel = ui.NewSquareLabelWithWidth("", geometry.Point{X: 0, Y: gridHeight - 2}, gridWidth)
    g.midLabel = ui.NewHalfLabelWithWidth("", geometry.Point{X: 0, Y: gridHeight - 1}, gridWidth*2)
    g.midLabel.SetAutoClearAndAnimations(true)
    g.Pager = nil
    g.ActionMap = make(map[geometry.Point]services.ContextAction)
    g.dialogueLabels = make(map[*core.Actor]*ui.MovableLabel)
    g.flashlightLight = &gridmap.LightSource{
        Radius:       10,
        Color:        common.RGBAColor{R: 1.5, G: 1.5, B: 1.2, A: 1.0},
        MaxIntensity: 15,
    }
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
        if t.Actor != g.engine.GetGame().GetMap().Player {
            g.Print(fmt.Sprintf("%s picked up %s.", t.Actor.Name, t.Item.Name))
        } else {
            g.Print(fmt.Sprintf("Picked up %s.", t.Item.Name))
        }
        return true
    }))

    // HUD / message routing — replaces the type-asserting UpdateHUD / PrintMessage on Model.
    g.engine.SubscribeToEvents(services.NewFilter(func(_ services.HUDDirtyEvent) bool {
        g.UpdateHUD()
        return true
    }))
    g.engine.SubscribeToEvents(services.NewFilter(func(e services.PrintMessageEvent) bool {
        g.Print(e.Text)
        return true
    }))

    // Mission-stats tracking — decoupled from model/ai/animation internals.
    g.engine.SubscribeToEvents(services.NewFilter(func(_ services.PlayerSpottedEvent) bool {
        stats.BeenSpotted = true
        return true
    }))
    g.engine.SubscribeToEvents(services.NewFilter(func(_ services.BodyDiscoveredEvent) bool {
        stats.BodiesFound = true
        return true
    }))
    g.engine.SubscribeToEvents(services.NewFilter(func(e services.ActorKilledEvent) bool {
        stats.AddKill(e.Victim, e.CauseOfDeath, e.Position, utils.UTicksToSeconds(g.engine.CurrentTick()))
        return true
    }))
    g.engine.SubscribeToEvents(services.NewFilter(func(e services.PlayerChangedClothesEvent) bool {
        stats.DisguisesWorn.Add(e.NewClothing.Name)
        return true
    }))

    // Ambience sound — reacts to zone changes instead of running in playerEnteredCell.
    g.engine.SubscribeToEvents(services.NewFilter(func(e services.ActorEnteredZoneEvent) bool {
        if e.Actor != g.engine.GetGame().GetMap().Player {
            return true
        }
        audioPlayer := g.engine.GetAudio()
        currentGameMap := g.engine.GetGame().GetMap()
        if e.OldZone.AmbienceCue != "" {
            audioPlayer.Stop(e.OldZone.AmbienceCue)
        } else if currentGameMap.AmbienceSoundCue != "" && e.NewZone.AmbienceCue != "" {
            audioPlayer.Stop(currentGameMap.AmbienceSoundCue)
        }
        if e.NewZone.AmbienceCue != "" {
            audioPlayer.StartLoop(e.NewZone.AmbienceCue)
        } else if currentGameMap.AmbienceSoundCue != "" && !audioPlayer.IsCuePlaying(currentGameMap.AmbienceSoundCue) {
            audioPlayer.StartLoop(currentGameMap.AmbienceSoundCue)
        }
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
    visionRange := 90
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
    player := g.engine.GetGame().GetMap().Player
    pathColor := core.CurrentTheme.LOSBackground
    if player.EquippedItem != nil && player.EquippedItem.Type == core.ItemTypeFlashlight {
        return
    }
    //g.ensureWorldPosInView(g.MousePositionOnScreen, 4)
    mapWidth, mapHeight := g.engine.MapWindowWidth(), g.engine.MapWindowHeight()
    for _, p := range g.TargetLoS {
        screenPos := cam.WorldToScreen(p)
        if screenPos.Y < 0 || screenPos.Y >= mapHeight || screenPos.X < 0 || screenPos.X >= mapWidth {
            continue
        }
        c := con.AtSquare(screenPos)
        con.SetSquare(screenPos, c.WithStyle(c.Style.WithBg(pathColor)))
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

func (g *GameStateGameplay) UpdateHUD() {
    g.updateContextActions()
    g.UpdateStatusLine()
}

func (g *GameStateGameplay) updateContextActions() {
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    player := currentMap.Player

    if player == nil {
        return
    }

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

    defer g.UpdateHUD()

    g.updatePlayerMovementMode(newPosition)

    g.printTileContentsMessage(newPosition)

    borderSize := 4

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
    player.FoVShift = geometry.PointZero
    player.InteractionShift = geometry.PointZero
    g.Ui = defaultUIState
    g.padAimActive = false
    g.lookModeActive = false
}

// toggleLookMode enters or exits the free look cursor mode.
func (g *GameStateGameplay) toggleLookMode() {
    if g.lookModeActive {
        // Exit look mode
        g.lookModeActive = false
        g.Ui = defaultUIState
        g.clearTooltip()
        player := g.engine.GetGame().GetMap().Player
        g.ensureWorldPosInView(player.Pos(), 4)
        return
    }
    // Enter look mode: start cursor at player position
    player := g.engine.GetGame().GetMap().Player
    g.lookModeActive = true
    g.lookCursorPos = player.Pos().ToPointF()
    g.Ui = lookUIState
    g.showTooltipForWorldPos(player.Pos())
}

// moveLookCursor moves the free look cursor by the analog stick deflection.
func (g *GameStateGameplay) moveLookCursor(xAxis, yAxis float64) {
    const lookSpeed = 0.18 // tiles per tick at full deflection
    const deadzone = 0.08

    if math.Abs(xAxis) < deadzone && math.Abs(yAxis) < deadzone {
        return
    }

    g.lookCursorPos.X += xAxis * lookSpeed
    g.lookCursorPos.Y += yAxis * lookSpeed

    worldPos := g.lookCursorPos.ToPointRounded()
    g.showTooltipForWorldPos(worldPos)
    g.ensureWorldPosInView(worldPos, 4)
    g.isDirty = true
}

// drawLookCursor draws a highlighted cell at the current look cursor position.
func (g *GameStateGameplay) drawLookCursor(con console.CellInterface) {
    cam := g.engine.GetGame().GetCamera()
    worldPos := g.lookCursorPos.ToPointRounded()
    screenPos := cam.WorldToScreen(worldPos)

    mapWidth, mapHeight := g.engine.MapWindowWidth(), g.engine.MapWindowHeight()
    if screenPos.X < 0 || screenPos.X >= mapWidth || screenPos.Y < 0 || screenPos.Y >= mapHeight {
        return
    }
    c := con.AtSquare(screenPos)
    con.SetSquare(screenPos, c.WithStyle(c.Style.WithBg(core.CurrentTheme.LOSBackground)))
}

func (g *GameStateGameplay) updatePlayerMovementMode(newPosition geometry.Point) {
    timeNow := time.Now()
    msSinceLastMove := timeNow.Sub(g.lastPlayerMovementAt).Milliseconds()
    g.lastPlayerMovementAt = timeNow

    player := g.engine.GetGame().GetMap().Player

    // Sneaking is only ever set via an explicit toggle (StartSneaking command).
    // Here we only decide between Running and Walking based on step cadence.
    if player.MovementMode == core.MovementModeSneaking {
        return
    }
    if msSinceLastMove < int64(core.WalkStepDelayMs) {
        player.MovementMode = core.MovementModeRunning
        g.engine.Schedule(1, func() { g.downgradeMovementMode(newPosition) })
    } else {
        player.MovementMode = core.MovementModeWalking
    }
}

func (g *GameStateGameplay) downgradeMovementMode(position geometry.Point) {
    player := g.engine.GetGame().GetMap().Player
    if player.Pos() != position || player.MovementMode != core.MovementModeRunning {
        return
    }
    defer g.UpdateStatusLine()
    player.MovementMode = core.MovementModeWalking
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
        if noPointerCmd && !g.lookModeActive {
            g.clearTooltip()
        }
    }

    g.midLabel.Update(input)

    g.scrollCameraForScope()
    g.updateScripts()
    g.updateDialogue()
    g.updateFlashlight()
    actions := game.GetActions()
    actions.Update()

    g.assassinationTargets = assassinationTargets(g.engine)

    currentMap := game.GetMap()
    if currentMap.DynamicLightsChanged && game.GetConfig().LightSources {
        currentMap.UpdateDynamicLights()
    }

    // Advance in-game time: one real second = one in-game minute.
    g.timeAccumulator++
    if g.timeAccumulator >= utils.SecondsToTicks(1) {
        g.timeAccumulator = 0
        g.tickTimeProgression()
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
    case "F4":
        g.adjustTimeOfDay(-30 * time.Minute)
    case "F5":
        g.adjustTimeOfDay(30 * time.Minute)
    case "t":
        g.openWaitMenu()
    }
}

// adjustTimeOfDay shifts the map's time of day by delta, then immediately
// recalculates ambient and baked lighting so the change is visible in-game.
func (g *GameStateGameplay) adjustTimeOfDay(delta time.Duration) {
    currentMap := g.engine.GetGame().GetMap()
    currentMap.TimeOfDay = currentMap.TimeOfDay.Add(delta)
    currentMap.SetAmbientLight(common.GetAmbientLightFromDayTime(currentMap.TimeOfDay).ToRGB())
    currentMap.UpdateBakedLights()
    currentMap.UpdateDynamicLights()
    g.isDirty = true
    g.Print(fmt.Sprintf("[debug] Time of day: %s", currentMap.TimeOfDay.Format("15:04")))
}

// tickTimeProgression is called once per real second from Update.
// It advances the in-game clock by one minute and updates ambient lighting so
// the environment gradually changes as the day/night cycle progresses.
func (g *GameStateGameplay) tickTimeProgression() {
    currentMap := g.engine.GetGame().GetMap()
    currentMap.TimeOfDay = currentMap.TimeOfDay.Add(time.Minute)
    currentMap.SetAmbientLight(common.GetAmbientLightFromDayTime(currentMap.TimeOfDay).ToRGB())
    g.isDirty = true
    g.UpdateStatusLine()
}

func (g *GameStateGameplay) clearTooltip() {
    userInterface := g.engine.GetUI()
    userInterface.ClearTooltip()
}

func (g *GameStateGameplay) scrollCameraForScope() {
    game := g.engine.GetGame()
    player := game.GetMap().Player
    if player == nil || player.FovMode != gridmap.FoVModeScoped {
        return
    }
    if g.aimScrollTimer == 0 {
        if g.padAimActive {
            // Pad: aim position is already in world space.
            g.ensureWorldPosInView(g.padAimPos.ToPointRounded(), 4)
        } else {
            // Mouse: aim position was already set by the last mouse-move event.
            // Just ensure the aim target stays in view as the camera scrolls.
            g.ensureWorldPosInView(game.GetCamera().ScreenToWorld(g.MousePositionOnScreen), 4)
        }
        g.aimScrollTimer = 10
    } else {
        g.aimScrollTimer--
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
    g.playerMovementFromInput(pdelta)

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
    g.playerPeekingFromInput(pdelta)
}

func (g *GameStateGameplay) playerPeekingFromInput(pdelta geometry.Point) {
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    player := currentMap.Player
    player.InteractionShift = pdelta
    if (pdelta.X != 0 || pdelta.Y != 0) && player.CanMove() && g.canPeekFrom(player.Pos().Add(pdelta)) {
        player.FoVShift = pdelta
    } else {
        player.FoVShift = geometry.PointZero
    }
    currentMap.UpdateFieldOfView(player)
    g.UpdateHUD()
}

func (g *GameStateGameplay) canPeekFrom(worldPos geometry.Point) bool {
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    return currentMap.IsTileWalkable(worldPos)
}

func (g *GameStateGameplay) playerMovementFromInput(pdelta geometry.Point) {
    m := g.engine.GetGame()
    currentMap := m.GetMap()
    player := currentMap.Player
    if (pdelta.X != 0 || pdelta.Y != 0) && player.CanMove() {
        oldPos := player.Pos()
        np := oldPos.Add(pdelta)
        if currentMap.CurrentlyPassableForActor(player)(np) {
            player.LookDirection = geometry.DirectionVectorToAngleInDegrees(pdelta)
            g.resetPlayerState()
            m.MoveActor(player, np)
            g.playerMoved(oldPos, np)
            g.aimInLookDirection()
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
        defaultCellStyle = defaultCellStyle.WithBg(core.CurrentTheme.BloodBackground)
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
                style = m.ApplyLighting(p, c, style)
                style = style.Desaturate()
                style = style.Darken(0.5)
            }
            posRelativeToPlayer := p.Sub(player.Pos())
            if action, isActionAtPos := g.ActionMap[posRelativeToPlayer]; isActionAtPos {
                _, actionStyle := action.Description(g.engine, player, p)
                style = style.WithBg(actionStyle.Background)
            }
            if c.Actor != nil {
                if _, ok := g.assassinationTargets[*c.Actor]; ok {
                    style = style.WithBg(core.CurrentTheme.IllegalActionBackground)
                }
            }
        }
        drawPos := m.GetCamera().WorldToScreen(p)
        con.SetSquare(drawPos, common.Cell{Rune: icon, Style: style})
    })

    emptyPoint := geometry.Point{}
    if m.GetMap().Player.FoVShift != emptyPoint {
        fovSourcePos := m.GetMap().Player.Pos().Add(m.GetMap().Player.FoVShift)
        screenFovSource := m.GetCamera().WorldToScreen(fovSourcePos)
        currentCell := con.AtSquare(screenFovSource)
        con.SetSquare(screenFovSource, currentCell.WithStyle(currentCell.Style.WithBg(core.CurrentTheme.LOSBackground)))
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

    //g.drawContextHUD(con)

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
            itemStyle = itemStyle.WithFg(core.CurrentTheme.DangerForeground)
        } else if player.EquippedItem.Type == core.ItemTypeEmeticPoison {
            itemStyle = itemStyle.WithFg(core.CurrentTheme.EmeticPoisonForeground)
        } else if player.EquippedItem.Type == core.ItemTypeLethalPoison {
            itemStyle = itemStyle.WithFg(core.CurrentTheme.LethalPoisonForeground)
        }
        if player.EquippedItem.Uses > 0 {
            itemSymbol += fmt.Sprintf(" (%d)", player.EquippedItem.Uses)
        }
    }
    if player.Health <= 1 {
        hpStyle = hpStyle.WithFg(core.CurrentTheme.DangerForeground)
    } else if player.Health == 2 {
        hpStyle = hpStyle.WithFg(core.CurrentTheme.HUDWarningForeground)
    } else {
        hpStyle = hpStyle.WithFg(core.CurrentTheme.HUDGoodForeground)
    }

    clothingStyle = clothingStyle.WithFg(player.Clothes.FgColor()) //.WithBg(player.Clothes.Color)

    healthString := createHealthBar(player.Health)

    zoneInformation := currentMap.ZoneAt(player.Pos()).Name
    if currentMap.IsInHostileZone(player) {
        zoneInformation = "hostile area"
        detectionStyle = detectionStyle.WithBg(core.CurrentTheme.HUDDangerBackground).WithFg(common.Black)
    } else if currentMap.IsTrespassing(player) {
        zoneInformation = "trespassing"
        detectionStyle = detectionStyle.WithBg(core.CurrentTheme.HUDWarningBackground).WithFg(common.Black)
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

    redStyle := common.DefaultStyle.WithBg(core.CurrentTheme.HUDDangerBackground)
    greenStyle := common.DefaultStyle.WithBg(core.CurrentTheme.HUDGoodBackground)
    //g.topLabel.SetStyledText(core.Text(fmt.Sprintf("@c%s@N | @h%s@N | %s | @i%s@N | @d%s@N | %s", player.Clothes.Name, healthString, string(player.MovementMode), itemSymbol, zoneInformation, challengeInformation)).
    g.topLabel.SetStyledText(core.Text(fmt.Sprintf("@c%s@N %s@i%s@N | @h%s@N | @d%s@N | %s | %s", player.Clothes.Name, string(player.MovementMode), itemSymbol, healthString, zoneInformation, challengeInformation, currentMap.TimeOfDay.Format("15:04"))).
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
        con.SetSquare(p, c.WithStyle(c.Style.WithBg(core.CurrentTheme.MapBackgroundLight)))
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
        player.MovementMode = core.MovementModeSneaking
        defer g.UpdateStatusLine()
    case core.StopSneaking:
        input.SetMovementDelayForWalkingAndRunning()
        player.MovementMode = core.MovementModeWalking
        defer g.UpdateStatusLine()
    case core.NextItem:
        g.EquipNextInventoryItem(player)
    case core.OpenInventory:
        var openInventory func()
        openInventory = func() {
            userInterface.OpenItemRingMenu(player.EquippedItem, player.Inventory.Items, func(item *core.Item) {
                if player.EquippedItem != nil && player.EquippedItem != item && player.EquippedItem.IsBig {
                    g.engine.GetGame().DropEquippedItem(player)
                }
                player.EquippedItem = item
                g.UpdateHUD()
                g.midLabel.SetText("")
                g.midLabel.SetDirty()
            }, func() {
                g.UpdateHUD()
                g.midLabel.SetText("")
                g.midLabel.SetDirty()
            }, func(itemToDrop *core.Item) {
                if player.EquippedItem == itemToDrop {
                    player.EquippedItem = nil
                }
                model.DropFromInventory(player, []*core.Item{itemToDrop})
                g.UpdateHUD()
                g.midLabel.SetText("")
                g.midLabel.SetDirty()
                if len(player.Inventory.Items) > 0 {
                    openInventory()
                }
            })
        }
        openInventory()
    case core.DropItem:
        model.DropEquippedItem(player)
        g.UpdateHUD()
    case core.PickUpItem:
        model.PickUpItem(player)
        g.UpdateHUD()
    case core.HolsterItem:
        player.HolsterItem()
        g.UpdateHUD()
    case core.UseRangedItem:
        if len(g.TargetLoS) == 0 {
            g.aimInLookDirection()
        }
        g.UseRangedItemInLoS()
        g.UpdateHUD()
    case core.BeginMouseAiming:
        g.BeginMouseAiming()
    case core.Assassinate:
        g.executeAssassination()
    // ── Gamepad plain commands – use current peek tile as direction ──────────
    case core.DiveTackle:
        g.playerDiveTackle()
    case core.UseItem:
        g.playerUseItem()
    case core.ContextAction:
        g.contextAction()
    case core.Cancel:
        g.OpenPauseMenu()
    case core.StopAiming:
        if g.padAimActive {
            g.resetPlayerState()
            g.ensureWorldPosInView(player.Pos(), 4)
            model.GetMap().UpdateFieldOfView(player)
        }
    case core.ToggleLookMode:
        g.toggleLookMode()
    }
}

func (g *GameStateGameplay) executeAssassination() {
    if len(g.assassinationTargets) == 0 {
        return
    }
    game := g.engine.GetGame()
    player := game.GetMap().Player

    var weapon *core.Item
    if player.EquippedItem != nil && player.EquippedItem.HasMeleePiercingDamage() {
        weapon = player.EquippedItem
    } else {
        for _, item := range player.Inventory.Items {
            if item.HasMeleePiercingDamage() {
                weapon = item
                break
            }
        }
    }

    // Collect targets and clear the field so no second trigger can fire.
    targets := make([]*core.Actor, 0, len(g.assassinationTargets))
    for target := range g.assassinationTargets {
        targets = append(targets, target)
    }
    g.assassinationTargets = nil

    // Choose the icon: weapon icon, fallback to dagger rune.
    icon := rune('†')
    if weapon != nil {
        icon = weapon.Icon()
    }

    // Lock player and victims for the duration of the animation.
    aic := g.engine.GetAI()
    animationDone := false
    until := func() bool { return animationDone }
    aic.SetEngaged(player, core.ActorStatusEngagedIllegal, until)
    for _, target := range targets {
        aic.SetEngaged(target, core.ActorStatusVictimOfEngagement, until)
    }

    capturedWeapon := weapon
    finished := func() {
        animationDone = true
        for _, target := range targets {
            game.IllegalActionAt(target.Pos(), core.ObservationPersonAttacked)
            game.Kill(target, core.CauseOfDeath{
                Description: core.CoDStabbed,
                Source:      core.EffectSource{Actor: player, Item: capturedWeapon},
            })
        }
    }

    g.engine.GetAnimator().AssassinationAnimation(targets, icon, finished)
}

func (g *GameStateGameplay) handleDirectionalInput(directionalCommand core.DirectionalGameCommand) {
    intDirection := toIntDirection(directionalCommand.XAxis, directionalCommand.YAxis)
    switch directionalCommand.Command {
    case core.MovementDirection:
        g.playerMovementFromInput(toIntDirection(directionalCommand.XAxis, directionalCommand.YAxis))
    case core.PeekingDirection:
        if g.lookModeActive {
            g.moveLookCursor(directionalCommand.XAxis, directionalCommand.YAxis)
        } else {
            g.playerPeekingFromInput(intDirection)
            g.setAimFromPeekDirection(intDirection)
        }
    case core.AimingDirection:
        g.AdjustPlayerAimFromPad(directionalCommand.XAxis, directionalCommand.YAxis)
    }
    return
}

// toIntDirection maps an analog stick vector to one of 9 tile directions
// {-1,0,1}² by dividing the full 360° into eight equal 45° sectors.
//
// Sector boundaries sit at ±22.5° from each cardinal and diagonal axis.
// tan(22.5°) = √2 − 1 ≈ 0.4142, its reciprocal tan(67.5°) = √2 + 1 ≈ 2.4142.
//
//   ratio = |yAxis| / |xAxis|
//   ratio < tan(22.5°)  → cardinal (horizontal wins)
//   ratio > tan(67.5°)  → cardinal (vertical wins)
//   otherwise           → diagonal
func toIntDirection(xAxis float64, yAxis float64) geometry.Point {
    const tanOf22_5 = 0.41421356237 // tan(22.5°) = √2 − 1
    const tanOf67_5 = 2.41421356237 // tan(67.5°) = √2 + 1  (= 1/tanOf22_5)

    absX := math.Abs(xAxis)
    absY := math.Abs(yAxis)

    sx, sy := 0, 0
    if xAxis > 0 {
        sx = 1
    } else if xAxis < 0 {
        sx = -1
    }
    if yAxis > 0 {
        sy = 1
    } else if yAxis < 0 {
        sy = -1
    }

    if absX == 0 && absY == 0 {
        return geometry.Point{}
    }
    if absX == 0 {
        return geometry.Point{X: 0, Y: sy}
    }
    if absY == 0 {
        return geometry.Point{X: sx, Y: 0}
    }

    ratio := absY / absX
    switch {
    case ratio < tanOf22_5:
        return geometry.Point{X: sx, Y: 0} // cardinal E / W
    case ratio > tanOf67_5:
        return geometry.Point{X: 0, Y: sy} // cardinal N / S
    default:
        return geometry.Point{X: sx, Y: sy} // diagonal
    }
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

func (g *GameStateGameplay) contextAction() {
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    player := currentMap.Player

    if !player.CanUseActions() {
        return
    }

    targetPos := player.InteractSource()
    direction := player.InteractionShift

    if player.MovementMode == core.MovementModeSneaking && targetPos != player.Pos() && currentMap.IsActorAt(targetPos) {
        g.tryPickpocket(currentMap.ActorAt(targetPos))
        return
    }

    if directionalAction, valid := g.ActionMap[direction]; valid {
        directionalAction.Action(g.engine, player, player.Pos().Add(direction))
        g.UpdateHUD()
        return
    }

    // Optimisation: when exactly one context action exists in the surroundings
    // and no explicit direction has been selected (or the selected direction has
    // no action), execute that single action immediately.
    if len(g.ActionMap) == 1 {
        for dir, action := range g.ActionMap {
            action.Action(g.engine, player, player.Pos().Add(dir))
            g.UpdateHUD()
        }
    }
}

// playerDiveTackle performs the Dive & Tackle action in the given direction.
//
// • Both tiles clear → player jumps/dives two tiles instantly.
// • NPC(s) on either tile → they are prodded in the direction (tackle);
//   player moves to the first terrain-passable tile.
func (g *GameStateGameplay) playerDiveTackle() {
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    player := currentMap.Player

    direction := player.FoVShift
    if direction.X == 0 && direction.Y == 0 {
        return
    }

    if !player.CanMove() {
        return
    }

    tile1 := player.Pos().Add(direction)
    tile2 := tile1.Add(direction)

    if !currentMap.Contains(tile1) {
        return
    }

    tile1HasNPC := currentMap.IsActorAt(tile1)
    tile2HasNPC := currentMap.Contains(tile2) && currentMap.IsActorAt(tile2)

    // Prod any NPCs in the direction first (tackle).
    if tile1HasNPC {
        game.TryPushActorInDirection(currentMap.ActorAt(tile1), direction)
    }
    if tile2HasNPC {
        game.TryPushActorInDirection(currentMap.ActorAt(tile2), direction)
    }

    // Determine how far the player can travel.
    passable := currentMap.CurrentlyPassableForActor(player)
    tile1Clear := passable(tile1) && !currentMap.IsActorAt(tile1)
    tile2Clear := currentMap.Contains(tile2) && passable(tile2) && !currentMap.IsActorAt(tile2)

    var diveTarget geometry.Point
    switch {
    case tile1Clear && tile2Clear:
        diveTarget = tile2 // Full two-tile dive
    case tile1Clear:
        diveTarget = tile1 // One-tile dive / tackle-step
    default:
        return // Completely blocked
    }

    oldPos := player.Pos()
    player.LookDirection = geometry.DirectionVectorToAngleInDegrees(direction)
    player.MovementMode = core.MovementModeRunning
    g.resetPlayerState()
    game.MoveActor(player, diveTarget)
    g.playerMoved(oldPos, diveTarget)
}

// playerUseItem handles the Square button:
// use equipped item (or bare-hand melee) at the tile pointed to by the
// right-analog stick.  Zero direction → self-apply.
func (g *GameStateGameplay) playerUseItem() {
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    player := currentMap.Player
    direction := player.FoVShift
    actions := game.GetActions()

    if !player.CanUseActions() {
        return
    }

    targetPos := player.Pos()
    if direction.X != 0 || direction.Y != 0 {
        targetPos = player.Pos().Add(direction)
    }

    dist := geometry.DistanceManhattan(player.Pos(), targetPos)
    isSelf := dist == 0
    isMelee := dist == 1

    if player.EquippedItem == nil {
        if isMelee && currentMap.IsActorAt(targetPos) {
            g.contextAction()
        }
        return
    }

    if player.EquippedItem.OnCooldown ||
        !player.EquippedItem.HasUsesLeft() ||
        !player.CanUseItems() {
        return
    }

    if player.EquippedItem.InsteadOfUse != nil {
        player.EquippedItem.DecreaseUsesLeft()
        player.EquippedItem.InsteadOfUse()
        return
    }

    if isSelf && player.EquippedItem.Type.CanSelfActivate() {
        player.EquippedItem.DecreaseUsesLeft()
        actions.UseEquippedItemOnSelf(player)
    } else if isMelee && player.EquippedItem.Type.HasMeleeAction() {
        if player.EquippedItem.Type.MeleeDecreaseUses() {
            player.EquippedItem.DecreaseUsesLeft()
        }
        actions.UseEquippedItemForMelee(player, targetPos)
    }

    if player.EquippedItem == nil {
        g.resetPlayerState()
    }
    g.UpdateHUD()
}

// tryPickpocket attempts to steal one random item from the target NPC.
//
// Conditions (all must be true):
//  1. Player is exactly in the tile directly behind the NPC (opposite of NPC's look direction).
//  2. NPC is alive and in an unaware state (not combat, not investigating, not searching, not snitching).
//  3. NPC has at least one non-equipped item in their inventory.
func (g *GameStateGameplay) tryPickpocket(target *core.Actor) {
    game := g.engine.GetGame()
    currentMap := game.GetMap()
    player := currentMap.Player

    // Guard: target must be unaware.

    if target.IsInCombat() || target.IsInvestigating() ||
        target.Status == core.ActorStatusSearching ||
        target.Status == core.ActorStatusSnitching ||
        target.Status == core.ActorStatusPanic {
        g.Print("Target is too alert to pickpocket.")
        return
    }

    // Guard: player must be directly behind the NPC — unless the target is
    // unconscious, in which case any adjacent position works.
    if !target.IsDowned() {
        // "Behind" = the tile in the OPPOSITE direction of the NPC's look direction.
        npcLookVec := geometry.Point{
            X: int(math.Round(math.Cos(target.LookDirection * math.Pi / 180))),
            Y: int(math.Round(math.Sin(target.LookDirection * math.Pi / 180))),
        }
        // Clamp to -1/0/1 to get the dominant 8-directional vector.
        clamp := func(v int) int {
            if v > 0 {
                return 1
            }
            if v < 0 {
                return -1
            }
            return 0
        }
        npcFacing := geometry.Point{X: clamp(npcLookVec.X), Y: clamp(npcLookVec.Y)}
        behindNPC := target.Pos().Sub(npcFacing) // tile directly behind NPC = NPC pos - facing

        if player.Pos() != behindNPC {
            g.Print("Must be directly behind the target to pickpocket.")
            return
        }
    }

    // Collect pickpocketable items (inventory minus equipped item).
    stealable := make([]*core.Item, 0, len(target.Inventory.Items))
    for _, item := range target.Inventory.Items {
        if item != target.EquippedItem {
            stealable = append(stealable, item)
        }
    }
    if len(stealable) == 0 {
        g.Print("Target has nothing to steal.")
        return
    }

    // Pick a random item.
    chosen := stealable[rand.Intn(len(stealable))]
    target.Inventory.RemoveItem(chosen)
    chosen.HeldBy = player
    player.Inventory.AddItem(chosen)
    g.Print(fmt.Sprintf("Pickpocketed: %s", chosen.Name))
    g.UpdateHUD()
}

func (g *GameStateGameplay) printTileContentsMessage(position geometry.Point) {
    eng := g.engine
    m := eng.GetGame()
    tileType := m.GetMap().CellAt(position).TileType
    if m.GetMap().IsItemAt(position) {
        itemHere := m.GetMap().ItemAt(position)
        if !itemHere.Buried {
            g.Print(fmt.Sprintf("There is %s here.", itemHere.Name))
        }
    } else if tileType.Special != gridmap.SpecialTileDefaultFloor {
        tileName := tileType.Description()
        g.Print(fmt.Sprintf("There is %s here.", tileName))
    }
}

func (g *GameStateGameplay) updateMouseOver() {
    g.showTooltipForWorldPos(g.MousePositionInWorld)
}

// showTooltipForWorldPos inspects the given world position and shows the
// appropriate tooltip (actor, item, object or special tile). Used by both
// mouse hover and gamepad peek-look.
func (g *GameStateGameplay) showTooltipForWorldPos(worldPos geometry.Point) {
    currentMap := g.engine.GetGame().GetMap()
    userInterface := g.engine.GetUI()

    g.isDirty = true

    if !currentMap.Contains(worldPos) || !currentMap.Player.CanSee(worldPos) {
        if userInterface.TooltipShown() {
            g.clearTooltip()
        }
        return
    }
    mapCell := currentMap.CellAt(worldPos)
    if currentMap.IsActorAt(worldPos) {
        actorAtPos := currentMap.ActorAt(worldPos)
        g.showActorTooltip(actorAtPos)
    } else if currentMap.IsDownedActorAt(worldPos) {
        actorAtPos := currentMap.DownedActorAt(worldPos)
        g.showActorTooltip(actorAtPos)
    } else if currentMap.IsItemAt(worldPos) {
        itemHere := currentMap.ItemAt(worldPos)
        if !itemHere.Buried {
            g.showItemTooltip(itemHere)
        }
    } else if currentMap.IsObjectAt(worldPos) {
        objectHere := currentMap.ObjectAt(worldPos)
        g.showObjectTooltip(objectHere)
    } else if mapCell.TileType.Special != gridmap.SpecialTileDefaultFloor {
        g.showCellTooltip(worldPos, mapCell)
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
    tooltipFontColor := core.CurrentTheme.TooltipForeground
    if person.Type == core.ActorTypeTarget {
        tooltipFontColor = common.NewRGBColorFromBytes(255, 51, 51)
    }
    styledText := core.Text(infoString).WithStyle(common.Style{Foreground: tooltipFontColor, Background: core.CurrentTheme.TooltipBackground})
    screenPos := g.engine.GetGame().GetCamera().WorldToScreen(person.Pos())
    g.showTooltipAt(screenPos, styledText)
}

func (g *GameStateGameplay) showItemTooltip(item *core.Item) {
    styledText := core.Text(item.Description()).WithStyle(common.Style{Foreground: core.CurrentTheme.TooltipForeground, Background: core.CurrentTheme.TooltipBackground})
    screenPos := g.engine.GetGame().GetCamera().WorldToScreen(item.Pos())
    g.showTooltipAt(screenPos, styledText)
}

func (g *GameStateGameplay) showObjectTooltip(object services.Object) {
    styledText := core.Text(object.Description()).WithStyle(common.Style{Foreground: core.CurrentTheme.TooltipForeground, Background: core.CurrentTheme.TooltipBackground})
    screenPos := g.engine.GetGame().GetCamera().WorldToScreen(object.Pos())
    g.showTooltipAt(screenPos, styledText)
}
func (g *GameStateGameplay) showCellTooltip(pos geometry.Point, cell gridmap.MapCell[*core.Actor, *core.Item, services.Object]) {
    infoString := fmt.Sprintf("%s", cell.TileType.Description())
    styledText := core.Text(infoString).WithStyle(common.Style{Foreground: core.CurrentTheme.TooltipForeground, Background: core.CurrentTheme.TooltipBackground})
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

func (g *GameStateGameplay) updateFlashlight() {
    game := g.engine.GetGame()
    if !game.GetConfig().LightSources {
        return
    }
    currentMap := game.GetMap()
    player := currentMap.Player

    flashlightActive := player.EquippedItem != nil &&
        player.EquippedItem.Type == core.ItemTypeFlashlight &&
        player.FovMode == gridmap.FoVModeScoped

    if !flashlightActive {
        // Remove the light if it was previously placed.
        if currentMap.IsDynamicLightSource(g.flashlightLight.Pos) {
            currentMap.RemoveDynamicLightAt(g.flashlightLight.Pos)
        }
        return
    }

    if len(g.TargetLoS) == 0 {
        return
    }

    // Reuse TargetLoS from AdjustPlayerAim – same path the sniper scope uses.
    aimPoint := g.TargetLoS[len(g.TargetLoS)-1]

    if currentMap.IsDynamicLightSource(g.flashlightLight.Pos) {
        if aimPoint != g.flashlightLight.Pos {
            currentMap.MoveLightSource(g.flashlightLight, aimPoint)
        }
    } else {
        g.flashlightLight.Pos = aimPoint
        currentMap.AddDynamicLightSource(aimPoint, g.flashlightLight)
        currentMap.UpdateDynamicLights()
    }
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
