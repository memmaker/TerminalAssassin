package ai

import (
	"fmt"
	"time"

	"github.com/memmaker/terminal-assassin/rng"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/mapset"
	"github.com/memmaker/terminal-assassin/utils"
)

// stateTransitionDelay is the seconds an actor waits before its first AI
// update after a state switch (≈20 ticks at 60 TPS).
const stateTransitionDelay = 1.0 / 3.0

type AIContext struct {
	Engine services.Engine
	Person *core.Actor
}

func NextUpdateIn(delayInSeconds float64) core.AIUpdate {
	return core.AIUpdate{
		DelayInSeconds:  delayInSeconds,
		UpdatePredicate: nil,
	}
}

func DeferredUpdate(predicate func() bool) core.AIUpdate {
	return core.AIUpdate{
		DelayInSeconds:  core.ManualDeferredUpdate.DelayInSeconds,
		UpdatePredicate: predicate,
	}
}

// Guard AI Reactions to new knowledge
// Suspicious Observation -> Investigate
// Dangerous Observation  -> Combat or Alarm

// When a guard was in Combat or Alarm was triggered
// -> Enter heightened alert state

type AIController struct {
	engine               services.Engine
	travelGroups         mapset.Set[mapset.Set[*core.Actor]]
	activeInvestigations mapset.Set[string]
	activeCleanups       mapset.Set[string]
}

func (a *AIController) CreateTravelGroup(group mapset.Set[*core.Actor]) {
	a.travelGroups.Add(group)
	println("Created travel group:")
	for _, person := range group.ToSlice() {
		println(fmt.Sprintf("\t%s", person.Name))
	}
}

func (a *AIController) DeleteTravelGroup(group mapset.Set[*core.Actor]) {
	a.travelGroups.Remove(group)
	println("Deleted travel group:")
	for _, person := range group.ToSlice() {
		println(fmt.Sprintf("\t%s", person.Name))
	}
}

func (a *AIController) IsPartOfTravelGroup(person *core.Actor) bool {
	for _, group := range a.travelGroups.ToSlice() {
		if group.Contains(person) {
			return true
		}
	}
	return false
}

func (a *AIController) GetTravelGroup(person *core.Actor) mapset.Set[*core.Actor] {
	for _, group := range a.travelGroups.ToSlice() {
		if group.Contains(person) {
			return group
		}
	}
	return nil
}

func (a *AIController) IsNearActiveIllegalIncident(_ *core.Actor, location geometry.Point) bool {
	currentMap := a.engine.GetGame().GetMap()
	for _, actor := range currentMap.DownedActors() {
		pos := actor.Pos()
		zone := currentMap.ZoneAt(pos)
		if zone != nil && zone.IsDropOff() {
			continue
		}
		if !actor.IsBodyBagged && geometry.DistanceManhattan(location, pos) < 7 {
			return true
		}
	}
	for _, item := range currentMap.Items() {
		pos := item.Pos()
		zone := currentMap.ZoneAt(pos)
		if zone != nil && zone.IsDropOff() {
			continue
		}
		if !item.Buried && item.IsObviousWeapon() && item.WasMoved() && geometry.DistanceManhattan(location, pos) < 7 {
			return true
		}
	}
	return false
}

func NewAIController(engine services.Engine) *AIController {
	ctrl := &AIController{
		engine:               engine,
		travelGroups:         mapset.NewSet[mapset.Set[*core.Actor]](),
		activeInvestigations: mapset.NewSet[string](),
		activeCleanups:       mapset.NewSet[string](),
	}
	// When an alarm fires, alert all guards and push investigation if they aren't aware.
	engine.SubscribeToEvents(services.NewFilter(func(e services.AlarmTriggeredEvent) bool {
		currentMap := engine.GetGame().GetMap()
		for _, actor := range currentMap.Actors() {
			if actor.Type != core.ActorTypeGuard || actor.IsDowned() {
				continue
			}
			ctrl.SetAlerted(actor)
			// Inject knowledge of the sighting if guard doesn't have it
			sighting := actor.AI.Knowledge.LastSightingOfDangerous
			if sighting.Time.IsZero() || sighting.HandledByMe {
				actor.AI.Knowledge.LastSightingOfDangerous = core.IncidentReport{
					Type:     core.ObservationCombatSeen,
					Location: e.SightingLocation,
					Time:     engine.CurrentGameTime(),
				}
				ctrl.SwitchStateBecauseOfNewKnowledge(actor)
			}
		}
		return true
	}))
	return ctrl
}

