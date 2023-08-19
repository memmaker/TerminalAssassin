package actions

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Projectile struct {
	Pos              geometry.Point
	WrappedItem      *core.Item
	DestroyOnImpact  bool
	travelPath       []geometry.Point
	currentPathIndex int
	tickCounter      int
	isDead           bool
	ItemIsWeapon     bool
	engine           services.Engine
	origin           geometry.Point
	User             *core.Actor
}

func NewThrownItem(engine services.Engine, source *core.Actor, item *core.Item) *Projectile {
	return &Projectile{
		engine:      engine,
		WrappedItem: item,
		User:        source,
		tickCounter: 1,
	}
}

func NewBulletFor(engine services.Engine, source *core.Actor, weapon *core.Item) *Projectile {
	return &Projectile{
		engine:       engine,
		WrappedItem:  weapon,
		User:         source,
		ItemIsWeapon: true,
		tickCounter:  1,
	}
}
func (p *Projectile) IsDead() bool {
	return p.isDead
}

func (p *Projectile) StartTravel(path []geometry.Point) {
	p.origin = path[0]
	p.travelPath = path[1:]
	p.currentPathIndex = 0
}

func (p *Projectile) Update() {
	if p.tickCounter > 0 {
		p.tickCounter--
		return
	}

	p.tickCounter = 1

	currentPos := p.travelPath[p.currentPathIndex]
	game := p.engine.GetGame()
	currentMap := game.GetMap()
	endOfFlightPathReached := p.currentPathIndex == len(p.travelPath)-1
	hitSomething := !currentMap.IsPassableForProjectile(currentPos)
	if p.ItemIsWeapon && (endOfFlightPathReached || hitSomething) {
		p.onHitWithBullet(currentPos)
		return
	} else if !p.ItemIsWeapon && (endOfFlightPathReached || hitSomething) {
		p.onHitWithItem(currentPos)
		return
	}
	p.onTravel(currentPos)
	p.currentPathIndex++
}

func (p *Projectile) onHitWithBullet(pos geometry.Point) {
	game := p.engine.GetGame()
	actions := game.GetActions()
	actions.TryFeedbackForImpact("bullet_hit_head", p.travelPath[0], pos, 5)
	if effects, ok := p.WrappedItem.TriggerEffects[core.TriggerOnRangedShotHit]; ok {
		game.Apply(pos, core.NewEffectSourceUsedItem(p.User, p.WrappedItem), effects)
	}
	p.die(pos)
}

func (p *Projectile) onHitWithItem(pos geometry.Point) {
	game := p.engine.GetGame()
	actions := game.GetActions()
	actions.TryFeedbackForImpact("flesh_hit", p.travelPath[0], pos, 5)
	if effects, ok := p.WrappedItem.TriggerEffects[core.TriggerOnItemImpact]; ok {
		if effects.DestroyOnApplication {
			p.DestroyOnImpact = true
		}
		game.Apply(pos, core.NewEffectSourceUsedItem(p.User, p.WrappedItem), effects)
	}
	if effects, ok := p.WrappedItem.TriggerEffects[core.TriggerAfterItemImpact]; ok {
		if effects.DestroyOnApplication {
			p.DestroyOnImpact = true
		}
		game.Apply(pos, core.NewEffectSourceUsedItem(p.User, p.WrappedItem), effects)
	}
	p.die(pos)
}
func (p *Projectile) onTravel(pos geometry.Point) {
	if effects, ok := p.WrappedItem.TriggerEffects[core.TriggerOnFlightpath]; ok {
		game := p.engine.GetGame()
		game.Apply(pos, core.NewEffectSourceUsedItem(p.User, p.WrappedItem), effects)
	}
}

func (p *Projectile) Draw(con console.CellInterface) {
	if p.currentPathIndex >= len(p.travelPath) {
		return
	}
	symbol := '*'
	if p.WrappedItem != nil && !p.ItemIsWeapon {
		symbol = p.WrappedItem.Icon()
	}

	worldPos := p.travelPath[p.currentPathIndex]
	mapHeight := p.engine.MapWindowHeight()
	camera := p.engine.GetGame().GetCamera()
	screenPos := camera.WorldToScreen(worldPos)
	if screenPos.Y >= mapHeight {
		return
	}
	cellAt := con.AtSquare(screenPos)
	projectileStyle := cellAt.Style.WithFg(common.Black)
	if !p.ItemIsWeapon {
		projectileStyle = cellAt.Style.WithFg(p.WrappedItem.DefinedStyle.Foreground)
	}
	con.SetSquare(screenPos, common.Cell{Rune: symbol, Style: projectileStyle})
}

func (p *Projectile) die(hitLocation geometry.Point) {
	p.isDead = true
	game := p.engine.GetGame()
	if p.DestroyOnImpact {
		game.Destroy(p.WrappedItem)
	} else if !p.ItemIsWeapon {
		origin := p.origin
		if p.currentPathIndex > 0 {
			origin = p.travelPath[p.currentPathIndex-1]
		}
		game.PlaceItemWithOrigin(origin, hitLocation, p.WrappedItem)
	}

	p.WrappedItem = nil
	p.engine = nil
	p.User = nil
}
