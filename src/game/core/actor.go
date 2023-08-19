package core

import (
	"bytes"
	"fmt"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
	"strconv"
	"strings"
	"time"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/mapset"
	"github.com/memmaker/terminal-assassin/utils"
)

type CoDDescription string

type CauseOfDeath struct {
	Description CoDDescription
	Source      EffectSource
}

func (d CauseOfDeath) IsPlayer() bool {
	return d.Source.Actor != nil && d.Source.Actor.IsPlayer()
}

func (d CauseOfDeath) WithoutKiller() string {
	if d.Source.Item != nil {
		return fmt.Sprintf(string(d.Description), d.Source.Item.Name)
	}
	return string(d.Description)
}

func (d CauseOfDeath) WithKiller() string {
	if d.Source.Actor != nil && d.Source.Item != nil {
		return fmt.Sprintf(string(d.Description), d.Source.Item.Name) + " by " + d.Source.Actor.Name
	} else if d.Source.Actor != nil {
		return string(d.Description) + " by " + d.Source.Actor.Name
	} else if d.Source.Item != nil {
		return fmt.Sprintf(string(d.Description), d.Source.Item.Name)
	}
	return string(d.Description)
}

const (
	CoDStrangledWithWire CoDDescription = "strangled with piano wire"
	CoDPoisoned          CoDDescription = "was lethally poisoned"
	CoDDrowned           CoDDescription = "drowned"
	CoDDrownedInToilet   CoDDescription = "drowned in a toilet"
	CoDBrokenNeck        CoDDescription = "broke his neck"
	CoDOnePistolRound    CoDDescription = "a straight shot using %s"
	CodExploded          CoDDescription = "was blown to pieces by %s"
	CoDBurned            CoDDescription = "was burned"
	CoDElectrocuted      CoDDescription = "was electrocuted"
	CoDSnipered          CoDDescription = "was sniped with %s"
	CoDSubShot           CoDDescription = "bullets from %s"
	CoDShotGun           CoDDescription = "was struck by %s"
	CoDAutoShot          CoDDescription = "heavy lead injection from %s"
	CoDStabbed           CoDDescription = "stabbed with %s"
	CoDPenetrated        CoDDescription = "deadly penetration with %s"
	CoDFalling           CoDDescription = "fell to his death"
)

func NewCauseOfDeath(description CoDDescription, killer *Actor) CauseOfDeath {
	return CauseOfDeath{Description: description, Source: EffectSource{Actor: killer}}
}

func NewCauseOfDeathFromEnvironment(description CoDDescription, killingTile gridmap.Tile) CauseOfDeath {
	return CauseOfDeath{Description: description, Source: EffectSource{Tile: killingTile}}
}
func NewCauseOfDeathFromStim(stim stimuli.StimulusType, source EffectSource) CauseOfDeath {
	return CauseOfDeath{Description: source.ToCoDFromStim(stim), Source: source}
}

type ActorState string

const (
	ActorStatusIdle               ActorState = "idle"
	ActorStatusOnSchedule         ActorState = "on schedule"
	ActorStatusFollowing          ActorState = "following"
	ActorStatusEngaged            ActorState = "engaged"
	ActorStatusEngagedIllegal     ActorState = "engaged illegal"
	ActorStatusVictimOfEngagement ActorState = "victim of engagement"
	ActorStatusSleeping           ActorState = "sleeping"
	ActorStatusDead               ActorState = "dead"
	ActorStatusInvestigating      ActorState = "investigating"
	ActorStatusCombat             ActorState = "combat"
	ActorStatusVomiting           ActorState = "vomiting"
	ActorStatusSearching          ActorState = "searching"
	ActorStatusScripted           ActorState = "scripted"
	ActorStatusWatching           ActorState = "watching"
	ActorStatusCleanup            ActorState = "cleanup"
	ActorStatusPanic              ActorState = "panic"
	ActorStatusInCloset           ActorState = "in closet"
	ActorStatusSnitching          ActorState = "snitching"
)

type DamageInfo struct {
	Amount int
	Type   stimuli.StimulusType
}

type ActorType string