func (a *AIController) HandleIncident(person *core.Actor, report core.IncidentReport) {
	if !person.IsActive() || person.IsInCombat() || person.IsFrenzied() ||
		person.Status() == core.ActorStatusPanic || person.Status() == core.ActorStatusSnitching {
		return
	}
	if report.Type.NeedsCleanup() && !a.activeCleanups.Contains(report.Hash()) {
		a.PushCleanupForIncident(person, report)
	}
	if report.Type.IsDangerousLocation() {
		a.SetAlerted(person)
		if person.Type == core.ActorTypeGuard {
			if person.Status() != core.ActorStatusInvestigating &&
				a.dangerousInvestigatorCount(report.Hash()) < 3 && !a.activeInvestigations.Contains(report.Hash()) {
				if a.hasActiveAlarm() {
					a.SwitchToInvestigation(person, report)
					a.SwitchToAlarmRun(person, report)
				} else {
					a.SwitchToInvestigation(person, report)
				}
			}
		} else {
			if a.IsGuardAvailable() {
				a.SwitchToSnitch(person)
			} else {
				a.SwitchToPanic(person, []geometry.Point{report.Location})
			}
		}
	} else if report.Type.IsSuspiciousLocation() && !a.activeInvestigations.Contains(report.Hash()) {
		a.SwitchToInvestigation(person, report)
	}
}

func (a *AIController) TryPopScripted(person *core.Actor) {
	// looks error prone to me: What if an investigation state was pushed over the scripted state?
	if _, ok := person.AI.GetState().(*ScriptedState); ok {
		person.AI.PopState()
	}
}
func (a *AIController) MarkAsDone(person *core.Actor, incident core.IncidentReport) {
	if incident.Type.IsContact() {
		person.AI.Knowledge.LastSightingOfDangerous.HandledByMe = true
	}
	println(fmt.Sprintf("%s MARKED AS DONE -> %s", person.DebugDisplayName(), incident.Hash()))
}

func (a *AIController) SwitchToWait(person *core.Actor) {
	a.PushWait(person, nil)
}

func (a *AIController) PushWaitWithStatus(person *core.Actor, _ core.ActorState, until func() bool) {
	a.PushWait(person, until)
}

func (a *AIController) PushWait(person *core.Actor, until func() bool) {
	if person.IsDowned() || person.IsInCombat() {
		return
	}
	a.pushStateTransition(person,
		&Wait{AIContext: AIContext{Engine: a.engine, Person: person}, Until: until})
}

func (a *AIController) PushGoto(person *core.Actor, destination geometry.Point) {
	a.PushGotoWithCall(person, destination, nil)
}

func (a *AIController) PushGotoWithCall(person *core.Actor, destination geometry.Point, callOnArrival func()) {
	if person.IsDowned() || person.IsInCombat() {
		return
	}
	a.pushStateTransition(person,
		&GotoBehaviour{AIContext: AIContext{Engine: a.engine, Person: person}, TargetLocation: destination, CallOnArrival: callOnArrival})
}

func (a *AIController) SwitchToCombat(person *core.Actor, target *core.Actor) {
	if person.IsDowned() || person.IsInCombat() {
		return
	}
	a.SetAlerted(person)
	currentTargetPos := target.Pos()
	a.pushStateTransition(person,
		&CombatMovement{AIContext: AIContext{Engine: a.engine, Person: person}, Target: target, LastKnownPosition: &currentTargetPos})
	println(fmt.Sprintf("%s is now in combat with %s", person.DebugDisplayName(), target.Name))
}

