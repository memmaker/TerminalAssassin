package states

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/mapset"
	"math/rand"
	"strconv"
	"strings"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/director"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/utils"
)

type ParserLogicRegisterer interface {
	RegisterAny(name string, anyFunc core.AnyFunc)
	RegisterPredicate(name string, predicate core.Predicate)
	RegisterAction(name string, actionFunc core.ActionFunc)
}

func (g *GameStateGameplay) registerPredicateAndAssignmentFunctions(parser ParserLogicRegisterer) {
	currentMap := g.engine.GetGame().GetMap()

	// VALUE FUNCTIONS
	parser.RegisterAny("NamedLocation", func(args ...any) any {
		nameArgument := args[0].(string)
		if pos, ok := currentMap.NamedLocations[nameArgument]; ok {
			return pos
		}
		println(fmt.Sprintf("(WARNING) NamedLocation: No location named '%s'", nameArgument))
		return geometry.Point{X: -1, Y: -1}
	})
	parser.RegisterAny("ActorWithName", func(args ...any) any {
		nameArgument := args[0].(string)
		for _, actor := range currentMap.AllActors {
			if actor.Name == nameArgument {
				return actor
			}
		}
		for _, actor := range currentMap.AllDownedActors {
			if actor.Name == nameArgument {
				return actor
			}
		}
		println(fmt.Sprintf("(WARNING) ActorWithName: No actor named '%s'", nameArgument))
		return nil
	})
	parser.RegisterAny("ItemWithNameInInventory", func(args ...any) any {
		person := args[0].(*core.Actor)
		itemName := args[1].(string)
		for _, item := range person.Inventory.Items {
			if item.Name == itemName {
				return item
			}
		}
		println(fmt.Sprintf("(WARNING) ItemWithNameInInventory: No item named '%s' in inventory of '%s'", itemName, person.DebugDisplayName()))
		return nil
	})
	parser.RegisterAny("KeyItem", func(args ...any) any {
		person := args[0].(*core.Actor)
		key := args[1].(string)
		for _, item := range person.Inventory.Items {
			if item.KeyString == key {
				return item
			}
		}
		println(fmt.Sprintf("(WARNING) KeyItem: No item with key '%s' in inventory of '%s'", key, person.DebugDisplayName()))
		return nil
	})
	parser.RegisterAny("NearestItemWithName", func(args ...any) any {
		actorArgument, argOK := args[0].(*core.Actor)
		if !argOK {
			println(fmt.Sprintf("Expected actor as first argument, got '%v'", args[0]))
			return nil
		}
		namePart, argOK := args[1].(string)
		if !argOK {
			println(fmt.Sprintf("Expected string as second argument, got '%v'", args[1]))
			return nil
		}
		var nearestItem *core.Item
		var nearestDistance int
		for _, item := range currentMap.AllItems {
			if strings.Contains(strings.ToLower(item.Name), strings.ToLower(namePart)) {
				distance := geometry.DistanceManhattan(item.Pos(), actorArgument.Pos())
				if nearestItem == nil || distance < nearestDistance {
					nearestItem = item
					nearestDistance = distance
				}
			}
		}
		if nearestItem == nil {
			println(fmt.Sprintf("(WARNING) NearestItemWithName: No item named '%s' was found", namePart))
		}
		return nearestItem
	})

	// PREDICATES
	parser.RegisterPredicate("AreActorsMeeting", func(args ...any) bool {
		actor1 := args[0].(*core.Actor)
		actor2 := args[1].(*core.Actor)
		distanceManhattan := geometry.DistanceManhattan(actor1.Pos(), actor2.Pos())
		canSeeEachOther := actor1.CanSee(actor2.Pos()) || actor2.CanSee(actor1.Pos())
		areNearToEachOther := distanceManhattan < 6
		return areNearToEachOther && canSeeEachOther
	})
	parser.RegisterPredicate("IsScriptFinished", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		return actor.Script.IsFinished()
	})
	parser.RegisterPredicate("IsItemAtPlayerPosition", func(args ...any) bool {
		player := g.engine.GetGame().GetMap().Player
		playerPos := player.Pos()
		return currentMap.IsItemAt(playerPos)
	})
	parser.RegisterPredicate("HasRangedWeapon", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		return actor.HasWeapon() // TODO: check if it's a ranged weapon
	})
	parser.RegisterPredicate("IsDead", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		return actor.IsDead()
	})
	parser.RegisterPredicate("IsScriptFinished", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		return actor.Script.IsFinished()
	})
	parser.RegisterPredicate("IsAtLocation", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		location := args[1].(geometry.Point)
		return actor.Pos() == location
	})
	parser.RegisterPredicate("IsMissionTimeInSeconds", func(args ...any) bool {
		missionRunningSinceSeconds, _ := strconv.Atoi(args[0].(string))
		if g.engine.CurrentTick() > uint64(utils.SecondsToTicks(float64(missionRunningSinceSeconds))) {
			return true
		}
		return false
	})
	parser.RegisterPredicate("IsWaiting", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		return actor.IsIdle()
	})
	parser.RegisterPredicate("IsDowned", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		return actor.IsDowned()
	})
	parser.RegisterPredicate("HasDialogueEnded", func(args ...any) bool {
		dialogueName := args[0].(string)
		if dialogue, ok := g.mapDialogues[dialogueName]; ok {
			return dialogue.LastSpeaker.Dialogue.HasSpoken(dialogue.LastSpeechCode)
		}
		return false
	})
	parser.RegisterPredicate("CanSeeActor", func(args ...any) bool {
		viewer := args[0].(*core.Actor)
		target := args[1].(*core.Actor)
		return viewer.CanSeeActor(target)
	})
	parser.RegisterPredicate("HasItemInInventory", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		item := args[1].(*core.Item)
		for _, heldItem := range actor.Inventory.Items {
			if item == heldItem {
				return true
			}
		}
		return false
	})
	parser.RegisterPredicate("HasEnteredZone", func(args ...any) bool {
		actor := args[0].(*core.Actor)
		zoneName := args[1].(string)
		currentZone := currentMap.ZoneAt(actor.Pos())
		lastZone := currentMap.ZoneAt(actor.LastPos)
		return currentZone.Name == zoneName && lastZone.Name != zoneName
	})
	parser.RegisterPredicate("HasPlayerEnteredZone", func(args ...any) bool {
		actor := currentMap.Player
		zoneName := args[0].(string)
		currentZone := currentMap.ZoneAt(actor.Pos())
		lastZone := currentMap.ZoneAt(actor.LastPos)
		return currentZone.Name == zoneName && lastZone.Name != zoneName
	})
	parser.RegisterPredicate("IsPlayerTrespassing", func(args ...any) bool {
		return currentMap.IsTrespassing(currentMap.Player)
	})
	parser.RegisterPredicate("HasNotPlayerChangedClothes", func(args ...any) bool {
		stats := g.engine.GetGame().GetStats()
		return stats.DisguisesWorn.Cardinality() == 0
	})
}

