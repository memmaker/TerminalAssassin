package game

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/actions"
	"github.com/memmaker/terminal-assassin/game/ai"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/objects"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/states"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/mapset"
	"github.com/memmaker/terminal-assassin/ui"
	"github.com/memmaker/terminal-assassin/utils"
)

type Model struct {
	engine            services.Engine
	currentGameStates []services.GameState

	MissionStats *core.MissionStats
	missionPlan  *core.MissionPlan

	oldMousePos geometry.Point
	playerPos   geometry.Point
	gridMap     *gridmap.GridMap[*core.Actor, *core.Item, services.Object]

	clearScreen bool

	config  *services.GameConfig
	camera  *geometry.Camera
	actions services.ActionsInterface
}

func (m *Model) GetActions() services.ActionsInterface {
	return m.actions
}

func (m *Model) CurrentGameState() services.GameState {
	return m.currentGameStates[len(m.currentGameStates)-1]
}

func (m *Model) PushGameplayState() {
	m.PushState(&states.GameStateGameplay{})
}

// PushState pushes a new game state onto the stack and will call its Init method with the engine.
func (m *Model) PushState(state services.GameState) {
	userInterface := m.engine.GetUI()
	userInterface.Reset()

	state.Init(m.engine)
	m.currentGameStates = append(m.currentGameStates, state)
	userInterface.SetGamestate(m.CurrentGameState())
}

func (m *Model) PopState() {
	if len(m.currentGameStates) <= 1 {
		return
	}
	userInterface := m.engine.GetUI()
	userInterface.Reset()
	m.currentGameStates = m.currentGameStates[:len(m.currentGameStates)-1]
	userInterface.SetGamestate(m.CurrentGameState())
}

func (m *Model) PopAndInitPrevious() {
	if len(m.currentGameStates) <= 1 {
		return
	}
	userInterface := m.engine.GetUI()
	userInterface.Reset()
	m.currentGameStates = m.currentGameStates[:len(m.currentGameStates)-1]
	m.CurrentGameState().Init(m.engine)
	userInterface.SetGamestate(m.CurrentGameState())
}
func (m *Model) GetCamera() *geometry.Camera {
	return m.camera
}
func (m *Model) GetMissionPlan() *core.MissionPlan {
	return m.missionPlan
}

func (m *Model) ClearMap(width int, height int) {
	m.gridMap = gridmap.NewEmptyMap[*core.Actor, *core.Item, services.Object](width, height, m.engine.GetGame().GetConfig().MaxVisionRange)
	m.gridMap.Fill(*m.engine.GetData().NewEmptyCell())
}

func (m *Model) ResetModel() {
	m.actions.Reset()
	m.ClearMap(m.engine.MapWindowWidth(), m.engine.MapWindowWidth())
	m.MissionStats = core.NewMissionStats()
}

func (m *Model) ResetGameState() {
	m.currentGameStates = []services.GameState{}
	m.PushState(&states.GameStateMainMenu{})
}
func (m *Model) DrawVisionCone(con console.CellInterface, actor *core.Actor) {
	if actor.IsInCombat() {
		return
	}
	player := m.engine.GetGame().GetMap().Player
	var color common.Color
	color = common.Transparent
	susLvl := actor.AI.SuspicionCounter
	if susLvl == 1 {
		color = core.ColorFromCode(core.ColorMapGreen)
	} else if susLvl == 2 {
		color = core.ColorFromCode(core.ColorMapYellow)
	} else if susLvl == 3 {
		color = core.ColorFromCode(core.ColorMapRed)
	} else {
		println("Suspicion level out of range")
		return
	}
	actor.VisionCone(func(worldPos geometry.Point) {
		screenPos := m.GetCamera().WorldToScreen(worldPos)
		if !con.Contains(screenPos) || !player.CanSee(worldPos) {
			return
		}
		cellAt := con.AtSquare(screenPos)
		currBg := cellAt.Style.Background
		if currBg == core.ColorFromCode(core.ColorMapRed) && susLvl < 3 {
			return
		}
		if currBg == core.ColorFromCode(core.ColorMapYellow) && susLvl < 2 {
			return
		}
		//newBackground := currBg.Multiply(color)
		if color == common.Transparent {
			color = currBg
		}
		con.SetSquare(screenPos, cellAt.WithBackgroundColor(color))
	})
}

func (m *Model) CreatePickupAction(forItem *core.Item) services.ContextAction {
	return &PickupAction{item: forItem}
}

func (m *Model) StartDialogueAction(initialSpeaker *core.Actor, dialogueName string) services.ContextAction {
	return &DialogueAction{InitialSpeaker: initialSpeaker, DialogueName: dialogueName}
}

func (m *Model) GetConfig() *services.GameConfig {
	return m.config
}

func (m *Model) GetStats() *core.MissionStats {
	return m.MissionStats
}

func (m *Model) GetMap() *gridmap.GridMap[*core.Actor, *core.Item, services.Object] {
	return m.gridMap
}
func NewModel(config *services.GameConfig) *Model {
	mapWidth := config.GridWidth
	mapHeight := config.GridHeight - ui.HUDHeight
	model := &Model{
		config:       config,
		camera:       geometry.NewCamera(0, 0, mapWidth, mapHeight),
		missionPlan:  &core.MissionPlan{},
		MissionStats: core.NewMissionStats(),
		gridMap:      gridmap.NewEmptyMap[*core.Actor, *core.Item, services.Object](mapWidth, mapHeight, config.MaxVisionRange),
	}

	return model
}

func (m *Model) Init(engine services.Engine) {
	m.engine = engine
	audio := engine.GetAudio()
	files := engine.GetFiles()
	sfxFiles := files.GetFilesInPath("datafiles/sfx")
	randomizedSfxFiles := files.GetSubdirectories("datafiles/sfx")
	ambienceFiles := files.GetFilesInPath("datafiles/ambience")
	musicFiles := files.GetFilesInPath("datafiles/music")

	audio.RegisterSoundCues(sfxFiles)
	audio.RegisterSoundCues(ambienceFiles)
	audio.RegisterSoundCues(musicFiles)
	audio.RegisterRandomizedSoundCues(randomizedSfxFiles)

	m.gridMap.Fill(*engine.GetData().NewEmptyCell())

	m.gridMap.SetMaxLightIntensity(1)
	m.gridMap.SetAmbientLight(gridmap.DefaultAmbientLight)

	m.actions = actions.NewActionProvider(m.engine)

	webMode := engine.GetGame().GetConfig().WebMode
	if webMode {
		career := engine.GetCareer()
		career.PlayerName = "0816"
		career.CurrentCampaignFolder = "first blood" //TODO: check if this is correct
		engine.GetGame().PushState(&states.GameStateMainMenu{})
	} else if engine.GetFiles().FileExists("career.gob") {
		engine.GetGame().PushState(&states.GameStateMainMenu{})
	} else {
		engine.GetGame().PushState(&states.GameStateIntro{})
	}
}