const (
	ActorTypeCivilian ActorType = "civilian"
	ActorTypeGuard    ActorType = "guard"
	ActorTypeEnforcer ActorType = "enforcer"
	ActorTypeTarget   ActorType = "target"
)

type MovementMode rune

const (
	MovementModeWalking  MovementMode = 'ˁ'
	MovementModeSneaking MovementMode = '˂'
	MovementModeRunning  MovementMode = '˃'
)

// AutoMove represents the information for an automatic-movement step.
type AutoMove struct {
	// Delta represents a position variation such as (0,1), that
	// will be used in position arithmetic to move from one position to an
	// adjacent one in a certain direction.
	Delta geometry.Point
	Path  bool // whether following a Path (instead of a simple direction)
}

type Actor struct {
	MapPos             geometry.Point
	LastPos            geometry.Point
	Move               AutoMove         // automatic movement
	Path               []geometry.Point // current Path (reverse highlighting)
	LookDirection      float64          // angle in degrees = 270 = north = up
	Fov                *geometry.FOV    // field of vision
	FovMode            gridmap.FoVMode  // field of vision mode
	FovShiftForPeeking geometry.Point   // field of vision shift for peeking
	Name               string
	EquippedItem       *Item
	Inventory          *InventoryComponent
	Dialogue           *DialogueComponent
	AutoMoveSpeed      int
	Status             ActorState
	Health             int
	DamageTaken        []DamageInfo
	FoVinDegrees       float64
	Type               ActorType
	MovementMode       MovementMode
	DebugFlag          bool
	DraggedBody        *Actor
	Clothes            Clothing
	MaxVisionRange     int
	AI                 *AIComponent
	Script             *ScriptComponent
	IsEyeWitness       bool
	IsHidden           bool
	IsBodyBagged       bool
}
type OrientedLocation struct {
	Location  geometry.Point
	Direction float64
}

type IndividualKnowledge struct {
	CompromisedDisguises          mapset.Set[string]
	LastSightingOfDangerousActor  IncidentReport
	LastSightingOfSuspiciousActor IncidentReport
}

func (k *IndividualKnowledge) AddSightingOfDangerousActor(witness, dangerMan *Actor, observedBehavior Observation, atTick uint64) {
	inTheKnow := mapset.NewSet[*Actor]()
	inTheKnow.Add(witness)
	noSightingBeforeThis := k.LastSightingOfDangerousActor.Tick == 0
	k.LastSightingOfDangerousActor = IncidentReport{
		Location:          dangerMan.Pos(),
		Type:              observedBehavior,
		Tick:              atTick,
		FinishedHandling:  false,
		RegisteredHandler: nil,
		KnownBy:           inTheKnow,
	}
	if dangerMan.IsPlayer() {
		k.CompromisedDisguises.Add(dangerMan.NameOfClothing())
	}
	if noSightingBeforeThis {
		println(fmt.Sprintf("%s saw a hostile person wearing '%s' doing '%s' at %s", witness.DebugDisplayName(), dangerMan.NameOfClothing(), observedBehavior, dangerMan.Pos()))
	}
}
func (k *IndividualKnowledge) AddSightingOfSuspiciousActor(witness *Actor, location geometry.Point, observedBehavior Observation, atTick uint64) {
	inTheKnow := mapset.NewSet[*Actor]()
	inTheKnow.Add(witness)
	noSightingBeforeThis := k.LastSightingOfSuspiciousActor.Tick == 0
	k.LastSightingOfSuspiciousActor = IncidentReport{
		Location:          location,
		Tick:              atTick,
		FinishedHandling:  false,
		RegisteredHandler: nil,
		KnownBy:           inTheKnow,
		Type:              observedBehavior,
	}
	if noSightingBeforeThis {
		println(fmt.Sprintf("%s saw a suspicious person doing '%s' at %s", witness.DebugDisplayName(), observedBehavior, location))
	}
}

type AIState interface{}
type AIComponent struct {
	PathBlockedCount    int
	Knowledge           *IndividualKnowledge
	Schedule            Schedule
	stateStack          []AIState
	StartPosition       geometry.Point
	StartLookDirection  float64
	SuspicionCounter    int
	LastSuspicionRaised time.Time
	Movement            AIMovement
	TicksToUpdate       int
	UpdatePredicate     func() bool
	DebugFlag           bool
}

