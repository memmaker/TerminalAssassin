package core

import (
    "fmt"
    "github.com/memmaker/terminal-assassin/game/stimuli"
    "regexp"
    //"github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/gridmap"
    "github.com/memmaker/terminal-assassin/mapset"
)

type AIUpdate struct {
    DelayInSeconds  float64
    UpdatePredicate func() bool
}

var ManualDeferredUpdate = AIUpdate{DelayInSeconds: -1}

// AIStateHandler is the interface every AI behaviour state must implement.
// Storing the stack as []AIStateHandler gives compile-time safety and
// eliminates the type assertion in AIController.StateOf.
type AIStateHandler interface {
    NextAction() AIUpdate
}

type AIMovement interface {
    Action(location geometry.Point, handler MoveHandler) AIUpdate
    OnBlockedPath() AIUpdate
}

type MoveHandler interface {
    OnDestinationReached() AIUpdate
    OnCannotReachDestination() AIUpdate
}
type Observation string

const (
    ObservationNull                       Observation = "none"
    ObservationNearActiveIllegalIncident  Observation = "near active illegal incident"
    ObservationStrangeNoiseHeard          Observation = "strange noise heard"
    ObservationPersonAttacked             Observation = "person attacked"
    ObservationGunshot                    Observation = "gunshot heard"
    ObservationOpenCarry                  Observation = "open carry"
    ObservationMeleeNoises                Observation = "melee noise heard"
    ObservationBodyFound                  Observation = "body found"
    ObservationBloodFound                 Observation = "blood found"
    ObservationWeaponFound                Observation = "weapon found"
    ObservationTrespassing                Observation = "trespassing"
    ObservationTrespassingInHostileZone   Observation = "trespassing in hostile zone"
    ObservationExplosion                  Observation = "explosion"
    ObservationCombatSeen                 Observation = "open combat seen"
    ObservationDeath                      Observation = "death"
    ObservationUnconscious                Observation = "unconscious"
    ObservationIllegalAction              Observation = "illegal action"
    ObservationOngoingSuspiciousBehaviour Observation = "ongoing suspicious behaviour"
    ObservationDraggingBody               Observation = "dragging body"
    ObservationWearingCompromisedDisguise Observation = "wearing compromised disguise"
    ObservationDeviceDistraction          Observation = "device distraction"
    ObservationDownedSpeaker              Observation = "DLG_downed_speaker_00"
)

type IncidentReport struct {
    Type              Observation
    Location          geometry.Point
    Tick              uint64
    FinishedHandling  bool
    RegisteredHandler *Actor
    RegisteredCleaner *Actor
    RegisteredSnitch  *Actor
    KnownBy           mapset.Set[*Actor]
}

func (i IncidentReport) IsKnownByGuards() bool {
    for _, actor := range i.KnownBy.ToSlice() {
        if actor.Type == ActorTypeGuard {
            return true
        }
    }
    return false
}

var EmptyReport = IncidentReport{}

// speechPattern is compiled once; IsSpeech() is called frequently.
var speechPattern = regexp.MustCompile(`^DLG_[a-zA-Z0-9_-]+_[a-zA-Z0-9-]+_[0-9]+$`)

// Lookup sets replace long OR-chains.  Adding a new Observation only requires
// updating the relevant set(s) below — no predicate method needs editing.
var (
    observationSuspiciousLocations = map[Observation]struct{}{
        ObservationStrangeNoiseHeard: {},
    }
    observationDangerousLocations = map[Observation]struct{}{
        ObservationGunshot:     {},
        ObservationMeleeNoises: {},
        ObservationExplosion:   {},
        ObservationBloodFound:  {},
        ObservationWeaponFound: {},
        ObservationBodyFound:   {},
        ObservationCombatSeen:  {},
        ObservationDeath:       {},
        ObservationUnconscious: {},
    }
    observationSuspiciousActors = map[Observation]struct{}{
        ObservationIllegalAction: {},
        ObservationTrespassing:   {},
        ObservationOpenCarry:     {},
    }
    observationEvents = map[Observation]struct{}{
        ObservationStrangeNoiseHeard: {},
        ObservationTrespassing:       {},
        ObservationIllegalAction:     {},
        ObservationExplosion:         {},
        ObservationCombatSeen:        {},
        ObservationDeath:             {},
        ObservationGunshot:           {},
        ObservationPersonAttacked:    {},
        ObservationUnconscious:       {},
        ObservationDeviceDistraction: {},
    }
    observationNeedsCleanup = map[Observation]struct{}{
        ObservationBodyFound:   {},
        ObservationBloodFound:  {},
        ObservationWeaponFound: {},
    }
    observationIllegal = map[Observation]struct{}{
        ObservationTrespassing:       {},
        ObservationIllegalAction:     {},
        ObservationExplosion:         {},
        ObservationCombatSeen:        {},
        ObservationDeath:             {},
        ObservationUnconscious:       {},
        ObservationBloodFound:        {},
        ObservationWeaponFound:       {},
        ObservationBodyFound:         {},
        ObservationDeviceDistraction: {},
    }
    observationContacts = map[Observation]struct{}{
        ObservationTrespassing:                {},
        ObservationTrespassingInHostileZone:   {},
        ObservationOpenCarry:                  {},
        ObservationIllegalAction:              {},
        ObservationOngoingSuspiciousBehaviour: {},
        ObservationDraggingBody:               {},
        ObservationNearActiveIllegalIncident:  {},
        ObservationCombatSeen:                 {},
        ObservationWearingCompromisedDisguise: {},
    }
)

