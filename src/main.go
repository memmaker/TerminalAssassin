package main

import (
    "embed"
    "encoding/gob"
    "errors"
    "image"
    "image/png"
    "math"
    "os"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/memmaker/terminal-assassin/audio"
    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/console"
    "github.com/memmaker/terminal-assassin/game"
    "github.com/memmaker/terminal-assassin/game/ai"
    "github.com/memmaker/terminal-assassin/game/objects"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/game/stimuli"
    "github.com/memmaker/terminal-assassin/mapset"
    "github.com/memmaker/terminal-assassin/ui"
    "log"
)

//go:embed datafiles
var embeddedFS embed.FS
var WEB_MODE = false

type ConsoleEngine struct {
    // Config
    Config console.GridConfig
    // Input
    Input *InputState
    // Audio
    Audio *audio.AudioPlayer
    // Console
    Console *console.Console
    // Model
    Model *game.Model
    // Files
    Files *Files
    // AI
    AIController *ai.AIController
    // External Data
    ExternalData *services.ExternalData
    // Career
    Career *services.CareerData
    // Animations
    Animator *game.Animator
    // UI
    UserInterface *ui.Manager
    // Creating complex items with engine dependencies
    ItemFactory   *services.ItemFactory
    ObjectFactory *objects.ObjectFactory

    graphicsConfig GraphicsConfig

    deviceDPIScale float64
    wantsToQuit    bool

    WorldTicks                  uint64
    scheduledCalls              map[uint64][]func()
    scheduledCallsWithCondition []ScheduledCallWithCondition
    subscribers                 []services.Subscriber

    // TimeFactor scales world time for all AI-controlled actors and objects.
    // 1.0 = normal, >1.0 = faster, <1.0 = slower, 0 = frozen.
    TimeFactor float64

    pendingScreenshotPath string
}

func (g *ConsoleEngine) GetObjectFactory() services.ObjectFactoryInterface {
    return g.ObjectFactory
}

func (g *ConsoleEngine) PublishEvent(event services.GameEvent) {
    for i := len(g.subscribers) - 1; i >= 0; i-- {
        subscriber := g.subscribers[i]
        if !subscriber.ReceiveMoreAfter(event) {
            g.subscribers = append(g.subscribers[:i], g.subscribers[i+1:]...)
        }
    }
}
func (g *ConsoleEngine) SubscribeToEvents(eventFilter services.Subscriber) {
    g.subscribers = append(g.subscribers, eventFilter)
}
func (g *ConsoleEngine) GetItemFactory() *services.ItemFactory {
    return g.ItemFactory
}

func (g *ConsoleEngine) GetAvailableTextFonts() []string {
    return g.Console.GetAvailableTextFonts()
}

func (g *ConsoleEngine) SetTextFont(fontName string) {
    g.Console.SetHalfWidthFont(fontName)
}

func (g *ConsoleEngine) SetTileFont(fontName string) {
    g.Console.SetSquareFont(fontName)
}

func (g *ConsoleEngine) PushState(newGameState services.GameState) {
    g.Model.PushState(newGameState)
}

func (g *ConsoleEngine) QuitGame() {
    SaveGraphicsConfig(g.graphicsConfig)
    g.wantsToQuit = true
}

// Scenarios:
// 1. A model is open
//    -> Everyone gets a draw call, but only the model gets an update call
// 2. Multiple UI components are open
//    -> One of these needs focus (how to switch focus?)

