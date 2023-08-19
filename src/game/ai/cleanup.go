package ai

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/stimuli"
)

type CleanupMovement struct {
	AIContext
	currentIncident core.IncidentReport
	cleaningIsDone  bool
	securedItems    []*core.Item
}

func (c *CleanupMovement) OnDestinationReached() core.AIUpdate {
	person := c.Person
	game := c.Engine.GetGame()
	currentMap := game.GetMap()
	currentZone := currentMap.ZoneAt(person.Pos())

	if currentZone.IsDropOff() {
		droppedStuff := false
		if len(c.securedItems) > 0 {
			c.dropOffItems()
			droppedStuff = true
		}
		if person.IsDraggingBody() {
			person.MovementMode = core.MovementModeWalking
			person.DraggedBody = nil
			droppedStuff = true
		}
		if droppedStuff {
			return NextUpdateIn(0.1)
		}
	}

	return c.performCleanup()
}

func (c *CleanupMovement) dropOffItems() {
	person := c.Person
	game := c.Engine.GetGame()
	game.DropFromInventory(person, c.securedItems)
	for _, item := range c.securedItems {
		item.StartPosition = item.Pos()
	}
	c.securedItems = []*core.Item{}
}

func (c *CleanupMovement) OnCannotReachDestination() core.AIUpdate {
	person := c.Person
	person.AI.PopState()
	return NextUpdateIn(0.1)
}

func (c *CleanupMovement) NextAction() core.AIUpdate {
	person := c.Person
	person.Status = core.ActorStatusCleanup

	if person.IsDraggingBody() {
		return c.gotoDropoff()
	}

	if c.noMoreCleaningTasks() {
		if len(c.securedItems) > 0 || person.IsDraggingBody() {
			return c.gotoDropoff()
		} else {
			person.AI.PopState()
			return NextUpdateIn(0.1)
		}
	}
	return person.AI.Movement.Action(c.currentIncident.Location, c)
}

func (c *CleanupMovement) noMoreCleaningTasks() bool {
	return (c.currentIncident == core.EmptyReport || c.cleaningIsDone) && !c.tryAcquireNewCleanupTask()
}

func (c *CleanupMovement) gotoDropoff() core.AIUpdate {
	person := c.Person
	game := c.Engine.GetGame()
	currentMap := game.GetMap()
	dropOffLocation := currentMap.GetNearestDropOffPosition(person.Pos())
	return person.AI.Movement.Action(dropOffLocation, c)
}

func (c *CleanupMovement) tryAcquireNewCleanupTask() bool {
	person := c.Person
	aic := c.Engine.GetAI()
	incident := aic.GetIncidentForCleanup(person)
	if incident == core.EmptyReport {
		return false
	}
	c.currentIncident = incident
	c.cleaningIsDone = false
	println(fmt.Sprintf("%s aims to clean up %s at %s", person.Name, incident.Type, incident.Location))
	return true
}

func (c *CleanupMovement) performCleanup() core.AIUpdate {
	person := c.Person
	aic := c.Engine.GetAI()
	game := c.Engine.GetGame()
	if c.currentIncident.Type == core.ObservationBloodFound {
		until := func() bool {
			return c.cleaningIsDone
		}
		aic.SetEngaged(person, core.ActorStatusEngaged, until)
		c.Engine.GetAnimator().ActorEngagedAnimation(person, 'c', c.currentIncident.Location, 3.0, func() {
			game.GetMap().RemoveStimulusFromTile(c.currentIncident.Location, stimuli.StimulusBlood)
			aic.MarkAsCleaned(c.currentIncident)
			c.cleaningIsDone = true
		})
		return DeferredUpdate(func() bool {
			return c.cleaningIsDone
		})
	} else if c.currentIncident.Type == core.ObservationWeaponFound {
		itemAt := game.GetMap().ItemAt(c.currentIncident.Location)
		if itemAt != nil {
			game.PickUpItem(person)
			c.securedItems = append(c.securedItems, itemAt)
		}
		return c.cleanupCompleted()
	} else if c.currentIncident.Type == core.ObservationBodyFound { // TODO: other incident types
		body := game.GetMap().DownedActorAt(c.currentIncident.Location)
		if body != nil {
			person.DraggedBody = body
			body.IsBodyBagged = true
		}
		return c.cleanupCompleted()
	}

	return NextUpdateIn(0.3)
}

func (c *CleanupMovement) cleanupCompleted() core.AIUpdate {
	aic := c.Engine.GetAI()
	c.cleaningIsDone = true
	aic.MarkAsCleaned(c.currentIncident)
	return NextUpdateIn(0.3)
}