func (a *AIComponent) GetState() AIState {
	return a.stateStack[len(a.stateStack)-1]
}
func (a *AIComponent) SetState(state AIState) {
	a.stateStack = []AIState{state}
	//println(fmt.Sprintf("AI state set to %T", state))
}
func (a *AIComponent) PushState(state AIState) {
	a.stateStack = append(a.stateStack, state)
	if !a.DebugFlag {
		return
	}
	println(fmt.Sprintf("AI state pushed: %T", state))
	a.debugPrintStateStack()
}
func (a *AIComponent) ReplaceState(state AIState) {
	a.stateStack[len(a.stateStack)-1] = state
	println(fmt.Sprintf("AI state replaced with %T", state))
	// print stack in reverse
	for i := len(a.stateStack) - 1; i >= 0; i-- {
		println(fmt.Sprintf("  %T", a.stateStack[i]))
	}
}
func (a *AIComponent) PopState() {
	if len(a.stateStack) <= 1 {
		println(fmt.Sprintf("WARNING: Tried to pop AI state, but stack would be empty"))
		return
	}
	stateToPop := a.stateStack[len(a.stateStack)-1]
	a.stateStack = a.stateStack[:len(a.stateStack)-1]
	if !a.DebugFlag {
		return
	}
	println(fmt.Sprintf("AI state popped: %T", stateToPop))
	a.debugPrintStateStack()
}

func (a *AIComponent) debugPrintStateStack() {
	for i := len(a.stateStack) - 1; i >= 0; i-- {
		println(fmt.Sprintf("| %T", a.stateStack[i]))
	}
}
func (a *AIComponent) HasTasks() bool {
	return len(a.Schedule.Tasks) > 0
}

func (a *AIComponent) addTask(task ScheduledTask) {
	a.Schedule.Tasks = append(a.Schedule.Tasks, task)
}

func (a *AIComponent) getTaskAt(position geometry.Point) *ScheduledTask {
	for i, task := range a.Schedule.Tasks {
		if task.Location == position {
			return &a.Schedule.Tasks[i]
		}
	}
	return nil
}

func (a *AIComponent) HasTaskAt(position geometry.Point) bool {
	return a.getTaskAt(position) != nil
}

func (a *AIComponent) GetTaskIndexAt(pos geometry.Point) int {
	for i, task := range a.Schedule.Tasks {
		if task.Location == pos {
			return i
		}
	}
	return -1
}

func (a *AIComponent) IsAtTaskLocation(person *Actor) bool {
	return a.Schedule.Tasks != nil &&
		len(a.Schedule.Tasks) > 0 &&
		person.Pos() == a.Schedule.CurrentTask().Location
}

func (a *AIComponent) LowerSuspicion() {
	a.SuspicionCounter--
	if a.SuspicionCounter < 0 {
		a.SuspicionCounter = 0
	}
}

func (a *AIComponent) IsUpdateAllowed() bool {
	return a.UpdatePredicate == nil || a.UpdatePredicate()
}
func NewEmptyAIComponent() *AIComponent {
	aiBehaviour := &AIComponent{
		Knowledge: &IndividualKnowledge{
			CompromisedDisguises: mapset.NewSet[string](),
		},
		Schedule: Schedule{Tasks: make([]ScheduledTask, 0)},
	}
	return aiBehaviour
}

func NewActor(name string, clothing Clothing) *Actor {
	newActor := &Actor{
		Name:           name,
		Type:           ActorTypeCivilian,
		Clothes:        clothing,
		MovementMode:   MovementModeWalking,
		AutoMoveSpeed:  3,
		FoVinDegrees:   90,
		MaxVisionRange: 10,
		LookDirection:  float64(geometry.East),
	}
	newActor.Fov = geometry.NewFOV(geometry.NewRect(-newActor.MaxVisionRange, -newActor.MaxVisionRange, newActor.MaxVisionRange+1, newActor.MaxVisionRange+1))
	newActor.AI = NewEmptyAIComponent()
	newActor.Script = &ScriptComponent{}
	newActor.Inventory = &InventoryComponent{Items: []*Item{}}
	newActor.Dialogue = &DialogueComponent{Conversations: make(map[string]*Conversation), SpokenSpeech: mapset.NewSet[string](), HeardSpeech: mapset.NewSet[string]()}
	return newActor
}

