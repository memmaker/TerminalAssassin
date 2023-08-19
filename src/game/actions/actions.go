package actions

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"math/rand"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

type ActionProvider struct {
	engine            services.Engine
	itemActions       map[core.ItemActionType]TargetedItemAction
	activeProjectiles []*Projectile
}

func (g *ActionProvider) UseEquippedItemOnSelf(player *core.Actor) {
	// do nothing for now
}

func NewActionProvider(engine services.Engine) *ActionProvider {
	actions := &ActionProvider{
		engine:            engine,
		itemActions:       make(map[core.ItemActionType]TargetedItemAction, 0),
		activeProjectiles: make([]*Projectile, 0),
	}

	actions.itemActions = map[core.ItemActionType]TargetedItemAction{
		core.ActionTypeMeleeAttack: actions.meleeAttack,
		core.ActionTypeMeleeUse:    actions.meleeAttack,
		core.ActionTypeTool:        actions.toolUsage,
		core.ActionTypeShot:        actions.rangedShot,
		core.ActionTypeSpreadShot:  actions.spreadShot,
		core.ActionTypeThrow:       actions.rangedThrow,
		core.ActionTypeThrowRemote: actions.rangedThrowRemote,
	}
	return actions
}

func (g *ActionProvider) Update() {
	for i := len(g.activeProjectiles) - 1; i >= 0; i-- {
		projectile := g.activeProjectiles[i]
		projectile.Update()
		if projectile.IsDead() {
			g.activeProjectiles = append(g.activeProjectiles[:i], g.activeProjectiles[i+1:]...)
		}
	}
	/*
		if len(g.activeProjectiles) > 0 {
			println("Active projectiles: ", len(g.activeProjectiles))
		}
	*/
}

func (g *ActionProvider) Draw(con console.CellInterface) {
	for _, projectile := range g.activeProjectiles {
		if projectile.IsDead() {
			continue
		}
		projectile.Draw(con)
	}
}

type TargetedItemAction func(source *core.Actor, item *core.Item, target geometry.Point)

func (g *ActionProvider) meleeAttack(source *core.Actor, item *core.Item, target geometry.Point) {
	m := g.engine.GetGame()
	m.IllegalActionAt(source.Pos(), core.ObservationCombatSeen)
	g.TryFeedbackForImpact("flesh_hit", source.Pos(), target, 5)
	m.SendTriggerStimuli(source, item, target, core.TriggerOnMeleeAttack)
	if item.DelayBetweenShotsInSecs > 0 {
		item.OnCooldown = true
		g.engine.Schedule(item.DelayBetweenShotsInSecs, func() { item.OnCooldown = false })
	}
}

func (g *ActionProvider) toolUsage(source *core.Actor, item *core.Item, target geometry.Point) {
	m := g.engine.GetGame()
	m.SendTriggerStimuli(source, item, target, core.TriggerOnToolUsage)
}

var gunLightSource = &gridmap.LightSource{
	Pos:          geometry.Point{},
	Radius:       10,
	Color:        common.RGBAColor{R: 4, G: 4, B: 4, A: 1.0},
	MaxIntensity: 15,
}

func (g *ActionProvider) rangedShot(attacker *core.Actor, weapon *core.Item, target geometry.Point) {
	m := g.engine.GetGame()
	aic := g.engine.GetAI()
	audio := g.engine.GetAudio()
	if aic.IsControlledByAI(attacker) { // give the AI unlimited ammo
		weapon.Uses -= 1
	}

	m.IllegalActionAt(attacker.Pos(), core.ObservationCombatSeen)

	if weapon.IsSilenced {
		if weapon.SilencedCue != "" {
			audio.PlayCueAt(weapon.SilencedCue, attacker.Pos())
		}
	} else {
		gunLightSource.Pos = attacker.Pos()
		g.flashDynamicLight(attacker.Pos(), gunLightSource, 0.1)
		if weapon.AudioCue != "" {
			audio.PlayCueAt(weapon.AudioCue, attacker.Pos())
		}
		if weapon.NoiseRadius > 0 {
			m.SoundEventAt(attacker.Pos(), core.ObservationGunshot, weapon.NoiseRadius)
		}
	}
	g.fireBullet(attacker, weapon, target)
	weapon.OnCooldown = true
	g.engine.Schedule(weapon.DelayBetweenShotsInSecs, func() { weapon.OnCooldown = false })
}

func (g *ActionProvider) flashDynamicLight(pos geometry.Point, lightSource *gridmap.LightSource, timeInSecs float64) {
	m := g.engine.GetGame()
	if !m.GetConfig().LightSources {
		return
	}
	currentMap := m.GetMap()
	currentMap.AddDynamicLightSource(pos, lightSource)
	currentMap.UpdateDynamicLights()
	g.engine.Schedule(timeInSecs, func() { currentMap.RemoveDynamicLightAt(pos); currentMap.UpdateDynamicLights() })
}
func (g *ActionProvider) spreadShot(attacker *core.Actor, weapon *core.Item, target geometry.Point) {
	m := g.engine.GetGame()
	aic := g.engine.GetAI()
	audio := g.engine.GetAudio()
	if aic.IsControlledByAI(attacker) { // give the AI unlimited ammo
		weapon.Uses -= 1
	}

	m.IllegalActionAt(attacker.Pos(), core.ObservationCombatSeen)

	if weapon.IsSilenced {
		if weapon.SilencedCue != "" {
			audio.PlayCueAt(weapon.SilencedCue, attacker.Pos())
		}
	} else {
		gunLightSource.Pos = attacker.Pos()
		g.flashDynamicLight(attacker.Pos(), gunLightSource, 0.18)
		if weapon.AudioCue != "" {
			audio.PlayCueAt(weapon.AudioCue, attacker.Pos())
		}
		if weapon.NoiseRadius > 0 {
			m.SoundEventAt(attacker.Pos(), core.ObservationGunshot, weapon.NoiseRadius)
		}
	}

	// for each bullet
	normalizedProjectileDirection := geometry.NewPointF(target.Sub(attacker.Pos()))
	normalizedProjectileDirection = normalizedProjectileDirection.Normalize()
	for i := uint8(0); i < weapon.ProjectileCount; i++ {
		// add some randomness to the projectile direction
		rotationInDegrees := rand.Intn(weapon.SpreadInDegrees) - (weapon.SpreadInDegrees / 2)
		projectileDirection := normalizedProjectileDirection.Rotate(rotationInDegrees)
		// project the bullet
		projectileHitTarget := attacker.Pos().Add(projectileDirection.MulInt(weapon.ProjectileRange).ToPoint())
		g.fireBullet(attacker, weapon, projectileHitTarget)
	}
	weapon.OnCooldown = true
	g.engine.Schedule(weapon.DelayBetweenShotsInSecs, func() { weapon.OnCooldown = false })
	// end
}