// PushCleanupForIncident pushes a CleanupMovement pre-loaded with the given incident.
// Used when cleanup is stacked alongside a dangerous-sighting investigation (§7).
func (a *AIController) PushCleanupForIncident(person *core.Actor, incident core.IncidentReport) {
	if person.IsDowned() {
		return
	}
	a.activeCleanups.Add(incident.Hash())
	a.pushStateTransition(person,
		&CleanupMovement{AIContext: AIContext{Engine: a.engine, Person: person}, currentIncident: incident})
	println(fmt.Sprintf("%s queued cleanup of %s", person.DebugDisplayName(), incident.Hash()))
}

func (a *AIController) SwitchToScript(target *core.Actor) {
	if target.IsDowned() || target.Status() == core.ActorStatusScripted {
		return
	}
	a.pushStateTransition(target,
		&ScriptedState{AIContext: AIContext{Engine: a.engine, Person: target}})
	println(fmt.Sprintf("%s is now in script mode", target.DebugDisplayName()))
}

func (a *AIController) SwitchToAlarmRun(person *core.Actor, incident core.IncidentReport) {
	if person.IsDowned() || person.IsInCombat() || person.IsInAlarmRun() {
		return
	}
	a.pushStateTransition(person,
		&AlarmRunMovement{AIContext: AIContext{Engine: a.engine, Person: person}, Incident: incident})
	println(fmt.Sprintf("%s is heading to alarm for %v", person.DebugDisplayName(), incident.Hash()))
}

func (a *AIController) SetAlerted(person *core.Actor) {
	if person.AI == nil || person.AI.IsAlerted {
		return
	}
	person.AI.IsAlerted = true
	println(fmt.Sprintf("%s is now ALERTED", person.DebugDisplayName()))
}

// hasActiveAlarm returns true if there is at least one active alarm object on the map.
func (a *AIController) hasActiveAlarm() bool {
	for _, obj := range a.engine.GetGame().GetMap().AllObjects {
		if dev, ok := obj.(services.AlarmDevice); ok && dev.IsActiveAlarm() {
			return true
		}
	}
	return false
}

// dangerousInvestigatorCount counts non-downed guards currently investigating or
// running to an alarm for the given incident hash. Naturally self-corrects on death.
func (a *AIController) dangerousInvestigatorCount(hash string) int {
	count := 0
	for _, actor := range a.engine.GetGame().GetMap().Actors() {
		if actor.IsDowned() || actor.Type != core.ActorTypeGuard {
			continue
		}
		if state, ok := actor.AI.GetState().(*InvestigationMovement); ok {
			if state.Incident.Hash() == hash && state.Incident.Type.IsDangerousLocation() {
				count++
				continue
			}
		}
		if state, ok := actor.AI.GetState().(*AlarmRunMovement); ok {
			if state.Incident.Hash() == hash {
				count++
			}
		}
	}
	return count
}

// isBeingWatched returns true if any active guard is already watching the given actor.
func (a *AIController) isBeingWatched(target *core.Actor) bool {
	for _, actor := range a.engine.GetGame().GetMap().Actors() {
		if actor.IsPlayer() || actor == target || actor.IsDowned() {
			continue
		}
		if state, ok := actor.AI.GetState().(*WatchMovement); ok {
			if state.suspiciousActor == target {
				return true
			}
		}
	}
	return false
}

// shoutKnowledge syncs LastSightingOfDangerous from person to all guards within hearingRange,
// sets their alert flag, and calls SwitchStateBecauseOfNewKnowledge on each.
// Callers must already have checked that a shout is due (e.g. via a time/step gate).
func (a *AIController) shoutKnowledge(person *core.Actor) {
	const hearingRange = 15
	sighting := person.AI.Knowledge.LastSightingOfDangerous
	if sighting.Time.IsZero() {
		return
	}
	for _, other := range a.engine.GetGame().GetMap().Actors() {
		if other == person || other.Type != core.ActorTypeGuard || other.IsDowned() {
			continue
		}
		if geometry.DistanceManhattan(person.Pos(), other.Pos()) > hearingRange {
			continue
		}
		a.TransferKnowledge(person, other)
		a.SetAlerted(other)
		a.SwitchStateBecauseOfNewKnowledge(other)
	}
}

