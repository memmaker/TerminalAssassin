package game

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

type TargetedAction func(m services.Engine, source *core.Actor, target geometry.Point)
type TargetedPlayerAction func(m services.Engine, target geometry.Point)

type PickupAction struct {
	item *core.Item
}

func (p PickupAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	return m.GetGame().GetMap().IsItemAt(actionAt)
}

func (p PickupAction) Description(m services.Engine, person *core.Actor, pos geometry.Point) (rune, common.Style) {
	style := common.DefaultStyle.WithBg(common.LegalActionGreen)
	if !p.item.IsLegalForActor(person) {
		style = style.WithBg(common.IllegalActionRed)
	}
	return p.item.Icon(), style
}

func (p PickupAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	m.GetGame().PickUpItem(person)
}

type OverflowAction struct{}

func (a OverflowAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphWater, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

func (a OverflowAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	m.GetGame().GetMap().AddStimulusToTile(position, stimuli.Stim{StimType: stimuli.StimulusWater, StimForce: 100})
	m.GetGame().SoundEventAt(position, core.ObservationStrangeNoiseHeard, 10)
	m.GetGame().Apply(position, core.NewEffectSourceFromActor(person), stimuli.EffectLeak(stimuli.StimulusWater, 10, 6))
}

func (a OverflowAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	return !m.GetGame().GetMap().IsStimulusOnTile(actionAt, stimuli.StimulusWater)
}

type ExitAction struct{}

func (a ExitAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphExit, common.DefaultStyle.WithBg(common.LegalActionGreen)
}

func (a ExitAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	m.GetGame().EndMissionWithSuccess()
}

func (a ExitAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	for _, inLevel := range m.GetGame().GetMap().Actors() {
		if inLevel.Type == core.ActorTypeTarget && inLevel.Status != core.ActorStatusDead {
			return false
		}
	}
	for _, inLevel := range m.GetGame().GetMap().DownedActors() {
		if inLevel.Type == core.ActorTypeTarget && inLevel.Status != core.ActorStatusDead {
			return false
		}
	}
	return true
}

type ExposeElectricityAction struct {
	Icon rune
}

func (e ExposeElectricityAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return e.Icon, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

func (e ExposeElectricityAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	return !m.GetGame().GetMap().IsStimulusOnTile(actionAt, stimuli.StimulusHighVoltage)
}

func (e ExposeElectricityAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	stim := stimuli.Stim{StimType: stimuli.StimulusHighVoltage, StimForce: 100}
	m.GetGame().GetMap().AddStimulusToTile(position, stim)
	m.GetGame().IllegalActionAt(position, core.ObservationIllegalAction)
	waterNeighbor := m.GetGame().GetMap().GetNeighborWithStim(position, stimuli.StimulusWater)
	if waterNeighbor != position {
		m.GetGame().GetMap().PropagateElectroStimFromWaterTileAt(waterNeighbor, stim)
	}
}

type PoisonAction struct {
	Icon rune
}

func (a PoisonAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	if !a.hasPoison(m) {
		return
	}
	currentMap := m.GetGame().GetMap()
	poisonItem := currentMap.Player.EquippedItem
	currentMap.Player.EquippedItem = nil
	currentMap.Player.Inventory.RemoveItem(poisonItem)
	switch poisonItem.Type {
	case core.ItemTypeEmeticPoison:
		currentMap.AddStimulusToTile(position, stimuli.Stim{StimType: stimuli.StimulusEmeticPoison, StimForce: 10})
	case core.ItemTypeLethalPoison:
		currentMap.AddStimulusToTile(position, stimuli.Stim{StimType: stimuli.StimulusLethalPoison, StimForce: 10})
	}
	m.GetGame().IllegalActionAt(position, core.ObservationIllegalAction)
}

func (a PoisonAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	return a.hasPoison(m) &&
		!m.GetGame().GetMap().IsStimulusOnTile(actionAt, stimuli.StimulusLethalPoison) &&
		!m.GetGame().GetMap().IsStimulusOnTile(actionAt, stimuli.StimulusEmeticPoison)
}

func (a PoisonAction) hasPoison(m services.Engine) bool {
	poisonItem := m.GetGame().GetMap().Player.EquippedItem
	hasPoison := poisonItem != nil && (poisonItem.Type == core.ItemTypeEmeticPoison || poisonItem.Type == core.ItemTypeLethalPoison)
	return hasPoison
}

func (a PoisonAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphPoison, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

type ChangeClothesAction struct{}

func (c ChangeClothesAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphClothing, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

func (c ChangeClothesAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	m.GetGame().SwitchClothesWith(person, m.GetGame().GetMap().DownedActorAt(position))
}

func (c ChangeClothesAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	noClothes := m.GetData().NoClothing()
	currentMap := m.GetGame().GetMap()
	return currentMap.IsDownedActorAt(actionAt) && currentMap.DownedActorAt(actionAt).Clothes != noClothes && person.Pos() == actionAt
}

type SnapNeckAction struct{}

func (s SnapNeckAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphEmptyHand, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

func (s SnapNeckAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	m.GetGame().SnapNeck(person, m.GetGame().GetMap().DownedActorAt(position))
}

func (s SnapNeckAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	actorAt := m.GetGame().GetMap().DownedActorAt(actionAt)
	if actorAt == nil {
		return false
	}
	return actorAt.IsSleeping() && person.Pos() != actionAt
}

type PushOverEdge struct {
}

func (p PushOverEdge) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphEmptyHand, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

func (p PushOverEdge) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	game := m.GetGame()

	var actorAt *core.Actor
	var isActorAt bool
	actorAt, isActorAt = game.GetMap().TryGetActorAt(position)
	if !isActorAt {
		actorAt, isActorAt = game.GetMap().TryGetDownedActorAt(position)
		if !isActorAt {
			return
		}
	}
	currentMap := game.GetMap()
	lethalNeighbors := currentMap.NeighborsCardinal(position, func(neighbor geometry.Point) bool {
		if !currentMap.Contains(neighbor) {
			return false
		}
		return currentMap.CellAt(neighbor).TileType.IsLethal()
	})
	if len(lethalNeighbors) == 0 {
		return
	}
	directionToTarget := lethalNeighbors[0].Sub(position)
	game.TryPushActorInDirection(actorAt, directionToTarget)
}

func (p PushOverEdge) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	var actorAt *core.Actor
	actorAt = m.GetGame().GetMap().ActorAt(actionAt)
	if actorAt == nil {
		actorAt = m.GetGame().GetMap().DownedActorAt(actionAt)
		if actorAt == nil {
			return false
		}
	}
	if actorAt == person {
		return false
	}
	currentMap := m.GetGame().GetMap()
	lethalNeighbors := currentMap.NeighborsCardinal(actionAt, func(neighbor geometry.Point) bool {
		if !currentMap.Contains(neighbor) {
			return false
		}
		return currentMap.CellAt(neighbor).TileType.IsLethal()
	})
	return !actorAt.IsInCombat() && len(lethalNeighbors) > 0
}

type DrownAction struct{}

func (d DrownAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphToilet, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

func (d DrownAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	victim := m.GetGame().GetMap().ActorAt(position)
	if victim == nil {
		return
	}
	animationCompleted := false
	until := func() bool {
		return animationCompleted
	}
	aic := m.GetAI()

	aic.SetEngaged(victim, core.ActorStatusVictimOfEngagement, until)
	aic.SetEngaged(person, core.ActorStatusEngagedIllegal, until)

	completed := func() {
		animationCompleted = true
		m.GetGame().Kill(victim, core.NewCauseOfDeath(core.CoDDrownedInToilet, person))
	}
	cancelled := func() {
		animationCompleted = true
	}
	m.GetAnimator().ActorEngagedIllegalAnimationWithSound(person, 'd', position, "drowning", completed, cancelled)
}

func (d DrownAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	actorAt := m.GetGame().GetMap().ActorAt(actionAt)
	if actorAt == nil || actorAt == person {
		return false
	}
	return actorAt.IsActive() && !actorAt.IsInvestigating() && !actorAt.IsInCombat() && m.GetGame().GetMap().ActorAt(actionAt) != person && m.GetGame().GetMap().IsNextToTileWithSpecial(actionAt, gridmap.SpecialTileToilet)
}

type PianoWire struct{}

func (s PianoWire) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphPianoWire, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

func (s PianoWire) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	m.GetGame().IllegalPlayerEngagementWithActorAtPos(position, core.GlyphPianoWire, func() {
		m.GetGame().Kill(m.GetGame().GetMap().ActorAt(position), core.NewCauseOfDeath(core.CoDStrangledWithWire, person))
	}, nil)
}

func (s PianoWire) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	currentMap := m.GetGame().GetMap()
	actorAt := currentMap.ActorAt(actionAt)
	if actorAt == nil {
		return false
	}
	isActive := actorAt.IsActive()
	isNotInvestigating := !actorAt.IsInvestigating()
	isNotInCombat := !actorAt.IsInCombat()
	isActiveAndUnsuspicious := isActive && isNotInvestigating && isNotInCombat
	return isActiveAndUnsuspicious && currentMap.ActorAt(actionAt) != person && person.EquippedItem != nil && person.EquippedItem.Type == core.ItemTypePianoWire
}

type MeleeTakedown struct{}

func (k MeleeTakedown) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return core.GlyphEmptyHand, common.DefaultStyle.WithBg(common.IllegalActionRed)
}

func (k MeleeTakedown) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	game := m.GetGame()
	currentMap := game.GetMap()
	victim := currentMap.ActorAt(position)
	if victim == nil {
		return
	}
	m.GetGame().IllegalActionAt(position, core.ObservationCombatSeen)
	if victim.IsInCombat() || victim.CanSeeInVisionCone(person.Pos()) {
		m.GetGame().SoundEventAt(position, core.ObservationMeleeNoises, 5)
	}
	println(fmt.Sprintf("Melee TAKEDOWN on %s", victim.DebugDisplayName()))
	// cancel gets triggered, meaning the victim was downed or moved..
	game.IllegalPlayerEngagementWithActorAtPos(position, core.GlyphEmptyHand, func() {
		game.SendToSleep(victim)
	}, nil)
}

func (k MeleeTakedown) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	currentMap := m.GetGame().GetMap()
	actorAt := currentMap.ActorAt(actionAt)
	if actorAt == nil || person == actorAt {
		return false
	}
	isActive := actorAt.IsActive()
	return isActive
}

type DialogueAction struct {
	DialogueName   string
	InitialSpeaker *core.Actor
}

func (d DialogueAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
	return 'T', common.DefaultStyle.WithBg(common.LegalActionGreen)
}

func (d DialogueAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	person.Dialogue.CurrentDialogue = d.DialogueName
	person.Dialogue.Situation = &core.OrientedLocation{Location: person.Pos()}
	person.Dialogue.LastHeardAtTick = m.CurrentTick()
	d.InitialSpeaker.Dialogue.Situation = &core.OrientedLocation{Location: d.InitialSpeaker.Pos()}
	d.InitialSpeaker.StartDialogue(d.DialogueName)
	println(fmt.Sprintf("Started dialogue %s from context action", d.DialogueName))
}

func (d DialogueAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	wasRecentlyActive := person.Dialogue.Active(m.CurrentTick())
	hasCurrentDialogueSet := person.Dialogue.CurrentDialogue != ""
	return !wasRecentlyActive && !hasCurrentDialogueSet
}