func (m *Model) SendTriggerStimuli(user *core.Actor, usedItem *core.Item, location geometry.Point, trigger core.ItemEffectTrigger) {
	if itemEffect, ok := usedItem.TriggerEffects[trigger]; ok {
		m.Apply(location, core.NewEffectSourceUsedItem(user, usedItem), itemEffect)
		if itemEffect.DestroyOnApplication {
			m.Destroy(usedItem)
		}
	}
}

func (m *Model) ApplyStimulusToItem(i *core.Item, source core.EffectSource, stim stimuli.Stimulus) {
	if i.IsDestroyed {
		return
	}
	if itemEffect, ok := i.ReactionEffects[stim.Type()]; ok {
		if stim.Force() >= itemEffect.ForceThreshold {
			if itemEffect.EffectOnReaction.DestroyOnApplication {
				m.Destroy(i)
			}
			m.ApplyDelayed(i.Pos(), source.WithItem(i), itemEffect.EffectOnReaction, 0.017)
		}
	}
}

func (m *Model) Destroy(i *core.Item) {
	i.IsDestroyed = true
	if i.HeldBy != nil {
		i.MapPos = i.HeldBy.Pos()
		i.HeldBy.Inventory.RemoveItem(i)
		if i.HeldBy.EquippedItem == i {
			i.HeldBy.EquippedItem = nil
		}
	}
	m.GetMap().RemoveItem(i)
}

func (m *Model) ApplyStimulusToTile(atLocation geometry.Point, source core.EffectSource, stim stimuli.Stimulus) {
	currentMap := m.GetMap()
	animator := m.engine.GetAnimator()

	switch stim.Type() {
	case stimuli.StimulusFire:
		m.ApplyFireToTile(atLocation, source, stim)
	case stimuli.StimulusBurnableLiquid:
		m.ApplyBurnableToTile(atLocation, source, stim)
	case stimuli.StimulusWater:
		currentMap.RemoveStimulusFromTile(atLocation, stimuli.StimulusFire)
		currentMap.RemoveStimulusFromTile(atLocation, stimuli.StimulusBurnableLiquid)
		currentMap.AddStimulusToTile(atLocation, stim)
		// todo check if a neighbor has electricity
		electroNeighbor := currentMap.GetNeighborWithStim(atLocation, stimuli.StimulusHighVoltage)
		if electroNeighbor != atLocation {
			electroStim := currentMap.GetStimAt(electroNeighbor, stimuli.StimulusHighVoltage)
			currentMap.PropagateElectroStimFromWaterTileAt(atLocation, electroStim)
		}
	case stimuli.StimulusHighVoltage:
		electroTiles := currentMap.GetConnected(atLocation, func(p geometry.Point) bool {
			return currentMap.IsStimulusOnTile(p, stimuli.StimulusWater)
		})
		if len(electroTiles) == 0 {
			electroTiles = append(electroTiles, atLocation)
		}
		animator.ElectricityAnimation(electroTiles, source, stim)
	}
	return
}
func (m *Model) SoundEventAt(soundLocation geometry.Point, kindOfSound core.Observation, maxDistance int) {
	currentMap := m.GetMap()
	animator := m.engine.GetAnimator()
	aic := m.engine.GetAI()
	soundTiles := currentMap.WavePropagationFrom(soundLocation, maxDistance, 0)
	animator.SoundPropagationAnimation(kindOfSound, soundTiles, func() {
		for _, tiles := range soundTiles {
			for _, p := range tiles {
				actorAt, isActorAt := currentMap.TryGetActorAt(p)
				//areAllies := game.AreAllies(actorAtSource, actorAt)
				if isActorAt && actorAt.CanPerceive() {
					if kindOfSound.IsSpeech() {
						actorAt.TryRespondToSpeech(m.engine.CurrentTick(), string(kindOfSound))
					} else if aic.IsControlledByAI(actorAt) {
						if !actorAt.IsInCombat() {
							actorAt.LookAt(soundLocation)
						}
						aic.ReportIncident(actorAt, soundLocation, kindOfSound)
						aic.SwitchStateBecauseOfNewKnowledge(actorAt)
					}
				}
			}
		}
	})
}
func (m *Model) SuspiciousActionAt(pos geometry.Point, kindOfEvent core.Observation) {
	currentMap := m.GetMap()
	aic := m.engine.GetAI()
	for _, a := range currentMap.Actors() {
		if a == currentMap.Player || (!a.CanUseItems()) || (!a.CanSeeInVisionCone(pos)) || a.IsInvestigating() {
			continue
		}
		aic.ReportIncident(a, pos, kindOfEvent)
	}
}
func (m *Model) IllegalActionAt(pos geometry.Point, kindOfEvent core.Observation) {
	currentMap := m.GetMap()
	aic := m.engine.GetAI()
	actorAt, isActorAt := currentMap.TryGetActorAt(pos)
	playerPos := currentMap.Player.Pos()
	if !isActorAt && geometry.DistanceChebyshev(pos, playerPos) <= 1 {
		actorAt = currentMap.Player
	}
	for _, a := range currentMap.Actors() {
		if a.IsPlayer() || a.IsInCombat() || !a.CanPerceive() || !a.CanSeeInVisionCone(pos) || m.AreAllies(a, actorAt) {
			continue
		}
		if actorAt != nil {
			if actorAt.IsPlayer() && a.CanSeeInVisionCone(actorAt.Pos()) {
				m.engine.GetGame().GetStats().BeenSpotted = true
			}
			if kindOfEvent.IsOpenViolence() {
				a.AI.Knowledge.AddSightingOfDangerousActor(a, actorAt, kindOfEvent, m.engine.CurrentTick())
			} else {
				a.AI.Knowledge.AddSightingOfSuspiciousActor(a, actorAt.Pos(), kindOfEvent, m.engine.CurrentTick())
			}
		} else {
			aic.ReportIncident(a, pos, kindOfEvent)
		}
	}
}
func (m *Model) ApplyStimulusToThings(atLocation geometry.Point, source core.EffectSource, stim stimuli.Stimulus) {
	currentMap := m.GetMap()
	var lastActor *core.Actor
	if currentMap.IsActorAt(atLocation) {
		actorAt := currentMap.ActorAt(atLocation)
		m.ApplyStimulusToActor(actorAt, source, stim)
		lastActor = actorAt
	}
	if currentMap.IsDownedActorAt(atLocation) {
		actorAt := currentMap.DownedActorAt(atLocation)
		if actorAt == lastActor { // don't apply twice
			return
		}
		m.ApplyStimulusToActor(actorAt, source, stim)
	}
	if currentMap.IsObjectAt(atLocation) {
		objectAt := currentMap.ObjectAt(atLocation)
		m.ApplyStimulusToObject(objectAt, stim)
	}
	if currentMap.IsItemAt(atLocation) {
		itemAt := currentMap.ItemAt(atLocation)
		m.ApplyStimulusToItem(itemAt, source, stim)
	}
}