func (a *AIController) SwitchToSnitch(person *core.Actor) {
	if person.IsDowned() || person.Status() == core.ActorStatusSnitching {
		return
	}
	newState := &SnitchMovement{AIContext: AIContext{Engine: a.engine, Person: person}}
	if person.IsInvestigating() {
		person.AI.ReplaceState(newState)
	} else {
		person.AI.PushState(newState)
	}
	a.resetTransitionFields(person)
	println(fmt.Sprintf("%s is now snitching", person.DebugDisplayName()))
}

func (a *AIController) SwitchToPanic(person *core.Actor, dangerLocations []geometry.Point) {
	if person.IsDowned() || person.Status() == core.ActorStatusPanic {
		return
	}
	var threatActor *core.Actor
	for _, danger := range dangerLocations {
		if actorAt, isActorAt := a.engine.GetGame().GetMap().TryGetActorAt(danger); isActorAt {
			threatActor = actorAt
			break
		}
	}
	a.pushStateTransition(person,
		&PanicMovement{AIContext: AIContext{Engine: a.engine, Person: person}, DangerousLocations: dangerLocations, ThreatActor: threatActor})
	println(fmt.Sprintf("%s is now panicking", person.DebugDisplayName()))
}

func (a *AIController) SwitchToInvestigation(person *core.Actor, incidentReport core.IncidentReport) {
	if person.IsDowned() || person.IsInCombat() ||
		person.Status() == core.ActorStatusPanic ||
		person.Status() == core.ActorStatusInvestigating ||
		person.Status() == core.ActorStatusSnitching {
		return
	}
	if a.IsPartOfTravelGroup(person) {
		a.handleSplitOfTravelGroup(person, a.GetTravelGroup(person))
	}
	a.activeInvestigations.Add(incidentReport.Hash())
	a.pushStateTransition(person,
		&InvestigationMovement{AIContext: AIContext{Engine: a.engine, Person: person}, Incident: incidentReport})
	println(fmt.Sprintf("%s is now investigating %v", person.DebugDisplayName(), incidentReport.Hash()))
}

func (a *AIController) UntrackInvestigation(hash string) {
	a.activeInvestigations.Remove(hash)
}

func (a *AIController) UntrackCleanup(hash string) {
	a.activeCleanups.Remove(hash)
}

func (a *AIController) SwitchToSchedule(person *core.Actor) {
	if person.IsDowned() || person.Engrossed || person.IsInCombat() ||
		person.Status() == core.ActorStatusPanic ||
		person.Status() == core.ActorStatusOnSchedule {
		return
	}
	a.setStateTransition(person,
		&ScheduledMovement{AIContext{Engine: a.engine, Person: person}})
	println(fmt.Sprintf("%s is now on schedule", person.DebugDisplayName()))
}

func (a *AIController) SwitchToWatch(person *core.Actor, target *core.Actor, report core.IncidentReport) {
	if person.IsDowned() || person.IsInCombat() ||
		person.Status() == core.ActorStatusPanic ||
		person.Status() == core.ActorStatusWatching ||
		person.Status() == core.ActorStatusSnitching {
		return
	}

	a.pushStateTransition(person,
		&WatchMovement{AIContext: AIContext{Engine: a.engine, Person: person}, suspiciousActor: target, lastKnownLocation: person.Pos(), incident: report})
	println(fmt.Sprintf("%s is now watching %s", person.DebugDisplayName(), target.Name))
}

func (a *AIController) SwitchToGuard(person *core.Actor) {
	if person.IsDowned() {
		return
	}
	a.setStateTransition(person,
		&GuardMovement{AIContext{Engine: a.engine, Person: person}})
	println(fmt.Sprintf("%s is now guarding", person.DebugDisplayName()))
}

func (a *AIController) SwitchToVomit(person *core.Actor) {
	if person.IsDowned() {
		return
	}
	person.IsNauseous = true
	a.pushStateTransition(person,
		&VomitMovement{AIContext: AIContext{Engine: a.engine, Person: person}})
	println(fmt.Sprintf("%s is now vomiting", person.DebugDisplayName()))
}

func (a *AIController) SwitchToFrenzy(person *core.Actor) {
	if person.IsDowned() || person.IsFrenzied() {
		return
	}
	a.pushStateTransition(person,
		&FrenzyMovement{AIContext: AIContext{Engine: a.engine, Person: person}})
	println(fmt.Sprintf("%s is now frenzied", person.DebugDisplayName()))
}