func (g *ConsoleEngine) Update() error {
    if g.wantsToQuit {
        return ebiten.Termination
    }
    // These are global and don't need any focus..
    //g.checkRecorderControls()

    g.Audio.Update()
    // do we need this? we do
    g.Input.Update()

    g.UserInterface.Update(g.Input)

    if !g.UserInterface.IsBlocking() {
        g.UpdateScheduledCalls()
        g.WorldTicks++
    }

    g.UserInterface.Draw(g.Console)

    g.Animator.Update()
    g.Animator.Draw(g.Console)

    // Compute delta and update previous grid state
    g.Console.Flush()
    return nil
}
func (g *ConsoleEngine) Draw(screen *ebiten.Image) {
    g.Console.Draw(screen)
    if g.pendingScreenshotPath != "" {
        savePath := g.pendingScreenshotPath
        g.pendingScreenshotPath = ""
        saveEbitenImageAsPNG(screen, savePath)
    }
}

// RequestScreenshot queues a screenshot of the next rendered frame to be
// saved as a PNG at the given file path.
func (g *ConsoleEngine) RequestScreenshot(filePath string) {
    g.pendingScreenshotPath = filePath
}

func (g *ConsoleEngine) GetTimeFactor() float64 {
    return g.TimeFactor
}

func (g *ConsoleEngine) SetTimeFactor(factor float64) {
    if factor < 0 {
        factor = 0
    }
    g.TimeFactor = factor
}

// saveEbitenImageAsPNG reads pixel data from an ebiten image and writes it
// to disk as a PNG file.
func saveEbitenImageAsPNG(img *ebiten.Image, filePath string) {
    bounds := img.Bounds()
    w, h := bounds.Dx(), bounds.Dy()
    pixels := make([]byte, w*h*4)
    img.ReadPixels(pixels)
    rgba := &image.RGBA{
        Pix:    pixels,
        Stride: w * 4,
        Rect:   image.Rect(0, 0, w, h),
    }
    f, err := os.Create(filePath)
    if err != nil {
        println("Camera: failed to create screenshot file:", err.Error())
        return
    }
    defer f.Close()
    if err := png.Encode(f, rgba); err != nil {
        println("Camera: failed to encode screenshot:", err.Error())
    }
    println("Camera: screenshot saved to", filePath)
}

/*
Ctrl + R: Start/Stop recording
Ctrl + S: Play recording
*/

func main() {
    println("Starting game...")
    if WEB_MODE {
        println("Web mode enabled")
    }
    gameTitle := "Terminal Assassin"
    textFontDirectory := "datafiles/textfonts"
    tileFontDirectory := "datafiles/tilefonts"
    squareFontName := "Square-H"
    halfWidthFontName := "Px437 EagleSpCGA Alt2-2y"

    config := console.OptimalGridConfig(50, 10)
    graphicsConfig := LoadGraphicsConfig(config.TileSize)
    gameConfig := &services.GameConfig{
        ActorDefaultHealth: 3,
        CampaignDirectory:  "datafiles/campaigns",
        GridConfig:         config,
        WebMode:            WEB_MODE,
        MusicStreaming:     true,
        Audio:              true,
        LightSources:       true,
        ShowHints:          true,
        Fullscreen:         graphicsConfig.Fullscreen,
    }

    con := console.NewConsole(config)
    con.LoadEmbeddedFonts(tileFontDirectory, textFontDirectory, squareFontName, halfWidthFontName, embeddedFS)
    files := &Files{fs: embeddedFS}
    externalData := services.NewExternalDataFromDisk(files)
    consoleGame := &ConsoleEngine{
        Config:         config,
        graphicsConfig: graphicsConfig,
        Console:        con,
        Input:          NewInput(config),
        Model:          game.NewModel(gameConfig),
        Files:          files,
        ExternalData:   externalData,
        Career:         game.NewCareerFromFile(externalData),
        scheduledCalls: map[uint64][]func(){},
        TimeFactor:     1.0,
    }
    ebiten.SetWindowTitle(gameTitle)
    ebiten.SetWindowSize(graphicsConfig.WindowedWidth, graphicsConfig.WindowedHeight)
    if graphicsConfig.Fullscreen {
        ebiten.SetFullscreen(true)
    }
    ebiten.SetScreenClearedEveryFrame(false)
    consoleGame.Init()
    if err := ebiten.RunGameWithOptions(consoleGame, &ebiten.RunGameOptions{
        GraphicsLibrary: ebiten.GraphicsLibraryOpenGL,
    }); err != nil && !errors.Is(err, ebiten.Termination) {
        log.Fatal(err)
    }
}

