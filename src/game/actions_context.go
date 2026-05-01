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
    currentMap := m.GetGame().GetMap()

    return currentMap.IsItemAt(actionAt) && !currentMap.ItemAt(actionAt).Buried &&
        geometry.DistanceManhattan(person.Pos(), actionAt) <= 1
}

func (p PickupAction) Description(m services.Engine, person *core.Actor, pos geometry.Point) (rune, common.Style) {
    style := common.DefaultStyle.WithBg(core.CurrentTheme.LegalActionBackground)
    if !p.item.IsLegalForActor(person) {
        style = style.WithBg(core.CurrentTheme.IllegalActionBackground)
    }
    return p.item.Icon(), style
}

func (p PickupAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
    m.GetGame().PickUpItemAt(person, position)
}

type OverflowAction struct{}

func (a OverflowAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
    return core.GlyphWater, common.DefaultStyle.WithBg(core.CurrentTheme.IllegalActionBackground)
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
    return core.GlyphExit, common.DefaultStyle.WithBg(core.CurrentTheme.LegalActionBackground)
}

func (a ExitAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
    m.GetGame().EndMissionWithSuccess()
}

func (a ExitAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
    for _, inLevel := range m.GetGame().GetMap().Actors() {
        if inLevel.IsTarget && inLevel.Status != core.ActorStatusDead {
            return false
        }
    }
    for _, inLevel := range m.GetGame().GetMap().DownedActors() {
        if inLevel.IsTarget && inLevel.Status != core.ActorStatusDead {
            return false
        }
    }
    return true
}

type ExposeElectricityAction struct {
    Icon rune
}

func (e ExposeElectricityAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
    return e.Icon, common.DefaultStyle.WithBg(core.CurrentTheme.IllegalActionBackground)
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
    if !a.hasPoison(person) {
        return
    }
    currentMap := m.GetGame().GetMap()
    poisonItem := person.EquippedItem
    person.EquippedItem = nil
    person.Inventory.RemoveItem(poisonItem)
    switch poisonItem.Type {
    case core.ItemTypeEmeticPoison:
        currentMap.AddStimulusToTile(position, stimuli.Stim{StimType: stimuli.StimulusEmeticPoison, StimForce: 10})
        m.GetGame().PrintMessage("You lace the food with emetic poison.")
    case core.ItemTypeLethalPoison:
        currentMap.AddStimulusToTile(position, stimuli.Stim{StimType: stimuli.StimulusLethalPoison, StimForce: 10})
        m.GetGame().PrintMessage("You lace the food with lethal poison.")
    case core.ItemTypeSleepPoison:
        currentMap.AddStimulusToTile(position, stimuli.Stim{StimType: stimuli.StimulusInducedSleep, StimForce: 10})
        m.GetGame().PrintMessage("You lace the food with sleeping poison.")
    }
    m.GetGame().IllegalActionAt(position, core.ObservationIllegalAction)
}

func (a PoisonAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
    currentMap := m.GetGame().GetMap()
    isFoodOrDrinkTile := currentMap.IsTileWithSpecialAt(actionAt, gridmap.SpecialTileTypeFood)
    return a.hasPoison(person) &&
        isFoodOrDrinkTile &&
        !currentMap.IsStimulusOnTile(actionAt, stimuli.StimulusLethalPoison) &&
        !currentMap.IsStimulusOnTile(actionAt, stimuli.StimulusEmeticPoison) &&
        !currentMap.IsStimulusOnTile(actionAt, stimuli.StimulusInducedSleep)
}

func (a PoisonAction) hasPoison(actor *core.Actor) bool {
    poisonItem := actor.EquippedItem
    hasPoison := poisonItem != nil && (poisonItem.Type == core.ItemTypeEmeticPoison || poisonItem.Type == core.ItemTypeLethalPoison || poisonItem.Type == core.ItemTypeSleepPoison)
    return hasPoison
}

func (a PoisonAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
    return core.GlyphPoison, common.DefaultStyle.WithBg(core.CurrentTheme.IllegalActionBackground)
}

type SnapNeckAction struct{}