func (a *AIController) IsFollowingActor(follower *core.Actor, leader *core.Actor) bool {
	if follower.AI == nil || follower.AI.GetState() == nil {
		return false
	}
	followerState, ok := follower.AI.GetState().(*FollowerMovement)
	if !ok {
		return false
	}
	return followerState.Leader == leader
}

func (a *AIController) TryContextActionAtTaskLocation(person *core.Actor, finishedCallback func()) bool {
	currentMap := a.engine.GetGame().GetMap()
	for _, neighbor := range currentMap.GetAllCardinalNeighbors(person.Pos()) {
		if currentMap.IsTileWithSpecialAt(neighbor, gridmap.SpecialTileTypeFood) {
			a.ConsumeFoodAt(person, neighbor, finishedCallback)
			return true
		}
	}
	return false
}

func (a *AIController) ConsumeFoodAt(person *core.Actor, foodPos geometry.Point, finishedCallback func()) {
	animator := a.engine.GetAnimator()
	game := a.engine.GetGame()
	currentMap := a.engine.GetGame().GetMap()
	animationCompleted := false
	until := func() bool { return animationCompleted }
	a.SetEngrossed(person, until)
	completed := func() {
		if currentMap.IsStimulusOnTile(foodPos, stimuli.StimulusLethalPoison) {
			game.ApplyStimulusToActor(person, core.NewEffectSourceFromTile(currentMap.CellAt(foodPos).TileType), stimuli.Stim{StimType: stimuli.StimulusLethalPoison, StimForce: 100})
		} else if currentMap.IsStimulusOnTile(foodPos, stimuli.StimulusEmeticPoison) {
			game.ApplyStimulusToActor(person, core.NewEffectSourceFromTile(currentMap.CellAt(foodPos).TileType), stimuli.Stim{StimType: stimuli.StimulusEmeticPoison, StimForce: 100})
		} else if currentMap.IsStimulusOnTile(foodPos, stimuli.StimulusInducedSleep) {
			game.ApplyStimulusToActor(person, core.NewEffectSourceFromTile(currentMap.CellAt(foodPos).TileType), stimuli.Stim{StimType: stimuli.StimulusInducedSleep, StimForce: 100})
		}
		finishedCallback()
		animationCompleted = true
	}
	animator.FoodAnimation(person, foodPos, completed)
}

func (a *AIController) UpdateVision(person *core.Actor) {
	if person.CanPerceive() {
		game := a.engine.GetGame()
		currentMap := game.GetMap()
		currentMap.UpdateFieldOfView(person)
		game.UpdateKnowledgeFromVision(person)
		a.SwitchStateBecauseOfNewKnowledge(person)
	}
}

func (a *AIController) SetEngrossed(person *core.Actor, until func() bool) {
	person.Engrossed = true
	a.engine.ScheduleWhen(until, func() {
		person.Engrossed = false
	})
}

func (a *AIController) StateOf(person *core.Actor) core.AIStateHandler {
	return person.AI.GetState()
}

// resetTransitionFields resets per-tick movement fields after a state transition.
func (a *AIController) resetTransitionFields(person *core.Actor) {
	person.Move = core.AutoMove{}
	person.Path = nil
	person.AI.UpdatePredicate = nil
	person.AI.NextUpdateIn = stateTransitionDelay
}

// pushStateTransition pushes a new state and resets movement fields.
func (a *AIController) pushStateTransition(person *core.Actor, state core.AIStateHandler) {
	person.AI.PushState(state)
	a.resetTransitionFields(person)
}

// setStateTransition replaces the entire stack (irreversible base states e.g. Guard, Schedule).
func (a *AIController) setStateTransition(person *core.Actor, state core.AIStateHandler) {
	person.AI.SetState(state)
	a.resetTransitionFields(person)
}

// PathSet updates the path of Person to a new position.
func (a *AIController) PathSet(person *core.Actor, p geometry.Point, passableFunc func(point geometry.Point) bool) bool {
	currentMap := a.engine.GetGame().GetMap()
	person.Path = currentMap.GetJPSPath(person.Pos(), p, passableFunc, person.Path)
	return len(person.Path) > 1
}

