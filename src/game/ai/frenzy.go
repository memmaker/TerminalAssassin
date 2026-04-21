package ai

import (
    "fmt"
    "math/rand"

    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/stimuli"
    "github.com/memmaker/terminal-assassin/geometry"
)

// FrenzyMovement is the AI state for a frenzied actor.
// The actor blindly attacks the nearest reachable actor every tick,
// using ranged weapons when available, otherwise melee.
// The effect lasts for frenzyDurationSeconds of game time.
type FrenzyMovement struct {
    AIContext
    target          *core.Actor
    elapsedSeconds  float64
    lastUpdateDelay float64
}

const frenzyDurationSeconds = 30.0

func (f *FrenzyMovement) OnDestinationReached() core.AIUpdate {
    return f.timedUpdate(0.3)
}

func (f *FrenzyMovement) OnCannotReachDestination() core.AIUpdate {
    // Pick a new target next tick.
    f.target = nil
    return f.timedUpdate(0.5)
}

// timedUpdate accumulates elapsed game-time and returns the next AI update.
func (f *FrenzyMovement) timedUpdate(delaySecs float64) core.AIUpdate {
    timeFactor := f.Engine.GetTimeFactor()
    if timeFactor <= 0 {
        timeFactor = 1
    }
    f.elapsedSeconds += f.lastUpdateDelay * timeFactor
    f.lastUpdateDelay = delaySecs
    return NextUpdateIn(delaySecs)
}

func (f *FrenzyMovement) NextAction() core.AIUpdate {
    person := f.Person
    person.Status = core.ActorStatusFrenzy

    if !person.IsPredator() && f.elapsedSeconds >= frenzyDurationSeconds {
        f.exitFrenzy(person)
        return NextUpdateIn(0.4)
    }

    // Pick nearest visible, alive actor as target.
    if f.target == nil || f.target.IsDowned() || !person.CanSeeActor(f.target) {
        f.target = f.findNearestTarget()
    }
    if f.target == nil {
        // Nobody around — wander randomly.
        currentMap := f.Engine.GetGame().GetMap()
        neighbors := currentMap.GetAllCardinalNeighbors(person.Pos())
        if len(neighbors) > 0 {
            dest := neighbors[rand.Intn(len(neighbors))]
            return person.AI.Movement.Action(dest, f)
        }
        return f.timedUpdate(0.5)
    }

    currentMap := f.Engine.GetGame().GetMap()

    // Try ranged attack first.
    if person.EquipWeapon() && person.EquippedItem.IsRangedWeapon() &&
        person.CanSeeActor(f.target) &&
        currentMap.IsPathPassableForProjectile(person.Pos(), f.target.Pos()) {
        return f.handleFiring(person)
    }

    // Close distance for melee.
    dist := geometry.DistanceChebyshev(person.Pos(), f.target.Pos())
    if dist <= 1 {
        return f.handleMelee(person)
    }

    return person.AI.Movement.Action(f.target.Pos(), f)
}

func (f *FrenzyMovement) findNearestTarget() *core.Actor {
    person := f.Person
    currentMap := f.Engine.GetGame().GetMap()
    var closest *core.Actor
    closestDist := int(^uint(0) >> 1) // max int
    for _, actor := range currentMap.Actors() {
        if actor == person || actor.IsDowned() || !person.CanSeeActor(actor) {
            continue
        }
        d := geometry.DistanceManhattan(person.Pos(), actor.Pos())
        if d < closestDist {
            closestDist = d
            closest = actor
        }
    }
    return closest
}

func (f *FrenzyMovement) handleFiring(person *core.Actor) core.AIUpdate {
    person.LookDirection = geometry.DirectionVectorToAngleInDegrees(f.target.Pos().Sub(person.Pos()))

    game := f.Engine.GetGame()
    actions := game.GetActions()
    actions.UseEquippedItemAtRange(person, f.target.Pos())

    game.IllegalActionAt(person.Pos(), core.ObservationIllegalAction)

    return f.timedUpdate(rand.Float64()*0.4 + 0.3)
}

func (f *FrenzyMovement) handleMelee(person *core.Actor) core.AIUpdate {
    person.LookDirection = geometry.DirectionVectorToAngleInDegrees(f.target.Pos().Sub(person.Pos()))

    game := f.Engine.GetGame()
    actions := game.GetActions()

    if person.IsPredator() && !f.target.IsDowned() {
        // Predator: blood-splatter animation, then kill.
        victim := f.target
        victimPos := victim.Pos()
        done := false
        aic := f.Engine.GetAI()

        aic.SetEngaged(person, core.ActorStatusEngaged, func() bool { return done })
        aic.SetEngaged(victim, core.ActorStatusVictimOfEngagement, func() bool { return done })

        f.Engine.GetAnimator().ActorEngagedIllegalAnimation(person, core.GlyphBlood, victimPos, 3.0, func() {
            game.ApplyStimulusToActor(victim, core.NewEffectSourceFromActor(person),
                stimuli.Stim{StimType: stimuli.StimulusPiercingDamage, StimForce: 30})
            done = true
        }, func() { // cancelled
            done = true
        })

        return f.timedUpdate(3.1)
    } else if person.EquippedItem != nil && person.EquippedItem.Type.HasMeleeAction() {
        actions.UseEquippedItemForMelee(person, f.target.Pos())
    } else {
        // Unarmed: apply blunt damage directly.
        game.ApplyStimulusToActor(f.target, core.NewEffectSourceFromActor(person),
            stimuli.Stim{StimType: stimuli.StimulusBluntDamage, StimForce: 20})
    }

    game.IllegalActionAt(person.Pos(), core.ObservationIllegalAction)

    return f.timedUpdate(rand.Float64()*0.3 + 0.4)
}

func (f *FrenzyMovement) exitFrenzy(person *core.Actor) {
    println(fmt.Sprintf("%s frenzy ended — collapsing", person.DebugDisplayName()))
    person.AI.PopState()
    game := f.Engine.GetGame()
    game.SendToSleep(person)
}