func NewDefaultResponses() map[string]Utterance {
	return map[string]Utterance{
		string(ObservationDownedSpeaker): {
			Line:      Text("Damn, what happened?!"),
			EventCode: "DLG_observed_downed_00",
		},
	}
}
func (a *Actor) Pos() geometry.Point {
	return a.MapPos
}

func (a *Actor) Icon() rune {
	if a.DraggedBody != nil {
		return '☠'
	}
	return '☺'
}

func (a *Actor) SetPos(point geometry.Point) {
	a.LastPos = a.MapPos
	a.MapPos = point
}

// what do we need to signal to the player?

// light conditions - Variation of Lightness of Foreground
// Line of sight - Variation of Lightness of Foreground
// Vision Cones?
// Type of enemy (especially targets, guards and enforcers)
// stateStack of enemy (sleeping, dead, engaged, etc)
// Environmental hazards (fire, water, etc) - Must: Background Color, Optional: Foreground Color & icon

type Clothing struct {
	Name    string
	FgColor common.HSVColor
	BgColor common.HSVColor
}

func (c Clothing) EncodeAsString() string {
	return fmt.Sprintf("%s\t(%.2f, %.2f, %.2f)\t(%.2f, %.2f, %.2f)", c.Name, c.FgColor.H, c.FgColor.S, c.FgColor.V, c.BgColor.H, c.BgColor.S, c.BgColor.V)
}

func NewClothingFromString(encoded string) *Clothing {
	parts := strings.Split(encoded, "\t")
	name := strings.TrimSpace(parts[0])
	fgParts := strings.Split(strings.Trim(parts[1], "() "), ",")
	bgParts := strings.Split(strings.Trim(parts[2], "() "), ",")
	fgColor := common.HSVColor{H: mustParseFloat(fgParts[0]), S: mustParseFloat(fgParts[1]), V: mustParseFloat(fgParts[2])}
	bgColor := common.HSVColor{H: mustParseFloat(bgParts[0]), S: mustParseFloat(bgParts[1]), V: mustParseFloat(bgParts[2])}
	return &Clothing{Name: name, FgColor: fgColor, BgColor: bgColor}
}

func mustParseFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		println("Error parsing float: " + s)
		return 0
	}
	return f
}

func MustParseInt(s string) int {
	f, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		println("Error parsing float: " + s)
		return 0
	}
	return int(f)
}

func (a *Actor) IsVisible() bool {
	return a.Status != ActorStatusInCloset && !a.IsHidden
}

func (a *Actor) CanBeDistracted() bool {
	return a.IsActive() && !a.IsVictimOfEngagement() && !a.IsPlayer() && !a.IsInvestigating()
}

func (a *Actor) CanUseActions() bool {
	return a.IsActive() && !a.IsEngaged() && !a.IsVictimOfEngagement()
}
func (a *Actor) CanMove() bool {
	return a.IsActive() && !a.IsEngaged() && !a.IsVictimOfEngagement() && a.Status != ActorStatusInCloset
}

func (a *Actor) CanPerceive() bool {
	return a.IsActive() && !a.IsVictimOfEngagement()
}
func (a *Actor) CanUseItems() bool {
	return a.IsActive() && !a.IsEngaged() && !a.IsVictimOfEngagement() && a.Status != ActorStatusInCloset
}
func (a *Actor) IsIdle() bool {
	return a.Status == ActorStatusIdle
}

func (a *Actor) IsInvestigating() bool {
	return a.Status == ActorStatusInvestigating
}
func (a *Actor) IsEngaged() bool {
	return a.Status == ActorStatusEngaged || a.Status == ActorStatusEngagedIllegal || a.Status == ActorStatusVomiting
}

func (a *Actor) IsVictimOfEngagement() bool {
	return a.Status == ActorStatusVictimOfEngagement
}
func (a *Actor) IsActive() bool {
	return a.Status != ActorStatusDead && a.Status != ActorStatusSleeping
}

func (a *Actor) IsDowned() bool {
	return a.Status == ActorStatusDead || a.Status == ActorStatusSleeping
}