// moveOnPath moves the player to next position in the path, updates the path
// accordingly, and returns a command that will deliver the message for the
// next automatic movement step along the path.
func (a *AIController) MoveOnPath(person *core.Actor) core.AIUpdate {
	if person.Path == nil || len(person.Path) < 2 || !person.CanMove() {
		return a.StopAuto(person)
	}
	p := person.Path[1]
	person.Move.Path = true
	currentMap := a.engine.GetGame().GetMap()
	moveDelta := p.Sub(person.Pos())
	person.LookDirection = geometry.DirectionVectorToAngleInDegrees(moveDelta)
	// All actors reject tiles with visible (unburied) mines.
	mineBlocking := currentMap.IsItemAt(p) && currentMap.ItemAt(p).IsMine() && !currentMap.ItemAt(p).Buried
	if !currentMap.CurrentlyPassableAndSafeForActor(person)(p) || mineBlocking {
		if !currentMap.IsActorAt(p) || (currentMap.ActorAt(p) == nil || !a.IsFollowingActor(currentMap.ActorAt(p), person)) {
			return a.HandleBlockedPath(person, moveDelta)
		}
		blockingActor := currentMap.ActorAt(p)
		if blockingActor != nil && a.IsFollowingActor(blockingActor, person) {
			person.Path = person.Path[1:]
			currentMap.SwapPositions(person, blockingActor)
		}
	}
	ai := person.AI
	if ai != nil {
		ai.PathBlockedCount = 0
	}

	person.Move.Delta = moveDelta
	person.Path = person.Path[1:]
	game := a.engine.GetGame()
	game.MoveActor(person, p)
	return NextUpdateIn(float64(person.MoveDelay()))
}
func (a *AIController) StopAuto(person *core.Actor) core.AIUpdate {
	person.Move = core.AutoMove{}
	person.Path = nil

	return NextUpdateIn(1.0)
}

func (a *AIController) HandleBlockedPath(person *core.Actor, moveDelta geometry.Point) core.AIUpdate {
	ai := person.AI
	ai.PathBlockedCount++
	if ai.PathBlockedCount > rng.R.Intn(3)+2 {
		ai.PathBlockedCount = 0
		return ai.Movement.OnBlockedPath()
	}
	person.Move.Delta = moveDelta
	return NextUpdateIn(float64(person.MoveDelay()))
}
func (a *AIController) IsControlledByAI(person *core.Actor) bool {
	return person.AI != nil
}

// SyncKnowledgeIfDue syncs LastSightingOfDangerous with nearby guards every 35 steps.
func (a *AIController) SyncKnowledgeIfDue(person *core.Actor) {
	if person.Type != core.ActorTypeGuard || person.AI == nil {
		return
	}
	if person.AI.Knowledge.LastSightingOfDangerous.Time.IsZero() {
		return
	}
	if person.StepsTaken%35 != 0 {
		return
	}
	const hearingRange = 15
	for _, other := range a.engine.GetGame().GetMap().Actors() {
		if other == person || other.Type != core.ActorTypeGuard || other.IsDowned() || other.AI == nil {
			continue
		}
		if geometry.DistanceManhattan(person.Pos(), other.Pos()) <= hearingRange {
			a.TransferKnowledge(person, other)
		}
	}
}

func (a *AIController) Update() {
	timeFactor := a.engine.GetTimeFactor()
	if timeFactor <= 0 {
		return // world is frozen; no AI updates
	}
	deltaTime := timeFactor * utils.TicksToSeconds(1)
	for _, person := range a.engine.GetGame().GetMap().Actors() {
		if !person.IsActive() {
			continue
		}
		if a.IsControlledByAI(person) {
			person.AI.NextUpdateIn -= deltaTime
			if person.AI.NextUpdateIn <= 0 && person.AI.IsUpdateAllowed() {
				person.AI.NextUpdateIn = a.UpdateAI(person)
			}
		}
	}
}
func (a *AIController) UpdateAI(person *core.Actor) float64 {
	updateForPerson := a.StateOf(person).NextAction()

	if updateForPerson.DelayInSeconds == core.ManualDeferredUpdate.DelayInSeconds && updateForPerson.UpdatePredicate != nil {
		person.AI.UpdatePredicate = updateForPerson.UpdatePredicate
		return 0
	}
	return updateForPerson.DelayInSeconds
}