func (m *Model) ApplyFireToTile(atLocation geometry.Point, source core.EffectSource, stim stimuli.Stimulus) {
	gridMap := m.GetMap()
	// also emit a delayed fire stim to all burnable neighbors
	if gridMap.IsStimulusOnTile(atLocation, stimuli.StimulusBurnableLiquid) {
		stim = stim.WithForce(80) // spark to flames..
		gridMap.AddStimulusToTile(atLocation, stim)
		for _, n := range gridMap.GetFilteredCardinalNeighbors(atLocation, func(p geometry.Point) bool {
			return gridMap.IsStimulusOnTile(p, stimuli.StimulusBurnableLiquid) && !gridMap.IsStimulusOnTile(p, stimuli.StimulusFire)
		}) {
			m.ApplyDelayed(n, source, stimuli.StimEffect{Stimuli: []stimuli.Stimulus{stim}}, rand.Float64()*0.5)
		}
	}
}

func (m *Model) createDistributedStimulus(se stimuli.StimEffect, source core.EffectSource, atLocation geometry.Point) {
	animator := m.engine.GetAnimator()
	audio := m.engine.GetAudio()
	switch se.Distribution {
	case stimuli.DistributeExplode:
		audio.PlayCue("small_explosion")
		m.SoundEventAt(atLocation, core.ObservationExplosion, se.Pressure)
		animator.BlastDistribution(atLocation, source, se.Stimuli, se.Distance, se.Pressure)
	case stimuli.DistributeLiquid:
		animator.LiquidDistribution(atLocation, source, se.Stimuli, se.Distance)
	}
}

func (m *Model) ApplyDelayed(pos geometry.Point, source core.EffectSource, effect stimuli.StimEffect, delay float64) {
	m.engine.Schedule(delay, func() { m.Apply(pos, source, effect) })
}

func (m *Model) Apply(atLocation geometry.Point, source core.EffectSource, effects stimuli.StimEffect) {
	if effects.Distribution != stimuli.DistributeDirect {
		m.createDistributedStimulus(effects, source, atLocation)
		return
	}
	for _, stimulus := range effects.Stimuli {
		m.ApplyStimulusToThings(atLocation, source, stimulus)
		m.ApplyStimulusToTile(atLocation, source, stimulus)
	}
}

func (m *Model) ApplyBurnableToTile(atLocation geometry.Point, source core.EffectSource, stim stimuli.Stimulus) {
	currentMap := m.GetMap()
	// if any neighbor is on fire, emit a delayed fire stim to this tile
	if !currentMap.IsStimulusOnTile(atLocation, stimuli.StimulusWater) {
		currentMap.AddStimulusToTile(atLocation, stim)
		for _, n := range currentMap.GetFilteredCardinalNeighbors(atLocation, func(p geometry.Point) bool {
			return currentMap.IsStimulusOnTile(p, stimuli.StimulusFire)
		}) {
			m.engine.Schedule(rand.Float64()*0.5, func() {
				fireForce := currentMap.ForceOfStimulusOnTile(n, stimuli.StimulusFire)
				currentMap.AddStimulusToTile(n, stimuli.Stim{StimType: stimuli.StimulusFire, StimForce: fireForce})
			})

			m.ApplyDelayed(atLocation, source, stimuli.StimEffect{Stimuli: []stimuli.Stimulus{stimuli.Stim{StimType: stimuli.StimulusFire, StimForce: currentMap.ForceOfStimulusOnTile(n, stimuli.StimulusFire)}}}, rand.Float64()*0.5)
			break
		}
	}
}

func (m *Model) ApplyStimulusToActor(a *core.Actor, source core.EffectSource, stim stimuli.Stimulus) {
	damageTaken := true
	switch stim.Type() {
	case stimuli.StimulusPiercingDamage:
		m.TakePiercingDamage(a, source, stim.Force())
	case stimuli.StimulusBluntDamage:
		m.TakeBluntDamage(a, source, stim.Force())
	case stimuli.StimulusFire:
		m.TakeFireDamage(a, source, stim.Force())
	case stimuli.StimulusLethalPoison:
		m.TakeLethalPoisonDamage(a, source, stim.Force())
	case stimuli.StimulusEmeticPoison:
		m.TakeEmeticPoisonDamage(a, source, stim.Force())
	case stimuli.StimulusExplosionDamage:
		m.TakeExplosionDamage(a, source, stim.Force())
	case stimuli.StimulusInducedSleep:
		m.TakeInducedSleep(a, source, stim.Force())
	case stimuli.StimulusHighVoltage:
		m.TakeElectricalDamage(a, source, stim.Force())
	default:
		damageTaken = false
	}
	if damageTaken {
		a.AddDamage(stim.Force(), stim.Type())
		//m.DisEngage(a)
		if m.engine.GetGame().GetMap().Player == a {
			m.UpdateHUD()
		}
	}
}

// Cause of Death for people being shot is sneak attack most of the time..
// We also don't know if a Sniper Rifle was used..
func (m *Model) TakePiercingDamage(a *core.Actor, source core.EffectSource, force int) {
	if m.engine.GetAI().IsControlledByAI(a) && (a.Status != core.ActorStatusCombat) {
		m.Kill(a, core.NewCauseOfDeathFromStim(stimuli.StimulusPiercingDamage, source))
		return
	}
	damageInPoints := int(3.0 * (float64(force) / 100.0))
	if a.DiesFromDamage(damageInPoints) {
		m.Kill(a, core.NewCauseOfDeathFromStim(stimuli.StimulusPiercingDamage, source))
		return
	}
	m.IllegalActionAt(a.Pos(), core.ObservationCombatSeen)
}

func (m *Model) TakeBluntDamage(a *core.Actor, source core.EffectSource, force int) {
	if a.IsDowned() || force > 75 {
		m.Kill(a, core.NewCauseOfDeathFromStim(stimuli.StimulusBluntDamage, source))
	} else {
		m.SendToSleep(a)
	}
}

