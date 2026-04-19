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
    activeProjectiles []*Projectile
}

func (g *ActionProvider) UseEquippedItemOnSelf(player *core.Actor) {
    // Drop the item at the player's current position (or the nearest free tile).
    // This is how a proximity mine becomes armed: the player activates it and
    // it is placed on the ground ready to trigger on the next actor contact.
    g.engine.GetGame().DropEquippedItem(player)
}

func NewActionProvider(engine services.Engine) *ActionProvider {
    return &ActionProvider{
        engine:            engine,
        activeProjectiles: make([]*Projectile, 0),
    }
}

func (g *ActionProvider) Update() {
    for i := len(g.activeProjectiles) - 1; i >= 0; i-- {
        projectile := g.activeProjectiles[i]
        projectile.Update()
        if projectile.IsDead() {
            g.activeProjectiles = append(g.activeProjectiles[:i], g.activeProjectiles[i+1:]...)
        }
    }
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
    if item.Type.CooldownSecs() > 0 {
        item.OnCooldown = true
        g.engine.Schedule(item.Type.CooldownSecs(), func() { item.OnCooldown = false })
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

    if !weapon.IsSilenced() {
        gunLightSource.Pos = attacker.Pos()
        g.flashDynamicLight(attacker.Pos(), gunLightSource, 0.1)
        m.SoundEventAt(attacker.Pos(), core.ObservationGunshot, weapon.NoiseRadius)
    }

    if weapon.AudioCue != "" {
        audio.PlayCueAt(weapon.AudioCue, attacker.Pos())
    }

    g.fireBullet(attacker, weapon, target)
    weapon.OnCooldown = true
    g.engine.Schedule(weapon.Type.CooldownSecs(), func() { weapon.OnCooldown = false })
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

    if !weapon.IsSilenced() {
        gunLightSource.Pos = attacker.Pos()
        g.flashDynamicLight(attacker.Pos(), gunLightSource, 0.18)
        m.SoundEventAt(attacker.Pos(), core.ObservationGunshot, weapon.NoiseRadius)
    }

    if weapon.AudioCue != "" {
        audio.PlayCueAt(weapon.AudioCue, attacker.Pos())
    }

    // for each bullet
    normalizedProjectileDirection := geometry.NewPointF(target.Sub(attacker.Pos()))
    normalizedProjectileDirection = normalizedProjectileDirection.Normalize()
    for i := uint8(0); i < core.ShotgunPelletCount; i++ {
        // add some randomness to the projectile direction
        rotationInDegrees := rand.Intn(core.SpreadShotDegrees) - (core.SpreadShotDegrees / 2)
        projectileDirection := normalizedProjectileDirection.Rotate(rotationInDegrees)
        // project the bullet
        projectileHitTarget := attacker.Pos().Add(projectileDirection.MulInt(weapon.ProjectileRange).ToPoint())
        g.fireBullet(attacker, weapon, projectileHitTarget)
    }
    weapon.OnCooldown = true
    g.engine.Schedule(weapon.Type.CooldownSecs(), func() { weapon.OnCooldown = false })
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
        m.IllegalActionAt(source, core.ObservationPersonAttacked)
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
    if person.EquippedItem == nil {
        return
    }
    m := g.engine.GetGame()
    item := person.EquippedItem
    println(fmt.Sprintf("%s used %s for a ranged attack", person.Name, item.Name))
    switch {
    case item.Type.IsSpreadShot():
        g.spreadShot(person, item, target)
    case item.Type.IsRemoteDetonated():
        g.rangedThrowRemote(person, item, target)
    case item.Type.IsThrowable():
        g.rangedThrow(person, item, target)
    default:
        g.rangedShot(person, item, target)
    }
    m.UpdateHUD()
}

func (g *ActionProvider) UseEquippedItemForMelee(person *core.Actor, target geometry.Point) {
    if person.EquippedItem == nil {
        return
    }
    m := g.engine.GetGame()
    item := person.EquippedItem
    println(fmt.Sprintf("%s used %s for a melee attack", person.Name, item.Name))
    if item.Type.IsMeleeTool() {
        g.toolUsage(person, item, target)
    } else {
        g.meleeAttack(person, item, target)
    }
    m.UpdateHUD()
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