// incident types and their priorities
// Unknown           - low       -> investigate
// CombatTraces  	 - medium    -> investigate / snitch
// Suspicious Actor  - high      -> watch
// Hostile Actor     - very high -> combat / snitch

func (a *AIController) SwitchStateBecauseOfNewKnowledge(person *core.Actor) {
	if !person.IsActive() || person.IsInCombat() || person.IsFrenzied() || person.Status() == core.ActorStatusPanic || person.Status() == core.ActorStatusSnitching {
		return
	}
	if a.ReactToDangerousActor(person) {
		return
	}
}

func (a *AIController) ReactToDangerousActor(person *core.Actor) bool {
	contactReport := person.AI.Knowledge.LastSightingOfDangerous
	if contactReport.Time.IsZero() || contactReport.HandledByMe {
		return false
	}
	currentMap := a.engine.GetGame().GetMap()
	dangerMan, isActorHere := currentMap.TryGetActorAt(contactReport.Location)
	if !isActorHere || a.engine.GetGame().AreAllies(person, dangerMan) {
		return false
	}
	if person.CanSeeActor(dangerMan) {
		if person.Type == core.ActorTypeGuard {
			a.SwitchToCombat(person, dangerMan)
		} else if a.IsGuardAvailable() {
			a.SwitchToSnitch(person)
		} else {
			a.SwitchToPanic(person, []geometry.Point{dangerMan.Pos()})
		}
		return true
	} else {
		if person.Type == core.ActorTypeGuard {
			a.SwitchToInvestigation(person, contactReport)
			return true
		} else if a.IsGuardAvailable() {
			a.SwitchToSnitch(person)
			return true
		} else if contactReport.Type.IsOpenViolence() {
			a.SwitchToPanic(person, []geometry.Point{contactReport.Location})
			return true
		}
	}
	return false
}

func (a *AIController) IsGuardAvailable() bool {
	for _, actor := range a.engine.GetGame().GetMap().Actors() {
		if actor.IsAvailableGuard() {
			return true
		}
	}
	return false
}

func (a *AIController) RaiseSuspicionAt(person *core.Actor, dangerousActor *core.Actor, delayInMS int) {
	ai := person.AI
	// Alerted guards fill suspicion 2× faster
	if ai.IsAlerted {
		delayInMS /= 2
		if delayInMS < 100 {
			delayInMS = 100
		}
	}
	now := time.Now()
	person.LookAt(dangerousActor.Pos())
	if ai.LastSuspicionRaised.Add(time.Duration(delayInMS)*time.Millisecond).After(now) && ai.SuspicionCounter > 0 {
		return
	}
	ai.SuspicionCounter++
	ai.LastSuspicionRaised = time.Now()
	println(fmt.Sprintf("%s raised suspicion at %s", person.DebugDisplayName(), dangerousActor.DebugDisplayName()))
	if ai.SuspicionCounter > 3 {
		if dangerousActor.IsPlayer() {
			a.engine.PublishEvent(services.PlayerSpottedEvent{})
		}
		ai.SuspicionCounter = 0
		person.IsEyeWitness = true
		person.AI.Knowledge.AddDangerousSighting(person, dangerousActor, core.ObservationOngoingSuspiciousBehaviour, a.engine.CurrentGameTime())
		if person.Type == core.ActorTypeGuard {
			a.SwitchToCombat(person, dangerousActor)
		} else if a.IsGuardAvailable() {
			a.SwitchToSnitch(person)
		} else {
			a.SwitchToPanic(person, []geometry.Point{dangerousActor.Pos()})
		}
	}
}

func (a *AIController) IsAtGuardPosition(person *core.Actor) bool {
	ai := person.AI
	return person.Pos() == ai.StartPosition
}