func (a *Actor) IsFollowing() bool {
	return a.Status == ActorStatusFollowing
}

func (a *Actor) IsInCombat() bool {
	return a.Status == ActorStatusCombat
}
func (a *Actor) IsDead() bool {
	return a.Status == ActorStatusDead
}
func (a *Actor) IsAlive() bool {
	return a.Status != ActorStatusDead
}
func (a *Actor) IsEngagedInIllegalAction() bool {
	return a.Status == ActorStatusEngagedIllegal
}

func (a *Actor) HasIllegalItemEquipped() bool {
	return a.EquippedItem != nil && !a.EquippedItem.IsLegalForActor(a)
}
func (a *Actor) AddDamage(Amount int, Type stimuli.StimulusType) {
	a.DamageTaken = append(a.DamageTaken, DamageInfo{Amount, Type})
}
func (a *Actor) FoVSource() geometry.Point {
	return a.Pos().Add(a.FovShiftForPeeking)
}

type AIMoveDelay float64

const (
	MoveDelayRunning        AIMoveDelay = 0.3
	MoveDelayBodyguardSpeed AIMoveDelay = 0.4
	MoveDelayWalking        AIMoveDelay = 0.8
	MoveDelayBrokenLegs     AIMoveDelay = 3
)

func (a *Actor) ShiftSchedulesBy(offset geometry.Point, mapSize geometry.Point) {
	if a.IsPlayer() {
		return
	}
	newTasks := make([]ScheduledTask, len(a.AI.Schedule.Tasks))
	for i, task := range a.AI.Schedule.Tasks {
		newTasks[i] = task.WithLocationShifted(offset, mapSize)
	}
	a.AI.Schedule.Tasks = newTasks
}

func (a *Actor) MoveDelay() AIMoveDelay {
	switch {
	case a.Status == ActorStatusFollowing:
		return MoveDelayBodyguardSpeed
	case a.IsInCombat():
		return MoveDelayRunning
	case a.Status == ActorStatusSnitching:
		return MoveDelayRunning
	default:
		return MoveDelayWalking
	}
}
func (a *Actor) Symbol() rune {
	switch a.Status {
	case ActorStatusSleeping:
		return 'z'
	case ActorStatusDead:
		return GlyphCorpse
	default:
		//return a.runeFromName()
		return a.runeFromDirection()
	}
}
func (a *Actor) startSneaking() {
	a.MovementMode = MovementModeSneaking
}

func (a *Actor) startRunning() {
	a.MovementMode = MovementModeRunning
}

func (a *Actor) startWalking() {
	a.MovementMode = MovementModeWalking
}

func (a *Actor) runeFromName() rune {
	return bytes.Runes([]byte(a.Name))[0]
}
func (a *Actor) runeFromDirection() rune {
	actorRune := runeFromDirection(a.LookDirection)
	offset := runeOffsetFromStatus(a.Status)
	actorRune += offset
	return actorRune
}

func (a *Actor) Style(st common.Style) common.Style {
	actorStyle := st.WithFg(a.Clothes.FgColor)
	if a.IsEyeWitness {
		actorStyle = st.WithFg(common.HSVColor{H: a.Clothes.FgColor.H, S: a.Clothes.FgColor.S, V: 4})
	}
	if a.Dialogue.IsCurrentlySpeaking {
		actorStyle = a.ChatStyle()
	}
	if a.DebugFlag {
		actorStyle = st.WithBg(common.Yellow)
	}
	return actorStyle // WithBg(a.Clothes.Color)
}

func (a *Actor) CanSee(p geometry.Point) bool {
	if geometry.DistanceChebyshev(p, a.Pos()) <= 1 {
		return true
	}
	switch a.FovMode {
	case gridmap.FoVModeScoped:
		return a.canSeeInScope(p)
	default:
		visible := a.Fov.Visible(p)
		return visible
	}
}

func (a *Actor) CanSeeActor(other *Actor) bool {
	if a == other {
		return true
	}
	return a.CanSee(other.Pos()) && other.IsVisible()
}

