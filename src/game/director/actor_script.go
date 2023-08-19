package director

import (
	"fmt"
	"math/rand"

	"github.com/memmaker/terminal-assassin/game/ai"
	"github.com/memmaker/terminal-assassin/game/core"

	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type AIAction struct {
	action         func(person *core.Actor) core.AIUpdate
	isDone         func(person *core.Actor) bool
	actionFinished bool
	description    string
}

func (a *AIAction) ToString() string {
	return a.description
}

func (a *AIAction) OnDestinationReached() core.AIUpdate {
	a.actionFinished = true
	println("Scripted action reached destination..")
	return ai.NextUpdateIn(1)
}

func (a *AIAction) OnCannotReachDestination() core.AIUpdate {
	//a.actionFinished = true
	println("Scripted action cannot reach destination..")
	return ai.NextUpdateIn(1)
}

func (a *AIAction) Execute(person *core.Actor) core.AIUpdate {
	return a.action(person)
}

func (a *AIAction) IsDone(person *core.Actor) bool {
	return (a.isDone != nil && a.isDone(person)) || a.actionFinished
}

func NewWaitAction(delayInSeconds float64) core.ScriptedAction {
	scriptedAction := &AIAction{description: fmt.Sprintf("wait %v seconds", delayInSeconds)}
	waitAction := func(person *core.Actor) core.AIUpdate {
		scriptedAction.actionFinished = true
		return ai.NextUpdateIn(delayInSeconds)
	}
	scriptedAction.action = waitAction
	return scriptedAction
}

func NewPickUpAction(engine services.Engine) core.ScriptedAction {
	scriptedAction := &AIAction{description: "pick up item"}
	pickUpAction := func(person *core.Actor) core.AIUpdate {
		engine.GetGame().PickUpItem(person)
		scriptedAction.actionFinished = true
		return ai.NextUpdateIn(1)
	}
	scriptedAction.action = pickUpAction
	return scriptedAction
}
func NewDropFromInventoryAction(engine services.Engine, item *core.Item) core.ScriptedAction {
	scriptedAction := &AIAction{description: "drop item from inventory"}
	dropAction := func(person *core.Actor) core.AIUpdate {
		engine.GetGame().DropFromInventory(person, []*core.Item{item})
		scriptedAction.actionFinished = true
		return ai.NextUpdateIn(1)
	}
	scriptedAction.action = dropAction
	return scriptedAction
}
func NewDanceAction(delayInSeconds float64) core.ScriptedAction {
	scriptedAction := &AIAction{description: "dance"}
	danceAction := func(person *core.Actor) core.AIUpdate {
		person.LookDirection = rand.Float64() * 360
		return ai.NextUpdateIn(delayInSeconds)
	}
	scriptedAction.action = danceAction
	return scriptedAction
}
func NewTurnTableAction(delayInSeconds float64) core.ScriptedAction {
	scriptedAction := &AIAction{description: "turn table"}
	firstCall := true
	originalDirection := 0.0
	turnTableAction := func(person *core.Actor) core.AIUpdate {
		if firstCall {
			firstCall = false
			originalDirection = person.LookDirection
		}
		person.LookDirection = originalDirection + (rand.Float64() * 90) - 45
		return ai.NextUpdateIn(delayInSeconds)
	}
	scriptedAction.action = turnTableAction
	return scriptedAction
}
func NewSwitchToWaitAction(aic services.AIInterface) core.ScriptedAction {
	scriptedAction := &AIAction{description: "switch to wait"}
	switchToWaitAction := func(person *core.Actor) core.AIUpdate {
		aic.SwitchToWait(person)
		scriptedAction.actionFinished = true
		return ai.NextUpdateIn(1)
	}
	scriptedAction.action = switchToWaitAction
	return scriptedAction
}

func NewMoveToItem(engine services.Engine, predicate func(item *core.Item) bool) core.ScriptedAction {
	scriptedAction := &AIAction{description: "move to item"}
	getItemAction := func(person *core.Actor) core.AIUpdate {
		nearestItem := engine.GetGame().GetMap().FindNearestItem(person.Pos(), predicate)
		if nearestItem != nil {
			return person.AI.Movement.Action(nearestItem.Pos(), scriptedAction)
		}
		scriptedAction.actionFinished = true
		return ai.NextUpdateIn(1)
	}
	scriptedAction.action = getItemAction
	return scriptedAction
}

func NewUseItemAtRangeAction(engine services.Engine, target *core.Actor) core.ScriptedAction {
	scriptedAction := &AIAction{description: "use item", actionFinished: true}
	combatAction := func(person *core.Actor) core.AIUpdate {
		person.EquipWeapon()
		engine.GetGame().GetActions().UseEquippedItemAtRange(person, target.Pos())
		return ai.NextUpdateIn(1)
	}
	scriptedAction.action = combatAction
	return scriptedAction
}
func NewMoveAction(destination geometry.Point) core.ScriptedAction {
	scriptedAction := &AIAction{description: "move"}
	moveAction := func(person *core.Actor) core.AIUpdate {
		return person.AI.Movement.Action(destination, scriptedAction)
	}
	scriptedAction.action = moveAction
	scriptedAction.isDone = func(person *core.Actor) bool {
		return person.Pos() == destination
	}
	return scriptedAction
}
func NewSetPreferredLocation(destination geometry.Point) core.ScriptedAction {
	registerPreferredLocationAction := func(person *core.Actor) core.AIUpdate {
		person.Script.SetPreferredLocation(destination)
		return ai.NextUpdateIn(1)
	}
	scriptedAction := &AIAction{
		description:    "move and wait",
		actionFinished: true,
		action:         registerPreferredLocationAction,
	}

	return scriptedAction
}

func NewSetPreferredLocationHere() core.ScriptedAction {
	registerPreferredLocationAction := func(person *core.Actor) core.AIUpdate {
		person.Script.SetPreferredLocation(person.Pos())
		return ai.NextUpdateIn(1)
	}
	scriptedAction := &AIAction{
		description:    "stay here",
		actionFinished: true,
		action:         registerPreferredLocationAction,
	}

	return scriptedAction
}
func NewStopStayingAtPreferredLocationAction() core.ScriptedAction {
	registerPreferredLocationAction := func(person *core.Actor) core.AIUpdate {
		person.Script.StopStayingInPreferredLocation()
		return ai.NextUpdateIn(1)
	}
	scriptedAction := &AIAction{
		description:    "stopping staying at preferred location",
		actionFinished: true,
		action:         registerPreferredLocationAction,
	}

	return scriptedAction
}
func NewLookAtActorAction(other *core.Actor) core.ScriptedAction {
	scriptedAction := &AIAction{description: "look at actor"}
	lookAction := func(person *core.Actor) core.AIUpdate {
		person.LookAt(other.Pos())
		scriptedAction.actionFinished = true
		return ai.NextUpdateIn(0.2)
	}
	scriptedAction.action = lookAction
	return scriptedAction
}
func NewLookAtAction(destination geometry.Point) core.ScriptedAction {
	scriptedAction := &AIAction{description: "look at position"}
	lookAction := func(person *core.Actor) core.AIUpdate {
		person.LookAt(destination)
		scriptedAction.actionFinished = true
		return ai.NextUpdateIn(0.2)
	}
	scriptedAction.action = lookAction
	return scriptedAction
}
func NewApproachAction(other *core.Actor) core.ScriptedAction {
	scriptedAction := &AIAction{description: "approach"}
	moveAction := func(person *core.Actor) core.AIUpdate {
		return person.AI.Movement.Action(other.Pos(), scriptedAction)
	}
	scriptedAction.action = moveAction
	return scriptedAction
}