func (g *ActionProvider) fireBullet(source *core.Actor, weapon *core.Item, target geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()
	los := currentMap.LineOfSight(source.FoVSource(), target)
	if len(los) <= 1 { // no path or only path to self -> misfire
		return
	}
	bullet := NewBulletFor(g.engine, source, weapon)
	bullet.StartTravel(los)
	g.activeProjectiles = append(g.activeProjectiles, bullet)
}

func (g *ActionProvider) TryFeedbackForImpact(audioCue string, source geometry.Point, hitLocation geometry.Point, distance int) {
	m := g.engine.GetGame()
	currentMap := m.GetMap()
	debugMessage := fmt.Sprintf("Impact at %v", hitLocation)
	if currentMap.IsActorAt(hitLocation) {
		directionVector := geometry.NewPointF(hitLocation.Sub(source))
		directionVector = directionVector.Normalize().Mul(1.5)
		bloodPos := hitLocation.Add(directionVector.ToPoint())
		currentMap.AddStimulusToTile(bloodPos, stimuli.Stim{StimType: stimuli.StimulusBlood, StimForce: 5})
		g.engine.GetAudio().PlayCue(audioCue)
		m.IllegalActionAt(hitLocation, core.ObservationPersonAttacked)
		debugMessage = fmt.Sprintf("Impact at %v hitting %s", hitLocation, currentMap.ActorAt(hitLocation).DebugDisplayName())
	}
	println(debugMessage)
	m.SoundEventAt(hitLocation, core.ObservationStrangeNoiseHeard, distance)
}

func (g *ActionProvider) rangedThrowRemote(source *core.Actor, item *core.Item, target geometry.Point) {
	thrownItem := item
	g.rangedThrow(source, thrownItem, target)

	itemFactory := g.engine.GetItemFactory()
	remote := itemFactory.CreateRemoteDetonator(source, thrownItem)

	source.Inventory.AddItem(remote)
	remote.HeldBy = source

	source.EquippedItem = remote
	g.engine.Schedule(0.32, func() { remote.OnCooldown = false })
}

func (g *ActionProvider) rangedThrow(source *core.Actor, item *core.Item, target geometry.Point) {
	audio := g.engine.GetAudio()
	// basic bookkeeping
	source.EquippedItem = nil
	source.Inventory.RemoveItem(item)
	currentMap := g.engine.GetGame().GetMap()
	los := currentMap.LineOfSight(source.FoVSource(), target)
	if len(los) <= 1 { // no path or only path to self -> misfire
		return
	}
	bullet := NewThrownItem(g.engine, source, item)
	bullet.StartTravel(los)
	g.activeProjectiles = append(g.activeProjectiles, bullet)
	// start animation
	audio.PlayCue("throw")
}

func (g *ActionProvider) UseEquippedItemAtRange(person *core.Actor, target geometry.Point) {
	m := g.engine.GetGame()
	if person.EquippedItem.RangedAttack == core.NoAction {
		return
	}

	if action, ok := g.itemActions[person.EquippedItem.RangedAttack]; ok {
		println(fmt.Sprintf("%s used %s for a ranged attack", person.Name, person.EquippedItem.Name))
		action(person, person.EquippedItem, target)
		m.UpdateHUD()
	}
	return
}

func (g *ActionProvider) UseEquippedItemForMelee(person *core.Actor, target geometry.Point) {
	m := g.engine.GetGame()
	if person.EquippedItem.MeleeAttack == core.NoAction {
		return
	}
	if action, ok := g.itemActions[person.EquippedItem.MeleeAttack]; ok {
		println(fmt.Sprintf("%s used %s for a melee attack", person.Name, person.EquippedItem.Name))
		action(person, person.EquippedItem, target)
		m.UpdateHUD()
	}
	return
}

func (g *ActionProvider) Prod(person *core.Actor, aimPoint geometry.Point) {
	m := g.engine.GetGame()
	currentMap := m.GetMap()
	ownPos := person.Pos()
	directionToTarget := aimPoint.Sub(ownPos)
	actorAt, isActorAt := currentMap.TryGetActorAt(aimPoint)
	if isActorAt {
		m.TryPushActorInDirection(actorAt, directionToTarget)
	}
	downedActorAt, isDownedActorAt := currentMap.TryGetDownedActorAt(aimPoint)
	if isDownedActorAt {
		m.TryPushActorInDirection(downedActorAt, directionToTarget)
	}
}
func (g *ActionProvider) Reset() {
	g.activeProjectiles = []*Projectile{}
}