func (a *Actor) PutItemAway() {
	if a.EquippedItem == nil || a.EquippedItem.IsBig {
		return
	}
	a.EquippedItem = nil
	return
}
func (a *Actor) HasLineToSpeak(currentTick uint64) bool {
	deltaTicks := int(currentTick - a.Dialogue.LastSpokenAtTick)
	if deltaTicks < utils.SecondsToTicks(4) {
		return false
	}
	atDialoguePosition := a.Dialogue.Situation == nil || a.Pos() == a.Dialogue.Situation.Location
	return a.Dialogue.Available(a.Status) && (atDialoguePosition || a.IsPlayer())

}
func (a *Actor) TryRespondToSpeech(currentTick uint64, speechCode string) {
	a.Dialogue.DidHear(speechCode, currentTick)
	response, hasResponse := a.Dialogue.TryResponding(speechCode)
	if hasResponse {
		println(fmt.Sprintf("%s is responding to %s with %s", a.DebugDisplayName(), speechCode, response.EventCode))
		a.SetNextUtterance(response)
	}
}
func (a *Actor) canSeeInScope(p geometry.Point) bool {
	if a.EquippedItem == nil || a.EquippedItem.Scope.Range == 0 {
		return false
	}
	scopeInfo := a.EquippedItem.Scope
	left, right := geometry.GetLeftAndRightBorderOfVisionCone(a.FoVSource(), a.LookDirection, scopeInfo.FoVinDegrees)
	inCone := geometry.InVisionCone(a.FoVSource(), p, left, right)
	visible := a.Fov.Visible(p)
	return inCone && visible
}

func (a *Actor) CanSeeInVisionCone(p geometry.Point) bool {
	left, right := geometry.GetLeftAndRightBorderOfVisionCone(a.FoVSource(), a.LookDirection, a.FoVinDegrees)
	inCone := geometry.InVisionCone(a.FoVSource(), p, left, right)
	visible := a.Fov.Visible(p)
	return visible && inCone
}
func (a *Actor) VisionCone(f func(p geometry.Point)) {
	left, right := geometry.GetLeftAndRightBorderOfVisionCone(a.FoVSource(), a.LookDirection, a.FoVinDegrees)
	a.Fov.IterSSC(func(p geometry.Point) {
		if geometry.DistanceSquared(a.FoVSource(), p) <= (a.VisionRange()*a.VisionRange()) && geometry.InVisionCone(a.FoVSource(), p, left, right) {
			f(p)
		}
	})
}

func (a *Actor) HasPath() bool {
	return a.Path != nil && len(a.Path) > 0
}

func (a *Actor) HasPathTo(location geometry.Point) bool {
	return a.Path != nil && len(a.Path) > 0 && a.Path[len(a.Path)-1] == location
}

func (a *Actor) Str(delimiter string) string {
	return strings.Join(a.StrList(), delimiter)
}

func (a *Actor) StrList() []string {
	charSheet := []string{
		fmt.Sprintf("name: %s", a.Name),
		fmt.Sprintf("mapPos: %s", a.Pos()),
		fmt.Sprintf("LookDirection: %f (%s)", a.LookDirection, DirectionAsString(a)),
		fmt.Sprintf("Health: %d", a.Health),
		fmt.Sprintf("AutoMoveSpeed: %d", a.AutoMoveSpeed),
		fmt.Sprintf("Status: %s", a.Status),
	}
	charSheet = append(charSheet, "State Stack:")
	for _, state := range a.AI.stateStack {
		charSheet = append(charSheet, fmt.Sprintf("  %T", state))
	}
	charSheet = append(charSheet, fmt.Sprintf("Damage Taken:"))
	for _, damage := range a.DamageTaken {
		charSheet = append(charSheet, fmt.Sprintf("  %d of %s damage", damage.Amount, damage.Type))
	}

	return charSheet
}

func (a *Actor) TurnLeft(angleInDegrees float64) {
	a.LookDirection -= angleInDegrees
	for a.LookDirection < 0 {
		a.LookDirection += 360
	}
}

func (a *Actor) TurnRight(angleInDegrees float64) {
	a.LookDirection = a.LookDirection + angleInDegrees
	for a.LookDirection >= 360 {
		a.LookDirection -= 360
	}
}