func (g *ConsoleEngine) Init() {
    g.deviceDPIScale = ebiten.DeviceScaleFactor()
    g.AIController = ai.NewAIController(g)
    g.Audio = audio.NewAudioPlayer(g)
    g.Animator = game.NewAnimator(g)
    g.UserInterface = ui.NewManager(g)
    g.ItemFactory = services.NewFactory(g)
    g.ObjectFactory = objects.NewFactory(g)
    g.Model.Init(g)
}

type ScheduledCallWithCondition struct {
    Condition func() bool
    Call      func()
}

func (g *ConsoleEngine) ScheduleWhen(condition func() bool, functionCall func()) {
    g.scheduledCallsWithCondition = append(g.scheduledCallsWithCondition, ScheduledCallWithCondition{
        Condition: condition,
        Call:      functionCall,
    })
}
func (g *ConsoleEngine) Schedule(relativeSeconds float64, call func()) {
    relativeTicks := uint64(relativeSeconds * float64(ebiten.TPS()))
    if relativeTicks == 0 {
        relativeTicks = 1
    }
    g.ScheduleAbs(g.WorldTicks+relativeTicks, call)
}
func (g *ConsoleEngine) ScheduleInTicks(relativeTicks uint64, call func()) {
    if relativeTicks == 0 {
        relativeTicks = 1
    }
    g.ScheduleAbs(g.WorldTicks+relativeTicks, call)
}

func (g *ConsoleEngine) ScheduleAbs(absoluteWorldTick uint64, call func()) {
    if _, ok := g.scheduledCalls[absoluteWorldTick]; !ok {
        g.scheduledCalls[absoluteWorldTick] = []func(){}
    }
    g.scheduledCalls[absoluteWorldTick] = append(g.scheduledCalls[absoluteWorldTick], call)
}
func (g *ConsoleEngine) UpdateScheduledCalls() {
    if calls, forThisTick := g.scheduledCalls[g.WorldTicks]; forThisTick {
        for _, call := range calls {
            call()
        }
        delete(g.scheduledCalls, g.WorldTicks)
    }

    for i := len(g.scheduledCallsWithCondition) - 1; i >= 0; i-- {
        if g.scheduledCallsWithCondition[i].Condition() {
            g.scheduledCallsWithCondition[i].Call()
            g.scheduledCallsWithCondition = append(g.scheduledCallsWithCondition[:i], g.scheduledCallsWithCondition[i+1:]...)
        }
    }
}
func (g *ConsoleEngine) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
    panic("should use layoutf")
}

func (g *ConsoleEngine) LayoutF(outsideWidth, outsideHeight float64) (screenWidth, screenHeight float64) {
    scale := ebiten.Monitor().DeviceScaleFactor()
    g.deviceDPIScale = scale

    physicalW := outsideWidth * scale
    physicalH := outsideHeight * scale

    gameW := float64(g.Config.GridWidth * g.Config.TileSize)
    gameH := float64(g.Config.GridHeight * g.Config.TileSize)

    renderScale := math.Min(physicalW/gameW, physicalH/gameH)
    offsetX := (physicalW - gameW*renderScale) / 2
    offsetY := (physicalH - gameH*renderScale) / 2

    g.Console.SetRenderParams(renderScale, offsetX, offsetY)
    g.Input.SetRenderParams(
        float64(g.Config.TileSize)*renderScale,
        float64(g.Config.TileSize)*renderScale,
        offsetX, offsetY,
    )

    // Track windowed size so we can persist it on quit / fullscreen toggle.
    if !g.GetGame().GetConfig().Fullscreen {
        newW, newH := int(outsideWidth), int(outsideHeight)
        if newW > 0 && newH > 0 && (newW != g.graphicsConfig.WindowedWidth || newH != g.graphicsConfig.WindowedHeight) {
            g.graphicsConfig.WindowedWidth = newW
            g.graphicsConfig.WindowedHeight = newH
        }
    }

    return physicalW, physicalH
}

