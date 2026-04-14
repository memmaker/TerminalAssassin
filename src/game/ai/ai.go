package ai

import (
    "fmt"
    "math/rand"
    "time"

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

type AIController struct {
    engine       services.Engine
    blackboard   *Blackboard
    travelGroups mapset.Set[mapset.Set[*core.Actor]]
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

func (a *AIController) IsNearActiveIllegalIncident(person *core.Actor, location geometry.Point) bool {
    for _, report := range a.blackboard.Filter(func(report core.IncidentReport) bool {
        return report.KnownBy.Contains(person)
    }) {
        if report.Type.IsIllegal() && geometry.DistanceManhattan(location, report.Location) < 7 {
            return true
        }
    }
    return false
}
func (a *AIController) IsNearActiveDangerousIndicator(location geometry.Point) bool {
    for _, report := range a.blackboard.ReportedIncidents {
        if report.Type.IsDangerousLocation() && geometry.DistanceManhattan(location, report.Location) < 5 {
            return true
        }
    }
    return false
}

func NewAIController(engine services.Engine) *AIController {
    return &AIController{
        engine:       engine,
        blackboard:   NewBlackboard(),
        travelGroups: mapset.NewSet[mapset.Set[*core.Actor]](),
    }
}

func NewBlackboard() *Blackboard {
    return &Blackboard{
        ReportedIncidents: make(map[string]core.IncidentReport),
    }
}
func (a *AIController) ReportIncident(person *core.Actor, location geometry.Point, incidentType core.Observation) core.IncidentReport {
    report := a.NewIncidentReport(incidentType, location)
    a.blackboard.AddKnowledge(person, report)
    return report
}
func (a *AIController) NewIncidentReport(typeOfIncident core.Observation, location geometry.Point) core.IncidentReport {
    report := core.IncidentReport{
        Type:     typeOfIncident,
        Location: location,
        Tick:     a.engine.CurrentTick(),
        KnownBy:  mapset.NewSet[*core.Actor](),
    }
    return report
}

// TryRegisterHandler returns true if the incident was not handled before,
// and marks it as handled.
func (a *AIController) TryRegisterHandler(person *core.Actor, incident core.IncidentReport) bool {
    a.blackboard.AddKnowledge(person, incident)
    report := a.blackboard.ReportedIncidents[incident.Hash()]
    if report.FinishedHandling || report.RegisteredHandler != nil {
        return false
    }
    report.RegisteredHandler = person
    a.blackboard.ReportedIncidents[incident.Hash()] = report
    println(fmt.Sprintf("%s is now handling %s", a.blackboard.ReportedIncidents[incident.Hash()].RegisteredHandler.DebugDisplayName(), incident.Hash()))
    return true
}

func (a *AIController) GetIncidentForSnitching(person *core.Actor) core.IncidentReport {
    incident := a.blackboard.GetNextIncidentForSnitching(person)
    if incident == core.EmptyReport {
        return incident
    }
    incident.RegisteredSnitch = person
    a.blackboard.ReportedIncidents[incident.Hash()] = incident
    return incident
}
func (a *AIController) IsIncidentHandled(person *core.Actor, incident core.IncidentReport) bool {
    report := a.blackboard.ReportedIncidents[incident.Hash()]
    return report.FinishedHandling || (report.RegisteredHandler != person && report.HasActiveHandler())
}

func (a *AIController) reportIsInMyZone(person *core.Actor, report core.IncidentReport) bool {
    currentMap := a.engine.GetGame().GetMap()
    myHomeZone := currentMap.ZoneAt(person.AI.StartPosition)
    reportZone := currentMap.ZoneAt(report.Location)
    return myHomeZone == reportZone
}
func (a *AIController) cleanupFilter(person *core.Actor) func(core.IncidentReport) bool {
    currentMap := a.engine.GetGame().GetMap()
    hasDropOff := currentMap.HasDropOffZone()
    return func(report core.IncidentReport) bool {
        if report.Type == core.ObservationBodyFound && !hasDropOff {
            return false
        }
        return a.reportIsInMyZone(person, report)
    }
}

func (a *AIController) GetIncidentForCleanup(person *core.Actor) core.IncidentReport {
    incident := a.blackboard.GetNextIncidentForCleanup(a.cleanupFilter(person))
    if incident == core.EmptyReport {
        return incident
    }
    incident.RegisteredCleaner = person
    a.blackboard.ReportedIncidents[incident.Hash()] = incident
    return incident
}
func (a *AIController) IncidentsNeedCleanup(person *core.Actor) bool {
    return a.blackboard.IncidentsNeedCleanup(a.cleanupFilter(person))
}

func (a *AIController) TryPopScripted(person *core.Actor) {
    if _, ok := person.AI.GetState().(*ScriptedState); ok {
        person.AI.PopState()
    }
}
func (a *AIController) MarkAsDone(incident core.IncidentReport) {
    if report, ok := a.blackboard.ReportedIncidents[incident.Hash()]; ok {
        report.FinishedHandling = true
        a.blackboard.ReportedIncidents[incident.Hash()] = report
        println(fmt.Sprintf("%s MARKED AS DONE -> %s", report.RegisteredHandler.DebugDisplayName(), incident.Hash()))
    }
}
func (a *AIController) MarkAsCleaned(incident core.IncidentReport) {
    a.blackboard.RemoveIncidentReport(incident)
}
func (a *AIController) SwitchToWait(person *core.Actor) {
    a.PushWaitWithStatus(person, person.Status, nil)
}

func (a *AIController) PushWaitWithStatus(person *core.Actor, status core.ActorState, until func() bool) {
    if person.IsDowned() || person.IsInCombat() {
        return
    }
    a.pushStateTransition(person, status,
        &WaitMovement{AIContext: AIContext{Engine: a.engine, Person: person}, Until: until})
}

func (a *AIController) PushGoto(person *core.Actor, destination geometry.Point) {
    a.PushGotoWithCall(person, destination, nil)
}

func (a *AIController) PushGotoWithCall(person *core.Actor, destination geometry.Point, callOnArrival func()) {
    if person.IsDowned() || person.IsInCombat() {
        return
    }
    a.pushStateTransition(person, person.Status,
        &GotoBehaviour{AIContext: AIContext{Engine: a.engine, Person: person}, TargetLocation: destination, CallOnArrival: callOnArrival})
}

func (a *AIController) SwitchToCombat(person *core.Actor, target *core.Actor) {
    if person.IsDowned() || person.IsInCombat() {
        return
    }
    currentTargetPos := target.Pos()
    a.pushStateTransition(person, core.ActorStatusCombat,
        &CombatMovement{AIContext: AIContext{Engine: a.engine, Person: person}, Target: target, LastKnownPosition: &currentTargetPos})
    println(fmt.Sprintf("%s is now in combat with %s", person.DebugDisplayName(), target.Name))
}

func (a *AIController) SwitchToCleanup(person *core.Actor) {
    if person.IsDowned() {
        return
    }
    a.pushStateTransition(person, core.ActorStatusCleanup,
        &CleanupMovement{AIContext: AIContext{Engine: a.engine, Person: person}})
    println(fmt.Sprintf("%s is now cleaning up", person.DebugDisplayName()))
}

func (a *AIController) SwitchToScript(target *core.Actor) {
    if target.IsDowned() || target.Status == core.ActorStatusScripted {
        return
    }
    a.pushStateTransition(target, core.ActorStatusScripted,
        &ScriptedState{AIContext: AIContext{Engine: a.engine, Person: target}})
    println(fmt.Sprintf("%s is now in script mode", target.DebugDisplayName()))
}

func (a *AIController) SwitchToSnitch(person *core.Actor) {
    if person.IsDowned() || person.Status == core.ActorStatusSnitching {
        return
    }
    ai := person.AI
    newState := &SnitchMovement{AIContext: AIContext{Engine: a.engine, Person: person}}
    if person.IsInvestigating() {
        ai.ReplaceState(newState)
    } else {
        ai.PushState(newState)
    }
    person.Status = core.ActorStatusSnitching
    person.Move = core.AutoMove{}
    person.Path = nil
    ai.UpdatePredicate = nil
    ai.NextUpdateIn = stateTransitionDelay
    println(fmt.Sprintf("%s is now snitching", person.DebugDisplayName()))
}

func (a *AIController) SwitchToPanic(person *core.Actor, dangerLocations []geometry.Point) {
    if person.IsDowned() || person.Status == core.ActorStatusPanic {
        return
    }
    var threatActor *core.Actor
    for _, danger := range dangerLocations {
        if actorAt, isActorAt := a.engine.GetGame().GetMap().TryGetActorAt(danger); isActorAt {
            threatActor = actorAt
            break
        }
    }
    a.pushStateTransition(person, core.ActorStatusPanic,
        &PanicMovement{AIContext: AIContext{Engine: a.engine, Person: person}, DangerousLocations: dangerLocations, ThreatActor: threatActor})
    println(fmt.Sprintf("%s is now panicking", person.DebugDisplayName()))
}

func (a *AIController) SwitchToInvestigation(person *core.Actor, incidentReport core.IncidentReport) {
    if person.IsDowned() || person.IsInCombat() ||
        person.Status == core.ActorStatusPanic ||
        person.Status == core.ActorStatusInvestigating ||
        person.Status == core.ActorStatusSnitching {
        return
    }
    if a.IsPartOfTravelGroup(person) {
        a.handleSplitOfTravelGroup(person, a.GetTravelGroup(person))
    }
    a.pushStateTransition(person, core.ActorStatusInvestigating,
        &InvestigationMovement{AIContext: AIContext{Engine: a.engine, Person: person}, Incident: incidentReport})
    println(fmt.Sprintf("%s is now investigating %v", person.DebugDisplayName(), incidentReport.Hash()))
}

func (a *AIController) SwitchToSchedule(person *core.Actor) {
    if person.IsDowned() || person.IsEngaged() || person.IsInCombat() ||
        person.Status == core.ActorStatusPanic ||
        person.Status == core.ActorStatusOnSchedule {
        return
    }
    a.setStateTransition(person, core.ActorStatusOnSchedule,
        &ScheduledMovement{AIContext{Engine: a.engine, Person: person}})
    println(fmt.Sprintf("%s is now on schedule", person.DebugDisplayName()))
}

func (a *AIController) SwitchToWatch(person *core.Actor, target *core.Actor, report core.IncidentReport) {
    if person.IsDowned() || person.IsInCombat() ||
        person.Status == core.ActorStatusPanic ||
        person.Status == core.ActorStatusWatching {
        return
    }
    a.pushStateTransition(person, core.ActorStatusWatching,
        &WatchMovement{AIContext: AIContext{Engine: a.engine, Person: person}, suspiciousActor: target, lastKnownLocation: person.Pos(), incident: report})
    println(fmt.Sprintf("%s is now watching %s", person.DebugDisplayName(), target.Name))
}

func (a *AIController) SwitchToGuard(person *core.Actor) {
    if person.IsDowned() {
        return
    }
    a.setStateTransition(person, core.ActorStatusIdle,
        &GuardMovement{AIContext{Engine: a.engine, Person: person}})
    println(fmt.Sprintf("%s is now guarding", person.DebugDisplayName()))
}

func (a *AIController) SwitchToVomit(person *core.Actor) {
    if person.IsDowned() {
        return
    }
    person.IsNauseous = true
    a.pushStateTransition(person, person.Status,
        &VomitMovement{AIContext: AIContext{Engine: a.engine, Person: person}})
    println(fmt.Sprintf("%s is now vomiting", person.DebugDisplayName()))
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
    a.SetEngaged(person, core.ActorStatusEngaged, until)
    completed := func() {
        a.SetEngaged(person, core.ActorStatusIdle, nil)
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
func (a *AIController) SetEngaged(person *core.Actor, engagedStatus core.ActorState, until func() bool) {
    if person.IsPlayer() {
        person.Status = engagedStatus
        a.engine.ScheduleWhen(until, func() {
            person.Status = core.ActorStatusIdle
        })
    } else { //ActorStatusVictimOfEngagement
        if person.DebugFlag {
            println(fmt.Sprintf("%s: New Engaged Status: %s", person.DebugDisplayName(), engagedStatus))
        }
        if engagedStatus == core.ActorStatusEngaged {
            a.PushWaitWithStatus(person, engagedStatus, until) // push wait state
        } else if engagedStatus == core.ActorStatusEngagedIllegal {
            a.PushWaitWithStatus(person, engagedStatus, until) // push wait state
        } else if engagedStatus == core.ActorStatusVictimOfEngagement {
            a.PushWaitWithStatus(person, engagedStatus, until) // push wait state
        }
    }
}
func (a *AIController) StateOf(person *core.Actor) core.AIStateHandler {
    return person.AI.GetState()
}

// pushStateTransition is the shared tail of every SwitchTo* / PushXxx method.
// It sets the actor status, pushes the new state onto the stack, and resets
// all per-tick movement fields.
func (a *AIController) pushStateTransition(person *core.Actor, status core.ActorState, state core.AIStateHandler) {
    person.Status = status
    person.AI.PushState(state)
    person.Move = core.AutoMove{}
    person.Path = nil
    person.AI.UpdatePredicate = nil
    person.AI.NextUpdateIn = stateTransitionDelay
}

// setStateTransition is like pushStateTransition but replaces the entire stack
// (used when the new state is an irreversible base state, e.g. Guard, Schedule).
func (a *AIController) setStateTransition(person *core.Actor, status core.ActorState, state core.AIStateHandler) {
    person.Status = status
    person.AI.SetState(state)
    person.Move = core.AutoMove{}
    person.Path = nil
    person.AI.UpdatePredicate = nil
    person.AI.NextUpdateIn = stateTransitionDelay
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
    // Default-state actors also reject tiles with visible (unburied) mines.
    mineBlocking := person.IsInDefaultState() && currentMap.IsItemAt(p) && currentMap.ItemAt(p).IsMine() && !currentMap.ItemAt(p).Buried
    if !currentMap.CurrentlyPassableAndSafeForActor(person)(p) || mineBlocking {
        blockingActor := currentMap.ActorAt(p)
        if blockingActor != nil && a.IsFollowingActor(blockingActor, person) {
            person.Path = person.Path[1:]
            currentMap.SwapPositions(person, blockingActor)
        } else {
            return a.HandleBlockedPath(person, moveDelta)
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
    if ai.PathBlockedCount > rand.Intn(3)+2 {
        ai.PathBlockedCount = 0
        return ai.Movement.OnBlockedPath()
    }
    person.Move.Delta = moveDelta
    return NextUpdateIn(float64(person.MoveDelay()))
}
func (a *AIController) IsControlledByAI(person *core.Actor) bool {
    return person.AI != nil
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
    if !person.IsActive() || person.IsInCombat() || person.Status == core.ActorStatusPanic || person.Status == core.ActorStatusSnitching {
        return
    }
    if a.ReactToDangerousActor(person) {
        return
    }
    if a.ReactToSuspiciousActor(person) {
        return
    }
    if person.Status == core.ActorStatusWatching {
        return
    }
    investigatingDangerousLocation := false
    investigatingSuspiciousLocation := false
    if investigationState, ok := person.AI.GetState().(*InvestigationMovement); ok {
        if investigationState.Incident != core.EmptyReport && investigationState.Incident.RegisteredHandler == person && !investigationState.Incident.FinishedHandling {
            investigatingDangerousLocation = investigationState.Incident.Type.IsDangerousLocation()
            investigatingSuspiciousLocation = investigationState.Incident.Type.IsSuspiciousLocation()
        }
    }
    if investigatingDangerousLocation {
        return
    }
    if a.ReactToDangerousLocations(person) {
        return
    }
    if investigatingSuspiciousLocation {
        return
    }
    if a.ReactToSuspiciousLocations(person) {
        return
    }
    person.AI.LowerSuspicion()
}

func (a *AIController) ReactToDangerousActor(person *core.Actor) bool {
    contactReport := person.AI.Knowledge.LastSightingOfDangerousActor
    if contactReport.Tick > 0 && !contactReport.FinishedHandling {
        currentMap := a.engine.GetGame().GetMap()
        dangerMan, isActorHere := currentMap.TryGetActorAt(contactReport.Location)
        if isActorHere && person.AI.Knowledge.CompromisedDisguises.Contains(dangerMan.NameOfClothing()) && person.CanSeeActor(dangerMan) {
            //a.RaiseSuspicionAt(person, dangerMan, 200)
            if person.Type == core.ActorTypeGuard {
                a.SwitchToCombat(person, dangerMan)
            } else if !contactReport.IsKnownByGuards() && a.IsGuardAvailable() {
                a.SwitchToSnitch(person)
            } else {
                a.SwitchToPanic(person, []geometry.Point{dangerMan.Pos()})
            }
            return true
        } else {
            if person.Type == core.ActorTypeGuard {
                a.SwitchToInvestigation(person, contactReport)
            } else if !contactReport.IsKnownByGuards() && a.IsGuardAvailable() {
                a.SwitchToSnitch(person)
            } else if contactReport.Type.IsOpenViolence() {
                a.SwitchToPanic(person, []geometry.Point{contactReport.Location})
            }
        }
    }
    return false
}

func (a *AIController) ReactToSuspiciousActor(person *core.Actor) bool {
    contactReport := person.AI.Knowledge.LastSightingOfSuspiciousActor
    if contactReport.Tick > 0 {
        currentMap := a.engine.GetGame().GetMap()
        susMan, isActorHere := currentMap.TryGetActorAt(contactReport.Location)
        if isActorHere && !a.engine.GetGame().AreAllies(person, susMan) {
            a.SwitchToWatch(person, susMan, contactReport)
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
func (a *AIController) GetDangerousIncidents(person *core.Actor) []core.IncidentReport {
    return a.blackboard.Filter(func(report core.IncidentReport) bool {
        return report.Type.IsDangerousLocation()
    })
}

// PROBLEM: guard is handling one incident and switches to investigate
// but the guard will aso happily acquire another incident
func (a *AIController) ReactToDangerousLocations(person *core.Actor) bool {
    incident, exists := a.blackboard.LastUnhandledReport(func(report core.IncidentReport) bool {
        return report.Type.IsDangerousLocation() && report.KnownBy.Contains(person) && (person.Type == core.ActorTypeGuard || report.RegisteredSnitch == nil)
    })
    if exists {
        if person.Type == core.ActorTypeGuard && a.TryRegisterHandler(person, incident) {
            incident.RegisteredHandler = person
            a.SwitchToInvestigation(person, incident)
        } else if !incident.IsKnownByGuards() && a.IsGuardAvailable() {
            a.SwitchToSnitch(person)
        }
        return true
    }
    return false
}
func (a *AIController) ReactToSuspiciousLocations(person *core.Actor) bool {
    incident, exists := a.blackboard.LastUnhandledReport(func(report core.IncidentReport) bool {
        return report.Type.IsSuspiciousLocation() && report.KnownBy.Contains(person) && (person.Type == core.ActorTypeGuard || !a.IsNearActiveDangerousIndicator(report.Location))
    })
    if exists && a.TryRegisterHandler(person, incident) {
        incident.RegisteredHandler = person
        a.SwitchToInvestigation(person, incident)
        return true
    }
    return false
}
func (a *AIController) RaiseSuspicionAt(person *core.Actor, dangerousActor *core.Actor, delayInMS int) {
    ai := person.AI
    now := time.Now()
    person.LookAt(dangerousActor.Pos())
    if ai.LastSuspicionRaised.Add(time.Duration(delayInMS)*time.Millisecond).After(now) && ai.SuspicionCounter > 0 {
        return
    }
    ai.SuspicionCounter++
    println(fmt.Sprintf("%s raised suspicion at %s", person.DebugDisplayName(), dangerousActor.DebugDisplayName()))
    if ai.SuspicionCounter > 3 {
        if dangerousActor.IsPlayer() {
            a.engine.PublishEvent(services.PlayerSpottedEvent{})
        }
        ai.SuspicionCounter = 0
        person.IsEyeWitness = true
        person.AI.Knowledge.AddSightingOfDangerousActor(person, dangerousActor, core.ObservationOngoingSuspiciousBehaviour, a.engine.CurrentTick())
        if person.Type == core.ActorTypeGuard {
            a.SwitchToCombat(person, dangerousActor)
        } else if a.IsGuardAvailable() {
            a.SwitchToSnitch(person)
        } else {
            a.SwitchToPanic(person, []geometry.Point{dangerousActor.Pos()})
        }
    }
    ai.LastSuspicionRaised = time.Now()
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
    a.blackboard = NewBlackboard()
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

func (a *AIController) TransferKnowledge(one *core.Actor, two *core.Actor) {
    oneDisguises := one.AI.Knowledge.CompromisedDisguises.ToSlice()
    twoDisguises := two.AI.Knowledge.CompromisedDisguises.ToSlice()
    for _, disguise := range oneDisguises {
        two.AI.Knowledge.CompromisedDisguises.Add(disguise)
    }
    for _, disguise := range twoDisguises {
        one.AI.Knowledge.CompromisedDisguises.Add(disguise)
    }
    if one.AI.Knowledge.LastSightingOfDangerousActor.Tick > 0 && one.AI.Knowledge.LastSightingOfDangerousActor.Tick > two.AI.Knowledge.LastSightingOfDangerousActor.Tick {
        lastSighting := one.AI.Knowledge.LastSightingOfDangerousActor
        lastSighting.KnownBy.Add(one)
        lastSighting.KnownBy.Add(two)
        two.AI.Knowledge.LastSightingOfDangerousActor = lastSighting
    } else if two.AI.Knowledge.LastSightingOfDangerousActor.Tick > 0 {
        lastSighting := two.AI.Knowledge.LastSightingOfDangerousActor
        lastSighting.KnownBy.Add(one)
        lastSighting.KnownBy.Add(two)
        one.AI.Knowledge.LastSightingOfDangerousActor = lastSighting
    }
    a.blackboard.TransferKnowledge(one, two)
    println(fmt.Sprintf("Blackboard: Knowledge transfer between %s and %s", one.DebugDisplayName(), two.DebugDisplayName()))
}

func (a *AIController) handleSplitOfTravelGroup(person *core.Actor, group mapset.Set[*core.Actor]) {
    println(fmt.Sprintf("%s split from group of %d", person.Name, group.Cardinality()))
    originalPositionOfLeavingActor := person.Pos()

    leaverHasReturned := false
    groupList := group.ToSlice()
    avgPos := geometry.PointF{}
    count := len(groupList) - 1
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
        a.PushWaitWithStatus(other, core.ActorStatusIdle, func() bool { return leaverHasReturned })
        println(fmt.Sprintf("%s waiting for %s to return", other.Name, person.Name))
    }
}