func (a *Actor) EquipWeapon() bool {
	if a.Inventory == nil || len(a.Inventory.Items) == 0 {
		return false
	}
	if a.EquippedItem != nil && a.EquippedItem.IsWeapon() {
		return true
	}
	for _, item := range a.Inventory.Items {
		if item.IsWeapon() {
			a.EquippedItem = item
			return true
		}
	}
	return false
}

func (a *Actor) HasToolEquipped(tool ItemType) bool {
	if a.EquippedItem == nil {
		return false
	}
	return a.EquippedItem.Type == tool
}

func (a *Actor) HasKeyInInventory(keyString string) bool {
	if a.Inventory == nil || len(a.Inventory.Items) == 0 {
		return false
	}
	for _, item := range a.Inventory.Items {
		if item.Type == ItemTypeKey && item.KeyString == keyString {
			return true
		}
	}
	return false
}
func (a *Actor) HasKeyCardInInventory(keyString string) bool {
	if a.Inventory == nil || len(a.Inventory.Items) == 0 {
		return false
	}
	for _, item := range a.Inventory.Items {
		if item.Type == ItemTypeKeyCard && item.KeyString == keyString {
			return true
		}
	}
	return false
}

func (a *Actor) IsDraggingBody() bool {
	return a.DraggedBody != nil
}

func (a *Actor) IsBleeding() bool {
	for _, damage := range a.DamageTaken {
		if damage.Type == stimuli.StimulusPiercingDamage {
			return true
		}
	}
	return false
}

func (a *Actor) HasWeapon() bool {
	for _, item := range a.Inventory.Items {
		if item.IsWeapon() {
			return true
		}
	}
	return false
}

func (a *Actor) HasBurstWeaponEquipped() bool {
	if a.EquippedItem == nil {
		return false
	}
	return a.EquippedItem.IsAutomaticRangedWeapon()
}

func (a *Actor) LookAt(pos geometry.Point) {
	directionVector := pos.Sub(a.Pos())
	a.LookDirection = geometry.DirectionVectorToAngleInDegrees(directionVector)
}

func runeOffsetFromStatus(status ActorState) rune {
	switch status {
	default:
		return 0
	case ActorStatusInvestigating:
		return 8
	case ActorStatusCombat:
		return 16
	}
}

func DirectionAsString(a *Actor) string {
	switch geometry.CompassDirection(a.LookDirection) {
	case geometry.North:
		return "North"
	case geometry.NorthEast:
		return "NorthEast"
	case geometry.East:
		return "East"
	case geometry.SouthEast:
		return "SouthEast"
	case geometry.South:
		return "South"
	case geometry.SouthWest:
		return "SouthWest"
	case geometry.West:
		return "West"
	case geometry.NorthWest:
		return "NorthWest"
	default:
		return "Unknown"
	}
}

// runeFromDirection will map any look direction in degrees to a compass direction index.
// eg. 12 degrees will map to 0 (East), 46 degrees will map to 1 (NorthEast), 90 degrees will map to 2 (North), etc.
func runeFromDirection(lookDirInDegrees float64) rune {
	// normalize to [0, 360]
	lookDirInDegrees += 22.5
	if lookDirInDegrees < 0 {
		lookDirInDegrees += 360
	} else if lookDirInDegrees > 360 {
		lookDirInDegrees -= 360
	}
	for i := 0; i < 8; i++ {
		compassDirectionDegrees := float64(i*45) + 22.5
		if lookDirInDegrees >= compassDirectionDegrees-22.5 && lookDirInDegrees < compassDirectionDegrees+22.5 {
			return rune(285) + rune(i)
		}
	}
	return '@'
}
func (a *Actor) FoV() *geometry.FOV {
	return a.Fov
}
func (a *Actor) FoVMode() gridmap.FoVMode {
	return a.FovMode
}
func (a *Actor) VisionRange() int {
	if a.FovMode == gridmap.FoVModeScoped && a.HasScopedItemEquipped() {
		return a.EquippedItem.Scope.Range
	}
	return a.MaxVisionRange
}
func (a *Actor) NameOfClothing() string {
	return a.Clothes.Name
}

func (a *Actor) HasScopedItemEquipped() bool {
	if a.EquippedItem == nil {
		return false
	}
	return a.EquippedItem.Scope.Range > 0
}