func (a *AIController) CalculateAllTaskPaths(actor *core.Actor) {
	nameOfSchedule := actor.AI.Schedule
	schedule := a.engine.GetGame().GetMap().GetSchedule(nameOfSchedule)
	if schedule == nil {
		return
	}
	lastTask := schedule.Tasks[len(schedule.Tasks)-1]
	prevPos := lastTask.Location
	currentMap := a.engine.GetGame().GetMap()
	var nextPos geometry.Point
	for index, task := range schedule.Tasks {
		nextPos = task.Location
		task.KnownPath = currentMap.GetJPSPath(prevPos, nextPos, currentMap.CurrentlyPassableForActor(actor), task.KnownPath)
		schedule.Tasks[index] = task
		prevPos = nextPos
	}
}

func (a *AIController) TaskCountFor(actor *core.Actor) int {
	schedule := a.engine.GetGame().GetMap().GetSchedule(actor.AI.Schedule)
	if schedule == nil {
		return 0
	}
	return len(schedule.Tasks)
}

func (a *AIController) Reset() {
	for _, actor := range a.engine.GetGame().GetMap().Actors() {
		if actor.AI != nil {
			actor.AI.Knowledge = &core.IndividualKnowledge{}
		}
	}
}

func (a *AIController) IsNearGuards(pos geometry.Point) bool {
	game := a.engine.GetGame()
	currentMap := game.GetMap()
	for _, actor := range currentMap.Actors() {
		if actor.Type != core.ActorTypeGuard || !actor.IsActive() || actor.IsInCombat() {
			continue
		}
		distance := geometry.DistanceManhattan(actor.Pos(), pos)
		if distance > 20 {
			continue
		}
		return true
	}
	return false
}

// TransferKnowledge shares dangerous-actor sightings between two guards on the same team.
// Suspicious activities and location incidents are personal and are never shared.
func (a *AIController) TransferKnowledge(one *core.Actor, two *core.Actor) {
	if one.Type != core.ActorTypeGuard || two.Type != core.ActorTypeGuard || one.Team != two.Team {
		return
	}
	oneS := one.AI.Knowledge.LastSightingOfDangerous
	twoS := two.AI.Knowledge.LastSightingOfDangerous
	if oneS.Time.After(twoS.Time) {
		two.AI.Knowledge.LastSightingOfDangerous = oneS
	} else if twoS.Time.After(oneS.Time) {
		one.AI.Knowledge.LastSightingOfDangerous = twoS
	}
	println(fmt.Sprintf("Knowledge transfer: %s <-> %s", one.DebugDisplayName(), two.DebugDisplayName()))
}

func (a *AIController) handleSplitOfTravelGroup(person *core.Actor, group mapset.Set[*core.Actor]) {
	println(fmt.Sprintf("%s split from group of %d", person.Name, group.Cardinality()))
	originalPositionOfLeavingActor := person.Pos()

	leaverHasReturned := false
	groupList := group.ToSlice()
	count := len(groupList) - 1
	if count <= 0 {
		// Person is the only member — nothing to coordinate.
		return
	}
	avgPos := geometry.PointF{}
	for _, other := range groupList {
		if other == person {
			continue
		}
		avgPos = avgPos.Add(other.Pos().ToPointF())
	}
	avgPos = avgPos.Div(float64(count))
	avgGroupPos := avgPos.ToPoint()
	currentMap := a.engine.GetGame().GetMap()
	freeNeighbor := currentMap.GetRandomFreeNeighbor(avgGroupPos)
	if freeNeighbor != originalPositionOfLeavingActor && geometry.DistanceManhattan(avgGroupPos, freeNeighbor) < geometry.DistanceManhattan(avgGroupPos, originalPositionOfLeavingActor) {
		originalPositionOfLeavingActor = freeNeighbor
	}
	for _, other := range groupList {
		if other == person {
			// push "return to group" state
			println(fmt.Sprintf("Scheduling %s returning to group", person.Name))
			a.PushGotoWithCall(person, originalPositionOfLeavingActor, func() { leaverHasReturned = true })
			continue
		}
		// push "wait for person to return" state
		a.PushWait(other, func() bool { return leaverHasReturned })
		println(fmt.Sprintf("%s waiting for %s to return", other.Name, person.Name))
	}
}