func (s SnapNeckAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
    return core.GlyphEmptyHand, common.DefaultStyle.WithBg(core.CurrentTheme.IllegalActionBackground)
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
    return core.GlyphEmptyHand, common.DefaultStyle.WithBg(core.CurrentTheme.IllegalActionBackground)
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
    return core.GlyphToilet, common.DefaultStyle.WithBg(core.CurrentTheme.IllegalActionBackground)
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
        triggerEscapeReaction(m, victim, person)
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
    return core.GlyphPianoWire, common.DefaultStyle.WithBg(core.CurrentTheme.IllegalActionBackground)
}

func (s PianoWire) Action(m services.Engine, person *core.Actor, position geometry.Point) {
	victim := m.GetGame().GetMap().ActorAt(position)
	if victim == nil {
		return
	}
	const pianoWireTime = meleeEngagementTime / 3.0
	m.GetGame().IllegalPlayerEngagementWithActorAtPos(position, core.GlyphPianoWire, pianoWireTime, func() {
		m.GetGame().Kill(victim, core.NewCauseOfDeath(core.CoDStrangledWithWire, person))
	}, func() {
		triggerEscapeReaction(m, victim, person)
	})
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

// meleeEngagementTime is the base duration in seconds for a melee takedown.
const meleeEngagementTime = 4.0

// triggerEscapeReaction puts the victim into the appropriate hostile state when
// a player-initiated engagement is cancelled (victim escaped). Guards enter combat
// against the attacker; civilians and targets panic.
func triggerEscapeReaction(m services.Engine, victim *core.Actor, attacker *core.Actor) {
	if victim == nil || victim.IsDowned() {
		return
	}
	aic := m.GetAI()
	if victim.Type == core.ActorTypeGuard {
		aic.SwitchToCombat(victim, attacker)
	} else {
		aic.SwitchToPanic(victim, []geometry.Point{attacker.Pos()})
	}
}

type MeleeTakedown struct{}

func (k MeleeTakedown) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
    return core.GlyphEmptyHand, common.DefaultStyle.WithBg(core.CurrentTheme.IllegalActionBackground)
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
    game.IllegalPlayerEngagementWithActorAtPos(position, core.GlyphEmptyHand, meleeEngagementTime, func() {
        game.SendToSleep(victim)
    }, func() {
        triggerEscapeReaction(m, victim, person)
    })
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

type KnockOnWallAction struct{}

func (k KnockOnWallAction) Description(_ services.Engine, _ *core.Actor, _ geometry.Point) (rune, common.Style) {
    return core.GlyphEmptyHand, common.DefaultStyle.WithBg(core.CurrentTheme.LegalActionBackground)
}

func (k KnockOnWallAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
    m.GetGame().SoundEventAt(person.Pos(), core.ObservationStrangeNoiseHeard, 8)
}

func (k KnockOnWallAction) IsActionPossible(m services.Engine, _ *core.Actor, actionAt geometry.Point) bool {
    currentMap := m.GetGame().GetMap()
    cell := currentMap.CellAt(actionAt)
    return !cell.TileType.IsWalkable && cell.TileType.Special == gridmap.SpecialTileNone
}

type DialogueAction struct {
    DialogueName   string
    InitialSpeaker *core.Actor
}

// FenceShopAction triggers the fence's buy/sell menu when the player is adjacent.
type FenceShopAction struct{}

func (f FenceShopAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
    actorAt := m.GetGame().GetMap().ActorAt(actionAt)
    return actorAt != nil && actorAt.IsFence() && actorAt.IsActive()
}

func (f FenceShopAction) Description(_ services.Engine, _ *core.Actor, _ geometry.Point) (rune, common.Style) {
    return 'F', common.DefaultStyle.WithBg(core.CurrentTheme.LegalActionBackground)
}

func (f FenceShopAction) Action(m services.Engine, person *core.Actor, position geometry.Point) {
    fence := m.GetGame().GetMap().ActorAt(position)
    if fence == nil {
        return
    }
    openFenceShopMenu(m, person, fence, 0)
}

func openFenceShopMenu(m services.Engine, player *core.Actor, fence *core.Actor, initialIndex int) {
    userInterface := m.GetUI()
    career := m.GetCareer()
    var menuItems []services.MenuItem

    hasLoot := false
    for _, item := range player.Inventory.Items {
        if item.LootValue > 0 && !item.IsLockpickType() {
            hasLoot = true
            break
        }
    }
    if hasLoot {
        menuItems = append(menuItems, services.MenuItem{
            Label: "Sell loot >",
            Handler: func() {
                openSellLootMenu(m, player)
            },
        })
    }

    for idx, itm := range fence.Inventory.Items {
        item := itm
        itemIdx := idx
        price := uint64(item.LootValue)
        if item.IsLockpickType() {
            // Lockpicks are sold one at a time (one use = one pick).
            menuItems = append(menuItems, services.MenuItem{
                Label: fmt.Sprintf("Buy %s ($%d each, %d left)", item.Name, price, item.Uses),
                Handler: func() {
                    if career.Money < price {
                        m.GetGame().PrintMessage(fmt.Sprintf("Not enough money. Balance: $%d", career.Money))
                        openFenceShopMenu(m, player, fence, itemIdx)
                        return
                    }
                    career.Money -= price
                    // Add one use to an existing stack in the player's inventory, or create a new item.
                    if existing := player.Inventory.FindItemByType(item.Type); existing != nil {
                        existing.Uses++
                    } else {
                        single := item.DeepCopy()
                        single.Uses = 1
                        player.Inventory.AddItem(single)
                        single.HeldBy = player
                    }
                    // Consume one use from the fence's stack.
                    item.Uses--
                    if item.Uses <= 0 {
                        fence.Inventory.RemoveItem(item)
                    }
                    m.GetGame().PrintMessage(fmt.Sprintf("Bought 1x %s for $%d. Balance: $%d", item.Name, price, career.Money))
                    m.GetGame().UpdateHUD()
                    openFenceShopMenu(m, player, fence, itemIdx)
                },
            })
        } else {
            menuItems = append(menuItems, services.MenuItem{
                Label: fmt.Sprintf("Buy %s ($%d)", item.Name, price),
                Handler: func() {
                    if career.Money < price {
                        m.GetGame().PrintMessage(fmt.Sprintf("Not enough money. Balance: $%d", career.Money))
                        openFenceShopMenu(m, player, fence, itemIdx)
                        return
                    }
                    career.Money -= price
                    fence.Inventory.RemoveItem(item)
                    player.Inventory.AddItem(item)
                    item.HeldBy = player
                    m.GetGame().PrintMessage(fmt.Sprintf("Bought %s for $%d. Balance: $%d", item.Name, price, career.Money))
                    m.GetGame().UpdateHUD()
                    openFenceShopMenu(m, player, fence, itemIdx)
                },
            })
        }
    }

    if len(menuItems) == 0 {
        m.GetGame().PrintMessage("The fence has nothing to offer.")
        return
    }
    m.GetGame().PrintMessage(fmt.Sprintf("Balance: $%d", career.Money))
    userInterface.OpenWideAutoCloseMenuWithCallback("The Fence", menuItems, initialIndex, nil)
}

func openSellLootMenu(m services.Engine, player *core.Actor) {
    userInterface := m.GetUI()
    career := m.GetCareer()
    var sellItems []services.MenuItem

    for _, itm := range player.Inventory.Items {
        item := itm
        if item.LootValue <= 0 || item.IsLockpickType() {
            continue
        }
        value := uint64(item.LootValue)
        sellItems = append(sellItems, services.MenuItem{
            Label: fmt.Sprintf("Sell %s (+$%d)", item.Name, value),
            Handler: func() {
                player.RemoveItem(item)
                career.Money += value
                m.GetGame().PrintMessage(fmt.Sprintf("Sold %s for $%d. Balance: $%d", item.Name, value, career.Money))
                m.GetGame().UpdateHUD()
                openSellLootMenu(m, player)
            },
        })
    }

    if len(sellItems) == 0 {
        return
    }
    userInterface.OpenWideAutoCloseMenuWithCallback("Sell Loot", sellItems, 0, nil)
}

func (d DialogueAction) Description(services.Engine, *core.Actor, geometry.Point) (rune, common.Style) {
    return 'T', common.DefaultStyle.WithBg(core.CurrentTheme.LegalActionBackground)
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