func (a *Actor) IsSleeping() bool {
	return a.Status == ActorStatusSleeping
}

func (a *Actor) DiesFromDamage(damage int) bool {
	a.Health -= damage
	if a.Health <= 0 {
		a.Health = 0
		return true
	}
	return false
}

func (a *Actor) DebugDisplayName() string {
	if a.AI == nil {
		return fmt.Sprintf("%s (%s - %s)", a.Name, a.Type, a.Status)
	}
	return fmt.Sprintf("%s (%s - %s / %T)", a.Name, a.Type, a.Status, a.AI.GetState())
}

func (a *Actor) IsAvailableGuard() bool {
	return a.Type == ActorTypeGuard && a.IsActive() && !a.IsInCombat()
}

func (a *Actor) IsPlayer() bool {
	return a.AI == nil
}

func (a *Actor) SetNextUtterance(text Utterance) {
	a.Dialogue.NextUtterance = text
}

func (a *Actor) StartDialogue(name string) {
	if conversation, ok := a.Dialogue.Conversations[name]; ok {
		if starter, startOk := conversation.Responses[""]; startOk {
			a.Dialogue.CurrentDialogue = name
			a.SetNextUtterance(starter)
		}
	}
}
func (a *Actor) ChatStyle() common.Style {
	return common.Style{
		Foreground: a.Clothes.FgColor.WithV(a.Clothes.FgColor.V + 2),
		Background: a.Clothes.BgColor,
	}
}

func (a *Actor) EnableDebugTrace() {
	a.DebugFlag = true
	a.AI.DebugFlag = true
}

func (a *Actor) DisableDebugTrace() {
	a.DebugFlag = false
	a.AI.DebugFlag = false
}

func (a *Actor) TooltipText() string {
	return fmt.Sprintf("%s / %s / %s", a.Name, a.Type, a.Status)
}

type ActorOnDisk struct {
	Name          string
	Clothing      string
	Inventory     []string
	ActorType     ActorType
	MoveSpeed     int
	FoVinDegrees  float64
	VisionRange   int
	LookDirection float64
	Position      geometry.Point
}

func (d ActorOnDisk) ToRecord() []rec_files.Field {
	record := []rec_files.Field{
		{Name: "Name", Value: d.Name},
		{Name: "ActorType", Value: string(d.ActorType)},
		{Name: "Position", Value: d.Position.String()},
		{Name: "LookDirection", Value: strconv.FormatFloat(d.LookDirection, 'f', 2, 64)},
		{Name: "Clothing", Value: d.Clothing},
		{Name: "MoveSpeed", Value: strconv.Itoa(d.MoveSpeed)},
		{Name: "FoVinDegrees", Value: strconv.FormatFloat(d.FoVinDegrees, 'f', 2, 64)},
		{Name: "VisionRange", Value: strconv.Itoa(d.VisionRange)},
	}
	//		{Name: "Inventory", Value: d.Inventory},
	for _, item := range d.Inventory {
		record = append(record, rec_files.Field{Name: "Inventory", Value: item})
	}
	return record
}
func ActorOnDiskFromRecord(record []rec_files.Field) ActorOnDisk {
	actor := ActorOnDisk{}
	for _, field := range record {
		switch field.Name {
		case "Name":
			actor.Name = strings.TrimSpace(field.Value)
		case "ActorType":
			actor.ActorType = ActorType(field.Value)
		case "Position":
			actor.Position, _ = geometry.NewPointFromString(field.Value)
		case "LookDirection":
			actor.LookDirection, _ = strconv.ParseFloat(field.Value, 64)
		case "Clothing":
			actor.Clothing = strings.TrimSpace(field.Value)
		case "MoveSpeed":
			actor.MoveSpeed, _ = strconv.Atoi(field.Value)
		case "FoVinDegrees":
			actor.FoVinDegrees, _ = strconv.ParseFloat(field.Value, 64)
		case "VisionRange":
			actor.VisionRange, _ = strconv.Atoi(field.Value)
		case "Inventory":
			actor.Inventory = append(actor.Inventory, strings.TrimSpace(field.Value))
		}
	}
	return actor
}