func (i Observation) IsSuspiciousLocation() bool {
    _, ok := observationSuspiciousLocations[i]
    return ok
}
func (i Observation) IsDangerousLocation() bool {
    _, ok := observationDangerousLocations[i]
    return ok
}
func (i Observation) IsSuspiciousActor() bool {
    _, ok := observationSuspiciousActors[i]
    return ok
}

// IsAnEvent returns true if the incident is an event, in contrast to a permanent state which has been observed.
func (i Observation) IsAnEvent() bool {
    _, ok := observationEvents[i]
    return ok
}

// NeedsCleanup returns true if the incident needs to be cleaned up by a guard after it has been handled.
func (i Observation) NeedsCleanup() bool {
    _, ok := observationNeedsCleanup[i]
    return ok
}

// IsIllegal returns true if the incident is a crime. This is used to determine if an actor next to these incidents
// is considered suspicious.
func (i Observation) IsIllegal() bool {
    _, ok := observationIllegal[i]
    return ok
}

func (i Observation) AbstractType() string {
    if i.IsAnEvent() {
        return "event"
    }
    if i.IsDangerousLocation() {
        return "obsDL"
    }
    if i.IsSuspiciousLocation() {
        return "obsSL"
    }
    if i.IsSuspiciousActor() {
        return "obsDA"
    }
    return "other"
}

func (i Observation) IsContact() bool {
    _, ok := observationContacts[i]
    return ok
}

func (i Observation) IsTrespassing() bool {
    return i == ObservationTrespassing || i == ObservationTrespassingInHostileZone
}

func (i Observation) IsOpenViolence() bool {
    return i == ObservationCombatSeen
}

func (i Observation) IsSpeech() bool {
    return speechPattern.MatchString(string(i))
}

func (i Observation) IsEnvironmentalToggle() bool {
    return i == ObservationDeviceDistraction
}

func (i IncidentReport) String() string {
    if i.KnownBy.Cardinality() == 0 {
        return fmt.Sprintf("%s at %v", i.Type, i.Location)
    }
    knowledgeGroup := ""
    for _, actor := range i.KnownBy.ToSlice() {
        knowledgeGroup += fmt.Sprintf(" - %s\n", actor.DebugDisplayName())
    }
    reportTitle := fmt.Sprintf("%s at %v - known by:\n", i.Type, i.Location)
    return reportTitle + knowledgeGroup
}
func (i IncidentReport) Hash() string {
    return fmt.Sprintf("%s:%v", i.Type, i.Location)
}

func (i IncidentReport) HasActiveHandler() bool {
    if i.RegisteredHandler == nil || i.RegisteredHandler.IsDowned() {
        return false
    }
    return true
}

type KnownObject interface {
    Description() string
}
type EffectSource struct {
    Actor  *Actor
    Item   *Item
    Object KnownObject
    Tile   gridmap.Tile
    // can be an actor himself (e.g. unarmed combat)
    // can be an actor using a weapon (e.g. knife to the head or sniper rifle)
    // can be an environmental effect (e.g. trap explosion or death tile)
}

func NewEffectSourceFromTile(tile gridmap.Tile) EffectSource {
    return EffectSource{
        Tile: tile,
    }
}
func (s EffectSource) WithItem(newItem *Item) EffectSource {
    s.Item = newItem
    return s
}

func (s EffectSource) ToCoDFromStim(sType stimuli.StimulusType) CoDDescription {
    switch sType {
    case stimuli.StimulusPiercingDamage:
        return s.ToCoDFromPiercingDamage()
    case stimuli.StimulusBluntDamage:
        return s.ToCoDFromBluntDamage()
    case stimuli.StimulusFire:
        return CoDBurned
    case stimuli.StimulusExplosionDamage:
        return CodExploded
    case stimuli.StimulusLethalPoison:
        return CoDPoisoned
    }
    return CoDDescription(fmt.Sprintf("killed by %s under mysterious circumstances", sType))
}

func (s EffectSource) ToCoDFromPiercingDamage() CoDDescription {
    if s.Item != nil {
        switch s.Item.Type {
        case ItemTypeSniperRifle:
            return CoDSnipered
        case ItemTypeAssaultRifle:
            return CoDAutoShot
        case ItemTypePistol:
            return CoDOnePistolRound
        case ItemTypeShotgun:
            return CoDShotGun
        case ItemTypeSubmachineGun:
            return CoDSubShot
        case ItemTypeMeleeSharp:
            return CoDStabbed
        case ItemTypeScrewdriver:
            return CoDPenetrated
        case ItemTypeKnife:
            return CoDStabbed
        default:
            return CoDDescription("pierced by %s")
        }
    }
    return CoDDescription("pierced")
}

func (s EffectSource) ToCoDFromBluntDamage() CoDDescription {
    if s.Item != nil {
        switch s.Item.Type {
        case ItemTypeCrowbar:
            return CoDDescription("battered with %s")
        case ItemTypeMeleeBlunt:
            return CoDDescription("bludgeoned with %s")
        case ItemTypeWrench:
            return CoDDescription("beaten with %s")
        default:
            return CoDDescription("blunt hit with %s")
        }
    }
    return CoDDescription("blunt object hit")
}

func NewEffectSourceUsedItem(user *Actor, item *Item) EffectSource {
    return EffectSource{
        Actor: user,
        Item:  item,
    }
}
func NewEffectSourceFromActor(actor *Actor) EffectSource {
    return EffectSource{
        Actor: actor,
    }
}
func NewEffectSourceFromItem(item *Item) EffectSource {
    return EffectSource{
        Item: item,
    }
}
func NewEffectSourceFromUsedObject(user *Actor, object KnownObject) EffectSource {
    return EffectSource{
        Actor:  user,
        Object: object,
    }
}

func NewEffectSourceFromObject(object KnownObject) EffectSource {
    return EffectSource{
        Object: object,
    }
}