func (g *ConsoleEngine) SetFullscreen(enabled bool) {
    g.GetGame().GetConfig().Fullscreen = enabled
    g.graphicsConfig.Fullscreen = enabled
    ebiten.SetFullscreen(enabled)
    g.Console.ClearSurface() // fillScreenNext=true: Draw() will clear screen black
    g.Console.ClearConsole() // mark all cells dirty so the full frame is redrawn
    SaveGraphicsConfig(g.graphicsConfig)
}

func (g *ConsoleEngine) GetUI() services.UIInterface {
    return g.UserInterface
}

func (g *ConsoleEngine) GetGame() services.GameInterface {
    return g.Model
}

func (g *ConsoleEngine) GetFilesystem() embed.FS {
    return embeddedFS
}

func (g *ConsoleEngine) GetAudio() services.AudioInterface {
    return g.Audio
}

func (g *ConsoleEngine) GetInput() services.InputInterface {
    return g.Input
}

func (g *ConsoleEngine) ScreenGridWidth() int {
    return g.Config.GridWidth
}

func (g *ConsoleEngine) ScreenGridHeight() int {
    return g.Config.GridHeight
}

func (g *ConsoleEngine) MapWindowHeight() int {
    return g.Config.GridHeight - g.UserInterface.HUDHeight()
}

func (g *ConsoleEngine) MapWindowWidth() int {
    return g.Config.GridWidth
}

func (g *ConsoleEngine) GetCareer() *services.CareerData {
    return g.Career
}

func (g *ConsoleEngine) GetFiles() services.FileInterface {
    return g.Files
}

func (g *ConsoleEngine) GetData() services.DataInterface {
    return g.ExternalData
}

func (g *ConsoleEngine) GetAI() services.AIInterface {
    return g.AIController
}

func (g *ConsoleEngine) GetAnimator() services.AnimationInterface {
    return g.Animator
}
func (g *ConsoleEngine) CurrentTick() uint64 {
    return g.WorldTicks
}

func (g *ConsoleEngine) Reset() {
    g.ResetForGameplay()

    g.UserInterface.Reset()

    g.Model.ResetGameState()
    g.Model.ResetModel()
}

func (g *ConsoleEngine) ResetForGameplay() {
    g.AIController.Reset()
    g.Animator.Reset()
    g.WorldTicks = 0
    g.subscribers = make([]services.Subscriber, 0)
    g.scheduledCalls = map[uint64][]func(){}
    g.scheduledCallsWithCondition = make([]ScheduledCallWithCondition, 0)
}

func init() {
    gob.Register(&objects.Door{})
    gob.Register(&objects.Window{})
    gob.Register(&objects.CorpseContainer{})
    gob.Register(&objects.LiquidLeaker{})
    gob.Register(&objects.Safe{})
    gob.Register(&ai.FollowerMovement{})
    gob.Register(&ai.GuardMovement{})
    gob.Register(&ai.ScheduledMovement{})
    gob.Register(&stimuli.Stim{})
    gob.Register(&stimuli.StimEffect{})
    gob.Register(&game.ExitAction{})
    gob.Register(&game.OverflowAction{})
    gob.Register(&game.PoisonAction{})
    gob.Register(&game.ExposeElectricityAction{})
    gob.Register(&mapset.MapSet[string]{})
    gob.Register(&common.RGBAColor{})
    gob.Register(&common.HSVColor{})
    gob.Register(&services.FixedChallenge{})
    gob.Register(&services.CustomChallenge{})
    gob.Register(&services.DiskChallenge{})
}