func (m *Model) TakeFireDamage(a *core.Actor, source core.EffectSource, force int) {
	if force < 20 {
		return
	}
	if a.DiesFromDamage(1) {
		m.Kill(a, core.NewCauseOfDeathFromStim(stimuli.StimulusFire, source))
	} else {
		m.engine.Schedule(1.25, func() {
			m.checkActorOnBurningTile(a)
		})
	}
}

func (m *Model) TakeLethalPoisonDamage(a *core.Actor, source core.EffectSource, force int) {
	randomDelay := rand.Float64() * 5
	m.engine.Schedule(randomDelay, func() {
		m.Kill(a, core.NewCauseOfDeathFromStim(stimuli.StimulusLethalPoison, source))
	})
}

func (m *Model) TakeEmeticPoisonDamage(a *core.Actor, source core.EffectSource, force int) {
	randomDelay := rand.Float64() * 10
	m.engine.Schedule(randomDelay, func() {
		m.engine.GetAI().SwitchToVomit(a)
	})
}

func (m *Model) TakeExplosionDamage(a *core.Actor, source core.EffectSource, force int) {
	m.Kill(a, core.NewCauseOfDeathFromStim(stimuli.StimulusExplosionDamage, source))
}

func (m *Model) TakeInducedSleep(a *core.Actor, source core.EffectSource, force int) {
	m.SendToSleep(a)
}

func (m *Model) TakeElectricalDamage(a *core.Actor, source core.EffectSource, force int) {
	if force > 50 {
		m.Kill(a, core.NewCauseOfDeathFromStim(stimuli.StimulusHighVoltage, source))
	} else {
		m.SendToSleep(a)
	}
}

func ActorMovedOrIncapacitated(person *core.Actor) func() bool {
	currentPos := person.Pos()
	return func() bool {
		return person.Pos() != currentPos || person.IsDowned()
	}
}

func ActorMovedOrStateChanged(person *core.Actor) func() bool {
	currentPos := person.Pos()
	currentStatus := person.Status
	return func() bool {
		result := person.Pos() != currentPos || person.Status != currentStatus
		return result
	}
}

func SoundStopped(handle services.AudioHandle) func() bool {
	return func() bool {
		return !handle.IsPlaying()
	}
}
func (m *Model) KillAndRemove(victim *core.Actor, causeOfDeath core.CauseOfDeath) {
	if victim.Status == core.ActorStatusDead {
		return
	}
	defer m.UpdateHUD()
	victim.Status = core.ActorStatusDead
	victim.IsEyeWitness = false

	println("KILLED and REMOVED -> " + victim.DebugDisplayName())
	player := m.GetMap().Player
	if victim == player {
		m.EndMissionWithFailure(causeOfDeath)
		return
	} else {
		m.GetMap().SetActorToRemoved(victim)
		m.MissionStats.AddKill(victim, causeOfDeath, victim.Pos(), utils.UTicksToSeconds(m.engine.CurrentTick()))
		return
	}
}

func (m *Model) RemoveDeadActor(victim *core.Actor) {
	if victim.Status != core.ActorStatusDead {
		return
	}
	defer m.UpdateHUD()
	println("REMOVED -> " + victim.DebugDisplayName())
	m.GetMap().SetActorToRemoved(victim)
}
func (m *Model) Kill(victim *core.Actor, causeOfDeath core.CauseOfDeath) {
	if victim.Status == core.ActorStatusDead {
		return
	}
	defer m.UpdateHUD()
	victim.Status = core.ActorStatusDead
	victim.IsEyeWitness = false

	m.IllegalActionAt(victim.Pos(), core.ObservationDeath)

	m.DropInventory(victim)
	println("KILLED -> " + victim.DebugDisplayName())
	player := m.GetMap().Player
	if victim == player {
		m.EndMissionWithFailure(causeOfDeath)
		return
	} else {
		m.GetMap().SetActorToDowned(victim)
		m.MissionStats.AddKill(victim, causeOfDeath, victim.Pos(), utils.UTicksToSeconds(m.engine.CurrentTick()))
		return
	}
}

func (m *Model) TryPushActorInDirection(person *core.Actor, targetDirection geometry.Point) {
	pos := person.Pos()
	locationBehindTarget := pos.Add(targetDirection)
	currentMap := m.GetMap()
	if !currentMap.IsWalkable(locationBehindTarget) {
		return
	}
	if person.IsActive() && !currentMap.IsActorAt(locationBehindTarget) {
		m.MoveActor(person, locationBehindTarget)
	} else if !person.IsActive() && !currentMap.IsDownedActorAt(locationBehindTarget) {
		m.MoveActor(person, locationBehindTarget)
	}
}
func (m *Model) DropInventory(a *core.Actor) {
	if len(a.Inventory.Items) == 0 {
		return
	}
	dropPositions := m.getItemDistributionPositions(a.Pos(), a.Inventory.Items)
	for i, item := range a.Inventory.Items {
		if i >= len(dropPositions) {
			println("ERROR: not enough free cells to drop all items")
			break
		}
		m.dropInventoryItemAt(a, item, dropPositions[i])
	}
	a.Inventory.Items = []*core.Item{}
}
func (m *Model) DropFromInventory(person *core.Actor, items []*core.Item) {
	dropPositions := m.getItemDistributionPositions(person.Pos(), items)
	for i, item := range items {
		if i >= len(dropPositions) {
			println("ERROR: not enough free cells to drop all items")
			break
		}
		m.dropInventoryItemAt(person, item, dropPositions[i])
	}

}
func (m *Model) getItemDistributionPositions(position geometry.Point, items []*core.Item) []geometry.Point {
	neededCells := len(items)
	freePredicate := func(p geometry.Point) bool {
		return m.GetMap().IsCurrentlyPassable(p) && !m.GetMap().IsItemAt(p)
	}
	freeCells := m.GetMap().GetFreeCellsForDistribution(position, neededCells, freePredicate)
	return freeCells
}

func (m *Model) MoveItemTo(position geometry.Point, item *core.Item) {
	m.GetMap().MoveItem(item, position)
	m.itemEnteredCell(item, position)
}

func (m *Model) itemEnteredCell(item *core.Item, newPosition geometry.Point) {
	m.ReEmitStimuliOnTileToThings(core.NewEffectSourceFromItem(item), newPosition)
}

func (m *Model) ReEmitStimuliOnTileToThings(source core.EffectSource, p geometry.Point) {
	for _, stim := range m.GetMap().CellAt(p).Stimuli {
		m.ApplyStimulusToThings(p, source, stim)
	}
}

