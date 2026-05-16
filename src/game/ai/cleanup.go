package ai

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/stimuli"
)

type CleanupMovement struct {
	AIContext
	currentIncident core.IncidentReport
	cleaningIsDone  bool
	securedItems    []*core.Item
}

func (c *CleanupMovement) Status() core.ActorState { return core.ActorStatusCleanup }

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
	if c.currentIncident != core.EmptyReport {
		c.Engine.GetAI().UntrackCleanup(c.currentIncident.Hash())
	}
	person.AI.PopState()
	return NextUpdateIn(0.1)
}

func (c *CleanupMovement) NextAction() core.AIUpdate {
	person := c.Person

	if len(c.securedItems) > 0 || person.IsDraggingBody() {
		return c.gotoDropoff()
	}
	if c.currentIncident != core.EmptyReport && !c.cleaningIsDone {
		return person.AI.Movement.Action(c.currentIncident.Location, c)
	}
	person.AI.PopState()
	return NextUpdateIn(0.1)
}

func (c *CleanupMovement) gotoDropoff() core.AIUpdate {
	person := c.Person
	game := c.Engine.GetGame()
	currentMap := game.GetMap()
	if !currentMap.HasDropOffZone() {
		c.securedItems = []*core.Item{}
		if person.IsDraggingBody() {
			person.DraggedBody = nil
		}
		person.AI.PopState()
		return NextUpdateIn(0.1)
	}
	dropOffLocation := currentMap.GetNearestDropOffPosition(person.Pos())
	return person.AI.Movement.Action(dropOffLocation, c)
}

func (c *CleanupMovement) performCleanup() core.AIUpdate {
	if c.currentIncident == core.EmptyReport {
		c.Person.AI.PopState()
		return NextUpdateIn(0.1)
	}
	person := c.Person
	aic := c.Engine.GetAI()
	game := c.Engine.GetGame()
	if c.currentIncident.Type == core.ObservationBloodFound {
		until := func() bool {
			return c.cleaningIsDone
		}
		aic.SetEngrossed(person, until)
		c.Engine.GetAnimator().ActorEngagedAnimation(person, 'c', c.currentIncident.Location, 3.0, func() {
			game.GetMap().RemoveStimulusFromTile(c.currentIncident.Location, stimuli.StimulusBlood)
			c.Engine.GetAI().UntrackCleanup(c.currentIncident.Hash())
			c.cleaningIsDone = true
		})
		return DeferredUpdate(func() bool {
			return c.cleaningIsDone
		})
	} else if c.currentIncident.Type == core.ObservationWeaponFound || c.currentIncident.Type == core.ObservationMineFound {
		isItemAt := game.GetMap().IsItemAt(c.currentIncident.Location)
		if isItemAt {
			itemAt := game.GetMap().ItemAt(c.currentIncident.Location)
			game.PickUpItemAt(person, c.currentIncident.Location)
			c.securedItems = append(c.securedItems, itemAt)
		}
		return c.cleanupCompleted()
	} else if c.currentIncident.Type == core.ObservationBodyFound {
		isBodyAt := game.GetMap().IsDownedActorAt(c.currentIncident.Location)
		if isBodyAt {
			body := game.GetMap().DownedActorAt(c.currentIncident.Location)
			if !body.IsAlive() {
				person.DraggedBody = body
				body.IsBodyBagged = true
			}
		}
		return c.cleanupCompleted()
	}

	return NextUpdateIn(0.3)
}

func (c *CleanupMovement) cleanupCompleted() core.AIUpdate {
	c.cleaningIsDone = true
	c.Engine.GetAI().UntrackCleanup(c.currentIncident.Hash())
	return NextUpdateIn(0.3)
}
