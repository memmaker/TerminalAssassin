package objects

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

func NewCorpseContainerAt(description string, symbol rune) *CorpseContainer {
	return &CorpseContainer{
		icon: symbol,
		Name: description,
	}
}

type CorpseContainer struct {
	ContainedActor  *core.Actor
	icon            rune
	Name            string
	position        geometry.Point
	GetOutPosition  geometry.Point
	IsUsedForHiding bool
}

func (cc *CorpseContainer) EncodeAsString() string {
	return cc.Name
}

func (cc *CorpseContainer) Description() string {
	return cc.Name
}

func (cc *CorpseContainer) ApplyStimulus(m services.Engine, stim stimuli.Stimulus) {}

func (cc *CorpseContainer) Pos() geometry.Point {
	return cc.position
}

func (cc *CorpseContainer) SetPos(pos geometry.Point) {
	cc.position = pos
}

func (cc *CorpseContainer) Style(st common.Style) common.Style {
	st = common.Style{Foreground: core.CurrentTheme.ObjectForeground, Background: st.Background}
	if cc.IsUsedForHiding {
		st = st.WithBg(core.CurrentTheme.ContainerHidingBackground)
	}
	if cc.ContainedActor != nil {
		st = st.WithBg(core.CurrentTheme.ContainerBodyBackground)
	}
	return st
}

func (cc *CorpseContainer) Action(m services.Engine, person *core.Actor) {
	currentMap := m.GetGame().GetMap()
	player := currentMap.Player
	if cc.IsUsedForHiding {
		cc.GetOut(currentMap, person)
		return
	} else if cc.hasSpace() && player.IsDraggingBody() {
		cc.PutDraggedBodyInContainer(m, player)
	} else if cc.hasSpace() && !player.IsDraggingBody() {
		cc.HidePerson(currentMap, person)
	}
}

func (cc *CorpseContainer) PutDraggedBodyInContainer(m services.Engine, holder *core.Actor) {
	currentMap := m.GetGame().GetMap()
	actorToContain := holder.DraggedBody
	holder.DraggedBody = nil
	cc.ContainedActor = actorToContain
	cc.ContainedActor.IsHidden = true
	currentMap.MoveDownedActor(actorToContain, cc.Pos())
}

func (cc *CorpseContainer) IsActionAllowed(m services.Engine, person *core.Actor) bool {
	return cc.hasSpace() || cc.IsUsedForHiding
}

func (cc *CorpseContainer) ActionDescription() string {
	if cc.IsUsedForHiding {
		return "Get out"
	} else if cc.hasSpace() {
		return "HideModal"
	}
	return "n/a"
}

func (cc *CorpseContainer) IsWalkable(*core.Actor) bool {
	return false
}

func (cc *CorpseContainer) IsTransparent() bool {
	return false
}

func (cc *CorpseContainer) IsPassableForProjectile() bool {
	return true
}

func (cc *CorpseContainer) Icon() rune {
	return cc.icon
}

func (cc *CorpseContainer) hasSpace() bool {
	return cc.ContainedActor == nil
}

func (cc *CorpseContainer) HidePerson(missionMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], person *core.Actor) {
	cc.GetOutPosition = person.Pos()
	cc.IsUsedForHiding = true
	cc.ContainedActor = person
	missionMap.MoveActor(person, cc.Pos())
	person.Status = core.ActorStatusInCloset
	person.IsHidden = true
}

func (cc *CorpseContainer) GetOut(missionMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], person *core.Actor) {
	cc.removePerson(person)
	cc.IsUsedForHiding = false
	cc.ContainedActor = nil
	missionMap.MoveActor(person, cc.GetOutPosition)
	person.Status = core.ActorStatusIdle
	person.IsHidden = false
}

func (cc *CorpseContainer) removePerson(person *core.Actor) {
	if cc.ContainedActor == person {
		cc.ContainedActor = nil
	}
}