func (m *Model) SendToSleep(a *core.Actor) {
	if a == nil || a.Status == core.ActorStatusDead || a.Status == core.ActorStatusSleeping {
		return
	}
	m.IllegalActionAt(a.Pos(), core.ObservationUnconscious)
	currentMap := m.GetMap()
	animator := m.engine.GetAnimator()
	a.Status = core.ActorStatusSleeping
	a.Fov.Visibles = nil
	if a == currentMap.Player {
		animator.SleepingAnimation(a, func() {
			currentMap.UpdateFieldOfView(a)
			a.Status = core.ActorStatusIdle
		})
	} else {
		currentMap.SetActorToDowned(a)
		m.DropInventory(a)
	}
}

// MoveActor will carry out the movement on the map and also the update the FoVinDegrees.
// It can return effects generated from entering a cell.
func (m *Model) MoveActor(person *core.Actor, to geometry.Point) {
	oldPos := person.Pos()
	currentMap := m.GetMap()

	if person.IsDowned() {
		currentMap.MoveDownedActor(person, to)
	} else {
		currentMap.MoveActor(person, to)
	}

	m.ActorEnteredCell(person, oldPos, to)

}

func (m *Model) ActorEnteredCell(person *core.Actor, oldPosition geometry.Point, newPosition geometry.Point) {
	aic := m.engine.GetAI()
	currentMap := m.GetMap()
	defer m.UpdateHUD()

	newCell := currentMap.CellAt(newPosition)
	if newCell.TileType.IsLethal() {
		m.handleDeathTile(person, newCell)
		return
	}

	if person.EquippedItem != nil {
		m.SendTriggerStimuli(person, person.EquippedItem, newPosition, core.TriggerOnTakenToNewCell)
	}
	if person.IsBleeding() && !person.IsBodyBagged && rand.Float64() < 0.1 {
		currentMap.AddStimulusToTile(person.Pos(), stimuli.Stim{StimType: stimuli.StimulusBlood, StimForce: 5})
	}
	if person == currentMap.Player {
		m.playerEnteredCell(oldPosition, newPosition)
		m.updateVisionForAll(newPosition)
		m.gridMap.UpdateFieldOfView(person)
	} else {
		aic.UpdateVision(person)
	}
	m.handleDragging(person, oldPosition)
	m.ReEmitStimuliOnTileToThings(core.NewEffectSourceFromActor(person), newPosition)

	oldZone := currentMap.ZoneAt(oldPosition)
	newZone := currentMap.ZoneAt(newPosition)

	if oldZone != newZone {
		m.engine.PublishEvent(services.ActorEnteredZoneEvent{Actor: person, OldPosition: oldPosition, NewPosition: newPosition, OldZone: oldZone, NewZone: newZone})
	}
}

func (m *Model) handleDeathTile(person *core.Actor, deathCell gridmap.MapCell[*core.Actor, *core.Item, services.Object]) {
	// TODO: play a small death animation..
	animationCompleted := false
	aic := m.engine.GetAI()
	aic.SetEngaged(person, core.ActorStatusEngaged, func() bool { return animationCompleted })
	animator := m.engine.GetAnimator()
	animator.FallingAnimation(person.Pos(), func() {
		if person.IsDead() {
			m.RemoveDeadActor(person)
		} else {
			m.KillAndRemove(person, core.NewCauseOfDeathFromEnvironment(core.CoDFalling, deathCell.TileType))
		}
	})
}

func (m *Model) playerEnteredCell(oldPosition geometry.Point, newPosition geometry.Point) {
	m.tryCleaning(oldPosition)
	m.tryItemSneakingStimuli(oldPosition)
	m.tryAdaptingAmbienceSounds(oldPosition, newPosition)
}

func (m *Model) tryAdaptingAmbienceSounds(oldPosition geometry.Point, newPosition geometry.Point) {
	audio := m.engine.GetAudio()
	currentMap := m.GetMap()
	oldZone := currentMap.ZoneAt(oldPosition)
	newZone := currentMap.ZoneAt(newPosition)
	if oldZone != newZone {
		if oldZone.AmbienceCue != "" {
			audio.Stop(oldZone.AmbienceCue)
		} else if currentMap.AmbienceSoundCue != "" && newZone.AmbienceCue != "" {
			audio.Stop(currentMap.AmbienceSoundCue)
		}
		if newZone.AmbienceCue != "" {
			audio.StartLoop(newZone.AmbienceCue)
		} else if currentMap.AmbienceSoundCue != "" && !audio.IsCuePlaying(currentMap.AmbienceSoundCue) {
			audio.StartLoop(currentMap.AmbienceSoundCue)
		}
	}
}
func (m *Model) handleDragging(dragger *core.Actor, oldPosition geometry.Point) {
	model := m
	currentMap := model.GetMap()

	if !dragger.IsDraggingBody() && !currentMap.IsDownedActorAt(oldPosition) {
		return
	}

	isSneaking := dragger.MovementMode == core.MovementModeSneaking
	isRunning := dragger.MovementMode == core.MovementModeRunning
	noBigItem := dragger.EquippedItem == nil || !dragger.EquippedItem.IsBig

	if isRunning && dragger.IsDraggingBody() { // stop dragging
		dragger.DraggedBody = nil
		return
	}

	actorAt, isActorAt := currentMap.TryGetDownedActorAt(oldPosition)
	onTopOfBody := false
	if isActorAt {
		onTopOfBody = true
		if isSneaking && noBigItem && onTopOfBody && !dragger.IsDraggingBody() { // start dragging
			dragger.DraggedBody = actorAt
		}
	}

	if dragger.IsDraggingBody() {
		oldPosOfBody := dragger.DraggedBody.Pos()
		newPosOfBody := oldPosition
		if currentMap.IsDownedActorAt(newPosOfBody) {
			bodyInTheWay := currentMap.DownedActorAt(newPosOfBody)
			draggedBody := dragger.DraggedBody
			currentMap.SwapDownedPositions(draggedBody, bodyInTheWay)
		} else {
			currentMap.MoveDownedActor(dragger.DraggedBody, newPosOfBody)
		}
		model.ActorEnteredCell(dragger.DraggedBody, oldPosOfBody, newPosOfBody)
	}
}

func (m *Model) tryCleaning(position geometry.Point) {
	currentMap := m.GetMap()
	hasCleanerEquipped := currentMap.Player.EquippedItem != nil && currentMap.Player.EquippedItem.Type == core.ItemTypeCleaner
	isSneaking := currentMap.Player.MovementMode == core.MovementModeSneaking
	isNotDragging := !currentMap.Player.IsDraggingBody()
	if hasCleanerEquipped && isSneaking && isNotDragging {
		currentMap.RemoveStimulusFromTile(position, stimuli.StimulusBurnableLiquid)
		currentMap.RemoveStimulusFromTile(position, stimuli.StimulusWater)
		currentMap.RemoveStimulusFromTile(position, stimuli.StimulusBlood)
	}
}

