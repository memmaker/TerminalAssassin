package services

import (
	"embed"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/memmaker/terminal-assassin/mapset"
	"io/fs"
	"time"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

type ContextAction interface {
	Description(m Engine, person *core.Actor, actionAt geometry.Point) (rune, common.Style)
	Action(m Engine, person *core.Actor, actionAt geometry.Point)
	IsActionPossible(m Engine, person *core.Actor, actionAt geometry.Point) bool
}
type Object interface {
	Style(st common.Style) common.Style
	Action(m Engine, person *core.Actor)
	IsActionAllowed(m Engine, person *core.Actor) bool
	IsWalkable(*core.Actor) bool
	IsTransparent() bool
	IsPassableForProjectile() bool
	ApplyStimulus(m Engine, stim stimuli.Stimulus)
	Description() string

	Pos() geometry.Point
	Icon() rune
	SetPos(geometry.Point)
	EncodeAsString() string
	SetStyle(style common.Style)
	GetStyle() common.Style
}

type KeyBound interface {
	GetKey() string
	SetKey(key string)
}

type UIWidget interface {
	Update(input InputInterface)
	Draw(grid console.CellInterface)
	SetDirty()
}

type Focusable interface {
	Update(input InputInterface)
}
type UIInterface interface {
	OpenFixedWidthAutoCloseMenu(title string, items []MenuItem)
	OpenFixedWidthAutoCloseMenuWithCallback(title string, items []MenuItem, onClose func())
	OpenFixedWidthStackedMenu(title string, items []MenuItem)
	HUDHeight() int
	OpenMapsMenu(afterLoad func(*gridmap.GridMap[*core.Actor, *core.Item, Object]))
	OpenFancyMenu(menuItems []MenuItem)
	OpenXOffsetAutoCloseMenuWithCallback(xOffset int, items []MenuItem, callback func())
	PopModal()
	ShowTextInputAt(pos geometry.Point, width int, prompt string, prefilled string, onClose func(string), onAbort func())
	ShowTextInput(prompt string, prefilled string, onClose func(string), onAbort func())
	PopAll()
	ShowPager(title string, lines []core.StyledText, quit func())
	OpenItemRingMenu(currentItem *core.Item, listOfItems []*core.Item, selectedFunc func(*core.Item), cancelFunc func())
	HideModal()
	ShowModal()
	ShowWidget(widget UIWidget)
	HideWidget(widget UIWidget)
	OpenColorPicker(color common.Color, onChanged func(color common.Color), changed func(color common.Color))
	IsShowingUI() bool
	AddToScene(widget UIWidget)
	RemoveFromScene(widget UIWidget)
	Reset()
	RenderFancyText(startPos geometry.Point, text []string, finished func())
	ShowNoAbortTextInputAt(pos geometry.Point, width int, prompt string, prefilled string, onComplete func(string))
	ShowTooltipAt(screenPosition geometry.Point, text core.StyledText)
	ClearTooltip()
	CalculateLabelPlacement(origin geometry.Point, textLength int) geometry.Point
	IntersectsTooltip(bounds geometry.Rect) bool
	TooltipShown() bool
	BoundsForWorldLabel(worldPos geometry.Point, stringLength int) geometry.Rect
	InitTooltip(tipFunc func(origin geometry.Point, stringLength int) geometry.Rect)
	ShowAlert(strings []string)
	ShowStyledAlert(strings []core.StyledText, background common.Color)
	SetGamestate(state GameState)
}
type GameConfig struct {
	console.GridConfig
	ActorDefaultHealth int
	CampaignDirectory  string
	WebMode            bool
	MusicStreaming     bool
	Audio              bool
	LightSources       bool
	ShowHints          bool
}
type GameInterface interface {
	UpdateHUD()

	UpdateKnowledgeFromVision(person *core.Actor)
	GetMap() *gridmap.GridMap[*core.Actor, *core.Item, Object]
	GetCamera() *geometry.Camera

	PushState(newGameState GameState)
	PushGameplayState()
	PopState()
	PopAndInitPrevious()

	SpawnClothingItem(position geometry.Point, clothing core.Clothing)

	SendToSleep(target *core.Actor)
	Kill(victim *core.Actor, causeOfDeath core.CauseOfDeath)
	SnapNeck(attacker, target *core.Actor)

	MoveActor(person *core.Actor, to geometry.Point)
	MoveItemTo(pos geometry.Point, item *core.Item)
	PickUpItem(person *core.Actor)
	DropEquippedItem(player *core.Actor)

	ActorEnteredCell(person *core.Actor, oldPosition geometry.Point, newPosition geometry.Point)
	ApplyDelayed(pos geometry.Point, source core.EffectSource, effect stimuli.StimEffect, delay float64)
	Apply(pos geometry.Point, source core.EffectSource, effects stimuli.StimEffect)

	ApplyStimulusToThings(location geometry.Point, source core.EffectSource, stimulus stimuli.Stimulus)
	ApplyStimulusToTile(location geometry.Point, source core.EffectSource, stimulus stimuli.Stimulus)
	ApplyStimulusToActor(person *core.Actor, source core.EffectSource, stimulus stimuli.Stimulus)

	SwitchClothesWith(taker, provider *core.Actor)

	GetStats() *core.MissionStats
	GetMissionPlan() *core.MissionPlan
	GetActions() ActionsInterface

	IllegalPlayerEngagementWithActorAtPos(position geometry.Point, icon rune, engagementFinishedAction func(), engagementCancelledAction func())

	UpdateAllFoVsFrom(pos geometry.Point)

	ClearMap(width int, height int)
	ResetModel()
	DrawVisionCone(con console.CellInterface, actor *core.Actor)
	TryPushActorInDirection(actor *core.Actor, target geometry.Point)
	GetContextActionAt(pos geometry.Point) ContextAction
	CreatePickupAction(forItem *core.Item) ContextAction
	InitLoadedMap(*gridmap.GridMap[*core.Actor, *core.Item, Object])

	DrawWorldAtPosition(p geometry.Point, c gridmap.MapCell[*core.Actor, *core.Item, Object]) (rune, common.Style)
	DrawMapAtPosition(p geometry.Point, c gridmap.MapCell[*core.Actor, *core.Item, Object]) (rune, common.Style)
	ApplyLighting(p geometry.Point, cell gridmap.MapCell[*core.Actor, *core.Item, Object], currentStyle common.Style) common.Style

	GetConfig() *GameConfig
	GetValidItemPlacementPosition(pos geometry.Point, item *core.Item) geometry.Point
	PlaceItemWithOrigin(origin, pos geometry.Point, item *core.Item) geometry.Point
	PlaceItem(pos geometry.Point, item *core.Item) geometry.Point
	SendTriggerStimuli(user *core.Actor, item *core.Item, target geometry.Point, trigger core.ItemEffectTrigger)
	IsUnlockedDoorAt(pos geometry.Point) bool

	IsDoorAt(pos geometry.Point) bool
	Destroy(item *core.Item)
	AreAllies(actorOne, actorTwo *core.Actor) bool
	EndMissionWithSuccess()
	EndMissionWithFailure(core.CauseOfDeath)
	StopEverything()
	DropInventory(person *core.Actor)
	DropFromInventory(person *core.Actor, items []*core.Item)
	IsOnScreen(worldPosition geometry.Point) bool

	SoundEventAt(pos geometry.Point, kindOfSound core.Observation, maxDistance int)
	SuspiciousActionAt(pos geometry.Point, kindOfEvent core.Observation)
	IllegalActionAt(pos geometry.Point, kindOfEvent core.Observation)
	StartDialogueAction(initialSpeaker *core.Actor, dialogueName string) ContextAction
	InitActor(actor *core.Actor)
}

type AnimationInterface interface {
	AddParticle(p Particle)
	FoodAnimation(a *core.Actor, pos geometry.Point, done func())
	FallingAnimation(pos geometry.Point, completed func())
	VomitingAnimation(person *core.Actor, actionPosition geometry.Point, finishedCallback func())
	SleepingAnimation(person *core.Actor, finished func())
	SoundPropagationAnimation(sound core.Observation, tiles map[int][]geometry.Point, completed func())
	TaskAnimation(person *core.Actor, seconds float64, cancelled func(), finished func())
	BlastDistribution(location geometry.Point, source core.EffectSource, applyStim []stimuli.Stimulus, size int, pressure int)
	LiquidDistribution(location geometry.Point, source core.EffectSource, applyStim []stimuli.Stimulus, size int)
	PlayerChangeClothesAnimation(location geometry.Point, clothing core.Clothing, finished, cancelled func())
	ElectricityAnimation(tiles []geometry.Point, source core.EffectSource, stim stimuli.Stimulus)
	ActorEngagedIllegalAnimation(person *core.Actor, icon rune, actionPosition geometry.Point, timeNeededInSeconds float64, finishedCallback func(), cancelled func())
	ActorEngagedIllegalAnimationWithSound(person *core.Actor, icon rune, actionPosition geometry.Point, audioCue string, finishedCallback func(), cancelled func())
	ActorEngagedAnimation(person *core.Actor, rune int32, pos geometry.Point, time float64, finished func())
	BriefingAnimation(script *core.BriefingAnimation, finish func())
	ImageFadeIn(pixels [][]common.Color, cancel, finish func())
	ImageToImageFade(src, dest [][]common.Color, draw, cancel, finish func())
	ClearParticles()
	Reset()
}

type DataInterface interface {
	NewEmptyCell() *gridmap.MapCell[*core.Actor, *core.Item, Object]

	GroundTile() gridmap.Tile
	WallTile() gridmap.Tile

	NoClothing() core.Clothing
	DefaultClothing() core.Clothing
	DefaultPlayerClothing() core.Clothing

	Items() []*core.Item
	Clothing() []*core.Clothing
	Tiles() []*gridmap.Tile
	NameToClothing(clothesName string) core.Clothing
}

type AIInterface interface {
	IsControlledByAI(a *core.Actor) bool

	SwitchToInvestigation(person *core.Actor, incidentReport core.IncidentReport)
	SwitchToCombat(person *core.Actor, target *core.Actor)
	SwitchToVomit(person *core.Actor)
	SwitchToSnitch(person *core.Actor)
	SwitchToCleanup(person *core.Actor)
	SwitchToPanic(person *core.Actor, dangerLocations []geometry.Point)
	SwitchToWait(actor *core.Actor)
	SwitchStateBecauseOfNewKnowledge(person *core.Actor)

	UpdateVision(person *core.Actor)
	CalculateAllTaskPaths(actor *core.Actor)
	AddTask(actor *core.Actor, task core.ScheduledTask)
	TaskCountFor(actor *core.Actor) int

	ReportIncident(person *core.Actor, location geometry.Point, incidentType core.Observation) core.IncidentReport
	IsNearActiveIllegalIncident(person *core.Actor, location geometry.Point) bool
	MoveOnPath(person *core.Actor) core.AIUpdate
	PathSet(person *core.Actor, point geometry.Point, passable func(point geometry.Point) bool) bool
	TryContextActionAtTaskLocation(person *core.Actor, action func()) bool
	IsAtGuardPosition(person *core.Actor) bool

	MarkAsDone(incident core.IncidentReport)
	RaiseSuspicionAt(witness *core.Actor, suspiciousActor *core.Actor, delayInMS int)

	TransferKnowledge(one *core.Actor, two *core.Actor)
	IncidentsNeedCleanup(person *core.Actor) bool
	GetIncidentForCleanup(person *core.Actor) core.IncidentReport
	MarkAsCleaned(incident core.IncidentReport)
	GetIncidentForSnitching(person *core.Actor) core.IncidentReport
	IsIncidentHandled(person *core.Actor, incident core.IncidentReport) bool
	IsGuardAvailable() bool
	GetDangerousIncidents(person *core.Actor) []core.IncidentReport

	Update()
	SwitchToScript(target *core.Actor)
	TryPopScripted(actor *core.Actor)
	SetEngaged(person *core.Actor, engagedStatus core.ActorState, until func() bool)
	TryRegisterHandler(target *core.Actor, report core.IncidentReport) bool
	CreateTravelGroup(group mapset.Set[*core.Actor])
	DeleteTravelGroup(group mapset.Set[*core.Actor])
}

type FileInterface interface {
	GetSubdirectories(path string) []string
	FileExists(path string) bool
	LoadTextFile(path string) []string
	Open(filename string) (fs.File, error)
	GetFilesInPath(path string) []string
	ReadDir(path string) ([]fs.DirEntry, error)
}

type ActionsInterface interface {
	UseEquippedItemAtRange(user *core.Actor, targetPos geometry.Point)
	UseEquippedItemForMelee(user *core.Actor, targetPos geometry.Point)
	UseEquippedItemOnSelf(player *core.Actor)
	Update()
	Draw(con console.CellInterface)
	TryFeedbackForImpact(cue string, source geometry.Point, targetPos geometry.Point, distance int)
	Prod(prodder *core.Actor, prodTargetPos geometry.Point)
	Reset()
}

type GameEvent interface{}

type ObjectCreator struct {
	Name   string
	Icon   rune
	Create func(identifier string) Object
}

type ObjectFactoryInterface interface {
	SimpleObjects() []ObjectCreator
}

type Engine interface {
	// Managed Functions
	GetInput() InputInterface
	GetAudio() AudioInterface
	GetUI() UIInterface
	GetFiles() FileInterface
	GetFilesystem() embed.FS
	GetGame() GameInterface
	GetAnimator() AnimationInterface
	GetData() DataInterface
	GetAI() AIInterface
	GetCareer() *CareerData
	GetItemFactory() *ItemFactory
	GetObjectFactory() ObjectFactoryInterface

	ScreenGridWidth() int
	ScreenGridHeight() int
	MapWindowWidth() int
	MapWindowHeight() int
	// Schedule should only be used if the effect MUST NOT be cancelled or re-scheduled.
	// And the call must handle being run after the gameplay state has quit.
	Schedule(delayInSeconds float64, functionCall func())
	ScheduleWhen(condition func() bool, functionCall func())
	QuitGame()

	SaveMap(currentMap *gridmap.GridMap[*core.Actor, *core.Item, Object], folder string) error
	LoadMap(name string) (*gridmap.GridMap[*core.Actor, *core.Item, Object], error)
	Reset()

	GetAvailableTextFonts() []string
	SetTextFont(fontName string)
	SetTileFont(fontName string)
	CurrentTick() uint64
	ResetForGameplay()
	PublishEvent(event GameEvent)
	// SubscribeToEvents is meant to be used together with objects.NewFilter
	SubscribeToEvents(subscriber Subscriber)
}
type Subscriber interface {
	ReceiveMoreAfter(event GameEvent) bool
}

// NewFilter - Pass a function that receives the type of event you want to filter for.
// The function should return true if you want to continue receiving events.
// Return false if you want to stop receiving events (unsubscribe).
// Use together with SubscribeToEvents().
func NewFilter[T any](receiveMore func(T) bool) *EventFilter[T] {
	return &EventFilter[T]{ReceiveMore: receiveMore}
}

type EventFilter[T any] struct {
	ReceiveMore func(T) bool
}

func (e *EventFilter[T]) ReceiveMoreAfter(event GameEvent) bool {
	if typedEvent, ok := event.(T); ok {
		return e.ReceiveMore(typedEvent)
	}
	return true
}

type GameState interface {
	Init(engine Engine)
	Update(input InputInterface)
	Draw(con console.CellInterface)
	SetDirty()
	ClearOverlay()
}
type Particle interface {
	Update(engine Engine)
	Draw(grid console.CellInterface)
	IsDead() bool
}
type AudioHandle interface {
	Close() error
	IsPlaying() bool
}

type AudioInterface interface {
	UnloadAll()
	StopAll()
	PlayCue(cue string) AudioHandle
	PlayCueWithCallback(cue string, callback func()) AudioHandle
	IsCuePlaying(cue string) bool
	StartLoop(cue string) AudioHandle
	StartLoopStream(cue string) AudioHandle
	Stop(cue string)
	RegisterSoundCues(filenames []string)
	RegisterRandomizedSoundCues(directories []string)
	PlayCueAt(cue string, pos geometry.Point) AudioHandle
	SetMasterVolume(volume float64)
	SetMusicVolume(volume float64)
	SetSoundVolume(volume float64)
	GetMasterVolume() float64
	GetMusicVolume() float64
	GetSoundVolume() float64
	PreLoadCuesIntoMemory(soundCue []string)
	UnloadCues(soundCue []string)
}

type KeyDefinitions struct {
	MovementKeys      [4]ebiten.Key
	PeekingKeys       [4]ebiten.Key
	ActionKeys        [4]ebiten.Key
	SameTileActionKey ebiten.Key
	SneakModeKey      ebiten.Key
	DropItemKey       ebiten.Key
	PutItemAwayKey    ebiten.Key
	UseItemKey        ebiten.Key
	InventoryKey      ebiten.Key
}

type InputInterface interface {
	SetMovementDelayForSneaking()
	SetMovementDelayForWalkingAndRunning()
	PollGameCommands() []core.InputCommand
	PollUICommands() []core.InputCommand
	PollEditorCommands() []core.InputCommand
	ConfirmOrCancel() bool
	DevTerminalKeyPressed() bool
	PollText() []core.InputCommand
	GetKeyDefinitions() KeyDefinitions
	IsShiftPressed() bool
}

type Challenge interface {
	ID() int
	Name() string
	Reward() int
	IsCompleted() bool
	CompletionTime() time.Duration

	IsCustom() bool
}