func (g *GameStateGameplay) registerActionFunctions(parser ParserLogicRegisterer) {
	// ACTIONS
	parser.RegisterAction("Wait", func(args ...any) {
		actor := args[0].(*core.Actor)
		delay, _ := strconv.ParseFloat(args[1].(string), 64)
		actor.Script.AddAction(director.NewWaitAction(delay))
	})
	parser.RegisterAction("Approach", func(args ...any) {
		actor := args[0].(*core.Actor)
		target := args[1].(*core.Actor)
		actor.Script.AddAction(director.NewApproachAction(target))
	})
	parser.RegisterAction("UseItemAtRange", func(args ...any) {
		actor := args[0].(*core.Actor)
		target := args[1].(*core.Actor)
		actor.Script.AddAction(director.NewUseItemAtRangeAction(g.engine, target))
	})
	parser.RegisterAction("MoveToItem", func(args ...any) {
		actor := args[0].(*core.Actor)
		item := args[1].(*core.Item)
		actor.Script.AddAction(director.NewMoveToItem(g.engine, func(currentItem *core.Item) bool {
			return currentItem == item
		}))
	})
	parser.RegisterAction("MoveToLocation", func(args ...any) {
		actor := args[0].(*core.Actor)
		destination := args[1].(geometry.Point)
		actor.Script.AddAction(director.NewMoveAction(destination))
	})
	parser.RegisterAction("SetPreferredLocation", func(args ...any) {
		actor := args[0].(*core.Actor)
		destination := args[1].(geometry.Point)
		actor.Script.AddAction(director.NewSetPreferredLocation(destination))
	})
	parser.RegisterAction("SetPreferredLocationHere", func(args ...any) {
		actor := args[0].(*core.Actor)
		actor.Script.AddAction(director.NewSetPreferredLocationHere())
	})
	parser.RegisterAction("StopStaying", func(args ...any) {
		actor := args[0].(*core.Actor)
		actor.Script.AddAction(director.NewStopStayingAtPreferredLocationAction())
	})
	parser.RegisterAction("PickUpItem", func(args ...any) {
		actor := args[0].(*core.Actor)
		actor.Script.AddAction(director.NewPickUpAction(g.engine))
	})
	parser.RegisterAction("Wait", func(args ...any) {
		actor := args[0].(*core.Actor)
		aic := g.engine.GetAI()
		actor.Script.AddAction(director.NewSwitchToWaitAction(aic))
	})
	parser.RegisterAction("DropFromInventory", func(args ...any) {
		actor := args[0].(*core.Actor)
		item := args[1].(*core.Item)
		actor.Script.AddAction(director.NewDropFromInventoryAction(g.engine, item))
	})
	parser.RegisterAction("Dance", func(args ...any) {
		actor := args[0].(*core.Actor)
		delayInSeconds, _ := strconv.ParseFloat(args[1].(string), 64)
		actor.Script.AddAction(director.NewDanceAction(delayInSeconds))
	})
	parser.RegisterAction("TurnTable", func(args ...any) {
		actor := args[0].(*core.Actor)
		delayInSeconds, _ := strconv.ParseFloat(args[1].(string), 64)
		actor.Script.AddAction(director.NewTurnTableAction(delayInSeconds))
	})

	//// INSTANT ACTIONS
	parser.RegisterAction("InstantDropFromInventory", func(args ...any) {
		actor := args[0].(*core.Actor)
		item := args[1].(*core.Item)
		director.NewDropFromInventoryAction(g.engine, item).Execute(actor)
	})
	parser.RegisterAction("StartDialogue", func(args ...any) {
		dialogueName := args[0].(string)
		if dialogue, ok := g.mapDialogues[dialogueName]; ok {
			dialogue.InitialSpeaker.StartDialogue(dialogueName)
		}
	})
	parser.RegisterAction("Print", func(args ...any) {
		text := args[1].(string)
		g.Print(text)
	})
	parser.RegisterAction("Say", func(args ...any) {
		actor := args[0].(*core.Actor)
		text := args[1].(string)
		actor.SetNextUtterance(core.Utterance{
			Line:      core.NewStyledText(text, actor.ChatStyle()),
			EventCode: "DLG_SAY",
		})
	})
	parser.RegisterAction("DeleteDialogue", func(args ...any) {
		dialogueName := args[0].(string)
		if dialogue, ok := g.mapDialogues[dialogueName]; ok {
			delete(g.mapDialogues, dialogueName)
			dialogue.Participants.Iter(func(participant *core.Actor) {
				if _, hasDialogue := participant.Dialogue.Conversations[dialogueName]; hasDialogue {
					delete(participant.Dialogue.Conversations, dialogueName)
				}
			})
		}
	})
	parser.RegisterAction("PinDialogueLocation", func(args ...any) {
		actor := args[0].(*core.Actor)
		actor.Dialogue.Situation = &core.OrientedLocation{
			Location:  actor.Pos(),
			Direction: actor.LookDirection,
		}
	})
	parser.RegisterAction("SwitchToScript", func(args ...any) {
		actor := args[0].(*core.Actor)
		actor.Script.SetDefaultMoveActionGenerator(director.NewMoveAction)
		aic := g.engine.GetAI()
		aic.SwitchToScript(actor)
	})
	parser.RegisterAction("LookAtActor", func(args ...any) {
		actor := args[0].(*core.Actor)
		target := args[1].(*core.Actor)
		actor.LookAt(target.Pos())
		//actor.Script.AddAction(director.NewLookAtActorAction(target))
	})
	parser.RegisterAction("StopScripted", func(args ...any) {
		actor := args[0].(*core.Actor)
		aic := g.engine.GetAI()
		aic.TryPopScripted(actor)
	})
	parser.RegisterAction("CreateTravelGroup", func(args ...any) {
		aic := g.engine.GetAI()
		group := mapset.NewSet[*core.Actor]()
		for _, actor := range args {
			group.Add(actor.(*core.Actor))
		}
		aic.CreateTravelGroup(group)
	})
	parser.RegisterAction("DeleteTravelGroup", func(args ...any) {
		group := mapset.NewSet[*core.Actor]()
		for _, actor := range args {
			group.Add(actor.(*core.Actor))
		}
		aic := g.engine.GetAI()
		aic.DeleteTravelGroup(group)
	})
	// Special Action for map
	parser.RegisterAction("FillZoneRandomlyWithStimuli", func(args ...any) {
		nameOfZone := args[0].(string)
		nameOfStim := stimuli.StimulusType(args[1].(string))
		amount, _ := strconv.Atoi(args[2].(string))
		currentMap := g.engine.GetGame().GetMap()
		var zoneToFill *gridmap.ZoneInfo
		for _, zone := range currentMap.ListOfZones {
			if zone.Name == nameOfZone {
				zoneToFill = zone
			}
		}
		zonePositions := make([]geometry.Point, 0)
		currentMap.IterAll(func(pos geometry.Point, cell gridmap.MapCell[*core.Actor, *core.Item, services.Object]) {
			if currentMap.ZoneAt(pos) == zoneToFill {
				zonePositions = append(zonePositions, pos)
			}
		})
		// 30 seconds = 30.000 ms
		locationCount := len(zonePositions)
		secondsForFill := 30.0
		secondsPerLocation := secondsForFill / float64(locationCount)
		cumulativeDelay := 0.0
		for i := 0; i < locationCount; i++ {
			// pop a random position from the list
			randomIndex := rand.Intn(len(zonePositions))
			randomPos := zonePositions[randomIndex]
			zonePositions = append(zonePositions[:randomIndex], zonePositions[randomIndex+1:]...)

			g.engine.Schedule(cumulativeDelay, func() {
				currentMap.AddStimulusToTile(randomPos, stimuli.Stim{StimType: nameOfStim, StimForce: amount / 2})
			})
			g.engine.Schedule(cumulativeDelay+(rand.Float64()*1), func() {
				currentMap.AddStimulusToTile(randomPos, stimuli.Stim{StimType: nameOfStim, StimForce: amount / 2})
			})
			cumulativeDelay += secondsPerLocation
		}
		//zone.FillRandomlyWithStimuli(stimuli, amount)
	})

	// Training Related
	parser.RegisterAction("PrintMovementHint", func(args ...any) {
		input := g.engine.GetInput()
		movementKey := input.GetKeyDefinitions().MovementKeys
		infoText := fmt.Sprintf("HINT: Use @l[%s,%s,%s,%s]@N to move around", movementKey[0].String(), movementKey[1].String(), movementKey[2].String(), movementKey[3].String())
		g.PrintStyled(core.NewStyledText(infoText, common.TerminalStyle).WithMarkup('l', common.DefaultStyle.WithFg(common.Green)))
	})
	parser.RegisterAction("PrintPickupHint", func(args ...any) {
		input := g.engine.GetInput()
		movementKey := input.GetKeyDefinitions().SameTileActionKey
		infoText := fmt.Sprintf("HINT: Use @l[%s]@N to pick up items", movementKey.String())
		g.PrintStyled(core.NewStyledText(infoText, common.TerminalStyle).WithMarkup('l', common.DefaultStyle.WithFg(common.Green)))
	})
	parser.RegisterAction("PrintExplorationHint", func(args ...any) {
		g.PrintStyled(core.NewStyledText("HINT: Move @lNorth-East@N towards the landing bridge.", common.TerminalStyle).WithMarkup('l', common.DefaultStyle.WithFg(common.Green)))
	})
	parser.RegisterAction("ShowInfiltrationAlert", func(args ...any) {
		training := NewTrainingHelper(g.engine)
		training.alertInfiltration()
	})
	parser.RegisterAction("ShowPeekingKeyHolesAlert", func(args ...any) {
		training := NewTrainingHelper(g.engine)
		training.alertPeekingThroughKeyholes()
	})
	parser.RegisterAction("ShowPeekingPickupAlert", func(args ...any) {
		training := NewTrainingHelper(g.engine)
		training.alertPeekingPickup()
	})
}