func (m *Model) tryItemSneakingStimuli(position geometry.Point) {
	currentMap := m.GetMap()
	player := currentMap.Player
	isSneaking := player.MovementMode == core.MovementModeSneaking
	isDragging := player.IsDraggingBody()

	if player.EquippedItem == nil || !isSneaking || isDragging {
		return
	}
	m.SendTriggerStimuli(player, player.EquippedItem, position, core.TriggerOnSneakingWithItem)
}

func (m *Model) updateVisionForAll(position geometry.Point) {
	currentMap := m.GetMap()
	aic := m.engine.GetAI()
	for _, actor := range currentMap.Actors() {
		if aic.IsControlledByAI(actor) && actor.CanSeeInVisionCone(position) {
			aic.UpdateVision(actor)
		}
	}
}

func (m *Model) IsUnlockedDoorAt(p geometry.Point) bool {
	currentMap := m.GetMap()
	if !currentMap.Contains(p) {
		return false
	}
	if obj, ok := currentMap.TryGetObjectAt(p); ok {
		door, isDoor := obj.(*objects.Door)
		return isDoor && door.State != objects.DoorStateLocked
	}
	return false
}

func (m *Model) IsDoorAt(p geometry.Point) bool {
	currentMap := m.GetMap()
	if !currentMap.Contains(p) {
		return false
	}
	if obj, ok := currentMap.TryGetObjectAt(p); ok {
		_, isDoor := obj.(*objects.Door)
		return isDoor
	}
	return false
}
func (m *Model) PickUpItem(person *core.Actor) {
	actorWantsToPickupAt := person.Pos().Add(person.FovShiftForPeeking)
	itemHere := m.GetMap().ItemAt(actorWantsToPickupAt)
	if itemHere == nil {
		return
	}

	defer m.UpdateHUD()

	if itemHere.InsteadOfPickup != nil {
		itemHere.InsteadOfPickup(person, itemHere)
		return
	}

	// NEVER PICK UP CLOTHING ITEMS
	if itemHere.Type == core.ItemTypeClothing {
		m.InsteadOfPickingUpClothes(person, itemHere)
		return
	}
	equippedItem := person.EquippedItem
	bigItemInHands := equippedItem != nil && equippedItem.IsBig
	if itemHere.IsBig && bigItemInHands {
		defer m.forceDropInventoryItem(person, equippedItem, actorWantsToPickupAt)
	}

	m.GetMap().RemoveItem(itemHere)

	person.Inventory.AddItem(itemHere)
	itemHere.HeldBy = person
	if itemHere.IsBig || !bigItemInHands {
		person.EquippedItem = itemHere
	}
	m.engine.PublishEvent(services.ItemPickedUpEvent{Item: itemHere, Actor: person})

}

func (m *Model) InsteadOfPickingUpClothes(actor *core.Actor, item *core.Item) {
	animator := m.engine.GetAnimator()

	clothesFromItem := core.Clothing{Name: item.Name, FgColor: item.DefinedStyle.Foreground.ToHSV(), BgColor: item.DefinedStyle.Background.ToHSV()}
	aic := m.engine.GetAI()
	animationCompleted := false
	until := func() bool {
		return animationCompleted
	}
	aic.SetEngaged(actor, core.ActorStatusEngaged, until)
	completed := func() {
		animationCompleted = true
		clothesForItem := actor.Clothes.Name
		fgForItem := actor.Clothes.FgColor
		bgForItem := actor.Clothes.BgColor
		actor.Clothes = clothesFromItem

		item.Name = clothesForItem
		item.DefinedStyle = common.Style{Foreground: fgForItem, Background: bgForItem}
		m.UpdateHUD()
	}
	cancelled := func() {
		animationCompleted = true
	}
	animator.PlayerChangeClothesAnimation(item.Pos(), clothesFromItem, completed, cancelled)
	return
}
func (m *Model) DropEquippedItem(person *core.Actor) {
	m.dropInventoryItemAt(person, person.EquippedItem, person.Pos().Add(person.FovShiftForPeeking))
	person.EquippedItem = nil
}

func (m *Model) dropInventoryItemAt(person *core.Actor, itemToDrop *core.Item, location geometry.Point) {
	droppedAtPos := m.GetValidItemPlacementPosition(location, itemToDrop)
	m.forceDropInventoryItem(person, itemToDrop, droppedAtPos)
}

func (m *Model) forceDropInventoryItem(person *core.Actor, itemToDrop *core.Item, droppedAtPos geometry.Point) {
	if itemToDrop == nil {
		return
	}
	person.Inventory.RemoveItem(itemToDrop)
	m.forceDropItemAt(itemToDrop, droppedAtPos)
}

func (m *Model) forceDropItemAt(itemToDrop *core.Item, droppedAtPos geometry.Point) {
	m.MoveItemTo(droppedAtPos, itemToDrop)
	m.SendTriggerStimuli(nil, itemToDrop, droppedAtPos, core.TriggerOnItemDropped)
}
func (m *Model) GetValidItemPlacementPosition(pos geometry.Point, item *core.Item) geometry.Point {
	currentMap := m.GetMap()
	if !currentMap.IsItemAt(pos) && !currentMap.IsActorAt(pos) && currentMap.IsWalkable(pos) {
		return pos
	} else {
		cellsBeingUsed := m.getItemDistributionPositions(pos, []*core.Item{item})
		return cellsBeingUsed[0]
	}
}
func (m *Model) PlaceItem(pos geometry.Point, item *core.Item) geometry.Point {
	validPos := m.GetValidItemPlacementPosition(pos, item)
	m.forceDropItemAt(item, validPos)
	return validPos
}
func (m *Model) PlaceItemWithOrigin(origin, pos geometry.Point, item *core.Item) geometry.Point {
	currentMap := m.GetMap()
	freeForItem := func(p geometry.Point) bool {
		return currentMap.IsWalkable(p) && !currentMap.IsItemAt(p)
	}
	if freeForItem(pos) {
		m.MoveItemTo(pos, item)
		return pos
	} else if (geometry.DistanceManhattan(origin, pos) == 1 || geometry.DistanceManhattan(origin, pos) == 2) && freeForItem(origin) {
		m.MoveItemTo(origin, item)
		return origin
	} else {
		freeCells := currentMap.GetFilteredNeighbors(pos, freeForItem)
		sort.Slice(freeCells, func(i, j int) bool {
			return geometry.DistanceManhattan(origin, freeCells[i]) < geometry.DistanceManhattan(origin, freeCells[j])
		})
		if len(freeCells) > 0 {
			m.MoveItemTo(freeCells[0], item)
			return freeCells[0]
		}
	}
	println(fmt.Sprintf("WARNING: could not place item %s at %v. Item was destroyed.", item.Name, pos))
	m.Destroy(item)
	return pos
}

