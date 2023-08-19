package objects

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

func NewAction(objectAt services.Object) *ObjectAction {
	return &ObjectAction{forObject: objectAt}
}

type ObjectAction struct {
	forObject services.Object
}

func (o *ObjectAction) Description(m services.Engine, person *core.Actor, actionAt geometry.Point) (rune, common.Style) {
	return o.forObject.Icon(), o.forObject.Style(common.DefaultStyle).WithBg(common.LegalActionGreen)
}
func (o *ObjectAction) Action(m services.Engine, person *core.Actor, actionAt geometry.Point) {
	o.forObject.Action(m, person)
}
func (o *ObjectAction) IsActionPossible(m services.Engine, person *core.Actor, actionAt geometry.Point) bool {
	return o.forObject.IsActionAllowed(m, person)
}