func (m *Model) SnapNeck(actingPerson, victim *core.Actor) {
	if victim == actingPerson.DraggedBody {
		actingPerson.DraggedBody = nil
	}
	m.Kill(victim, core.NewCauseOfDeath(core.CoDBrokenNeck, actingPerson))
}

func (m *Model) UpdateHUD() {
	if gs, ok := m.CurrentGameState().(*states.GameStateGameplay); ok {
		gs.UpdateHUD()
	}
}

func (m *Model) IllegalPlayerEngagementWithActorAtPos(position geometry.Point, icon rune, engagementFinishedAction func(), engagementCancelledAction func()) {
	const InteractionTime = 4
	animator := m.engine.GetAnimator()
	currentMap := m.GetMap()
	actorAt := currentMap.ActorAt(position)

	aic := m.engine.GetAI()
	animationCompleted := false
	until := func() bool {
		return animationCompleted
	}
	aic.SetEngaged(actorAt, core.ActorStatusVictimOfEngagement, until)
	aic.SetEngaged(currentMap.Player, core.ActorStatusEngagedIllegal, until)

	completed := func() {
		animationCompleted = true
		engagementFinishedAction()
	}
	// cancel gets triggered, meaning the victim was downed or moved..
	cancelled := func() {
		animationCompleted = true
		if engagementCancelledAction != nil {
			engagementCancelledAction()
		}
	}
	animator.ActorEngagedIllegalAnimation(actorAt, icon, position, InteractionTime, completed, cancelled)
}

func (m *Model) ApplyStimulusToObject(object services.Object, stim stimuli.Stimulus) {
	object.ApplyStimulus(m.engine, stim)
}

func (m *Model) SwitchClothesWith(taker *core.Actor, provider *core.Actor) {
	animator := m.engine.GetAnimator()
	aic := m.engine.GetAI()
	animationCompleted := false
	until := func() bool {
		return animationCompleted
	}
	aic.SetEngaged(taker, core.ActorStatusEngaged, until)
	onChangingFinished := func() {
		animationCompleted = true
		clothesToSpawn := taker.Clothes
		taker.Clothes = provider.Clothes
		provider.Clothes = m.engine.GetData().NoClothing()
		m.SpawnClothingItem(taker.Pos(), clothesToSpawn)
		m.UpdateHUD()
	}
	onChangingCancelled := func() {
		animationCompleted = true
	}
	animator.PlayerChangeClothesAnimation(provider.Pos(), provider.Clothes, onChangingFinished, onChangingCancelled)
}

func (m *Model) SpawnClothingItem(pos geometry.Point, clothing core.Clothing) {
	newClothes := &core.Item{Name: clothing.Name, DefinedIcon: core.GlyphClothing, Type: core.ItemTypeClothing, DefinedStyle: common.Style{Foreground: clothing.FgColor, Background: clothing.BgColor}}
	spawnPos := m.GetValidItemPlacementPosition(pos, newClothes)
	m.forceDropItemAt(newClothes, spawnPos)
}

var actorActions = []services.ContextAction{
	PianoWire{},
	DrownAction{},
	PushOverEdge{},
	MeleeTakedown{},
}

var downedActorActions = []services.ContextAction{
	SnapNeckAction{},
	ChangeClothesAction{},
}

var tileActions = map[gridmap.SpecialTileType]services.ContextAction{
	gridmap.SpecialTilePlayerExit:      ExitAction{},
	gridmap.SpecialTileTypeFood:        PoisonAction{},
	gridmap.SpecialTileTypePowerOutlet: ExposeElectricityAction{},
}

func (m *Model) GetContextActionAt(position geometry.Point) services.ContextAction {
	model := m
	currentMap := model.GetMap()
	cellAt := currentMap.CellAt(position)
	player := currentMap.Player
	var action services.ContextAction

	// for actors, we check the standard actions
	if currentMap.IsActorAt(position) {
		for _, possibleAction := range actorActions {
			if possibleAction.IsActionPossible(m.engine, player, position) {
				return possibleAction
			}
		}
	}

	// for downed actors, we check the standard actions
	if currentMap.IsDownedActorAt(position) {
		for _, possibleAction := range downedActorActions {
			if possibleAction.IsActionPossible(m.engine, player, position) {
				return possibleAction
			}
		}
	}
	// for objects, we just wrap them in an action
	if currentMap.IsObjectAt(position) {
		objectAt := currentMap.ObjectAt(position)
		if objectAt.IsActionAllowed(m.engine, player) {
			return objects.NewAction(objectAt)
		}
	}
	// for tiles, we check the pre-defined standard actions
	if tileAction, ok := tileActions[cellAt.TileType.Special]; ok {
		if tileAction.IsActionPossible(m.engine, player, position) {
			return tileAction
		}
	}

	return action
}

func (m *Model) UpdateKnowledgeFromVision(person *core.Actor) {
	a := person.AI
	left, right := geometry.GetLeftAndRightBorderOfVisionCone(person.Pos(), person.LookDirection, person.FoVinDegrees)
	currentMap := m.GetMap()
	visionRangeSquared := person.VisionRange() * person.VisionRange()
	person.Fov.IterSSC(func(p geometry.Point) {
		if geometry.DistanceSquared(person.Pos(), p) > visionRangeSquared || (!geometry.InVisionCone(person.Pos(), p, left, right)) {
			return
		}

		m.createIncidentsForSuspiciousActivity(person, p)

		actorAt, isActorAt := currentMap.TryGetActorAt(p)
		if !isActorAt || !actorAt.IsVisible() || m.AreAllies(person, actorAt) || !actorAt.IsActive() {
			return
		}

		dangerObservation := m.GetDangerObservation(person, actorAt)
		if dangerObservation != core.ObservationNull {
			person.IsEyeWitness = true
			a.Knowledge.AddSightingOfDangerousActor(person, actorAt, dangerObservation, m.engine.CurrentTick())
			if actorAt.IsPlayer() {
				m.GetStats().BeenSpotted = true
			}
		} else {
			suspicionObservation := m.GetSuspicionObservation(person, actorAt)
			if suspicionObservation != core.ObservationNull {
				a.Knowledge.AddSightingOfSuspiciousActor(person, p, suspicionObservation, m.engine.CurrentTick())
				if actorAt.IsPlayer() {
					m.GetStats().BeenSpotted = true
				}
			}
		}
	})
}

func (m *Model) GetSuspicionObservation(person *core.Actor, susActor *core.Actor) core.Observation {
	currentMap := m.GetMap()
	aic := m.engine.GetAI()

	if susActor.HasIllegalItemEquipped() {
		return core.ObservationOpenCarry
	}
	if susActor.IsEngagedInIllegalAction() {
		return core.ObservationIllegalAction
	}
	if currentMap.IsInHostileZone(susActor) {
		return core.ObservationTrespassingInHostileZone
	}
	if aic.IsNearActiveIllegalIncident(person, susActor.Pos()) {
		return core.ObservationNearActiveIllegalIncident
	}
	if currentMap.IsTrespassing(susActor) {
		return core.ObservationTrespassing
	}

	return core.ObservationNull
}

func (m *Model) GetDangerObservation(person *core.Actor, dangerActor *core.Actor) core.Observation {

	if dangerActor.IsDraggingBody() {
		return core.ObservationDraggingBody
	}

	if dangerActor.IsInCombat() {
		return core.ObservationCombatSeen
	}

	if person.AI.Knowledge.CompromisedDisguises.Contains(dangerActor.NameOfClothing()) {
		return core.ObservationWearingCompromisedDisguise
	}

	return core.ObservationNull
}

func (m *Model) createIncidentsForSuspiciousActivity(person *core.Actor, p geometry.Point) {
	stats := m.GetStats()
	currentMap := m.GetMap()
	aic := m.engine.GetAI()

	isDownedActorHere := currentMap.IsDownedActorAt(p)
	if isDownedActorHere {
		downedActorAt := currentMap.DownedActorAt(p)
		if !downedActorAt.IsBodyBagged {
			stats.BodiesFound = true // this is redundant now..
			aic.ReportIncident(person, p, core.ObservationBodyFound)
		}
	}

	if currentMap.IsItemAt(p) {
		itemAt := currentMap.ItemAt(p)
		if itemAt.IsObviousWeapon() && itemAt.WasMoved() {
			aic.ReportIncident(person, p, core.ObservationWeaponFound)
		}
	}

	if currentMap.IsStimulusOnTile(p, stimuli.StimulusBlood) {
		aic.ReportIncident(person, p, core.ObservationBloodFound)
	}
}

func (m *Model) UpdateAllFoVsFrom(position geometry.Point) {
	currentMap := m.GetMap()
	for _, actor := range currentMap.Actors() {
		if actor.CanSee(position) && actor.CanPerceive() {
			currentMap.UpdateFieldOfView(actor)
		}
	}
}

// InitLoadedMap initializes the actors that were loaded from a saved map file.
// should be called once after loading a map file.
func (m *Model) InitLoadedMap(loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object]) {
	m.gridMap = loadedMap
	// set start positions for all items
	for _, i := range m.gridMap.Items() {
		i.StartPosition = i.MapPos
	}
	// init all actors
	for _, a := range m.gridMap.Actors() {
		m.InitActor(a)
	}
	// set ambient light from time of day
	m.gridMap.SetAmbientLight(common.GetAmbientLightFromDayTime(m.gridMap.TimeOfDay).ToRGB())
}

func (m *Model) InitActor(a *core.Actor) {
	a.Fov = geometry.NewFOV(geometry.NewRect(-a.MaxVisionRange, -a.MaxVisionRange, a.MaxVisionRange+1, a.MaxVisionRange+1))
	a.Health = m.engine.GetGame().GetConfig().ActorDefaultHealth
	a.MovementMode = core.MovementModeWalking
	if a.AI != nil {
		startZone := m.gridMap.ZoneAt(a.Pos())
		startZone.AllowedClothing.Add(a.NameOfClothing())
		a.AI.Knowledge = &core.IndividualKnowledge{
			CompromisedDisguises: mapset.NewSet[string](),
		}
		a.AI.StartPosition = a.Pos()
		a.AI.StartLookDirection = a.LookDirection
		a.AI.Movement = &actions.Movement{Person: a, Engine: m.engine}
		if a.AI.HasTasks() {
			a.AI.SetState(&ai.ScheduledMovement{AIContext: ai.AIContext{Engine: m.engine, Person: a}})
		} else if !a.IsFollowing() {
			a.AI.SetState(&ai.GuardMovement{AIContext: ai.AIContext{Engine: m.engine, Person: a}})
		}
		//a.AI.stateStack.OnMapLoad(m) // TODO?
		a.FoVinDegrees = 90
	}
	m.gridMap.UpdateFieldOfView(a)
}

func (m *Model) AreAllies(actorOne, actorTwo *core.Actor) bool {
	if actorOne == nil || actorTwo == nil || actorOne.AI == nil || actorTwo.AI == nil {
		return false
	}
	if actorOne == actorTwo {
		return true
	}
	clothesOne := actorOne.NameOfClothing()
	clothesTwo := actorTwo.NameOfClothing()

	if clothesOne == clothesTwo {
		return true
	}

	startZoneOne := m.gridMap.ZoneAt(actorOne.AI.StartPosition)
	startZoneTwo := m.gridMap.ZoneAt(actorTwo.AI.StartPosition)

	return startZoneOne == startZoneTwo || startZoneOne.AllowedClothing.Contains(clothesTwo) || startZoneTwo.AllowedClothing.Contains(clothesOne)
}

func (m *Model) checkActorOnBurningTile(actor *core.Actor) {
	currentMap := m.GetMap()
	if currentMap.IsStimulusOnTile(actor.Pos(), stimuli.StimulusFire) {
		actor.AddDamage(1, stimuli.StimulusFire)
		if actor.DiesFromDamage(1) {
			m.Kill(actor, core.NewCauseOfDeath(core.CoDBurned, nil))
		} else {
			if currentMap.Player == actor {
				m.UpdateHUD()
			}
			m.engine.Schedule(1.25, func() {
				m.checkActorOnBurningTile(actor)
			})
		}
	}
}

func (m *Model) EndMissionWithFailure(cod core.CauseOfDeath) {
	m.engine.Schedule(0, func() {
		m.StopEverything()
		m.PushState(&states.GameStateGameOver{MissionExitedWithGoalCompletion: false, CauseOfPlayerDeath: cod})
	})
}
func (m *Model) EndMissionWithSuccess() {
	m.engine.Schedule(0, func() {
		m.StopEverything()
		m.PushState(&states.GameStateGameOver{MissionExitedWithGoalCompletion: true})
	})
}

func (m *Model) StopEverything() {
	animations := m.engine.GetAnimator()
	animations.Reset()

	audio := m.engine.GetAudio()
	audio.StopAll()

	actionsManager := m.engine.GetGame().GetActions()
	actionsManager.Reset()

	userInterface := m.engine.GetUI()
	userInterface.Reset()
}

func (m *Model) IsOnScreen(worldPosition geometry.Point) bool {
	camera := m.GetCamera()
	return camera.ViewPort.Contains(worldPosition)
}
