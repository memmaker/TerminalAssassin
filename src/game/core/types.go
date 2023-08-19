package core

import (
	"fmt"
	"regexp"

	"github.com/memmaker/terminal-assassin/game/stimuli"
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

func (i IncidentReport) IsKnownByGuard() bool {
	for _, actor := range i.KnownBy.ToSlice() {
		if actor.Type == ActorTypeGuard {
			return true
		}
	}
	return false
}

var EmptyReport = IncidentReport{}

func (i Observation) IsSuspiciousLocation() bool {
	return i == ObservationStrangeNoiseHeard
}
func (i Observation) IsDangerousLocation() bool {
	return i == ObservationGunshot ||
		i == ObservationMeleeNoises ||
		i == ObservationExplosion ||
		i == ObservationBloodFound ||
		i == ObservationWeaponFound ||
		i == ObservationBodyFound ||
		i == ObservationCombatSeen ||
		i == ObservationDeath ||
		i == ObservationUnconscious
}
func (i Observation) IsSuspiciousActor() bool {
	return i == ObservationIllegalAction || i == ObservationTrespassing || i == ObservationOpenCarry
}

// IsAnEvent returns true if the incident is an event, in contrast to a permanent state which has been observed.
func (i Observation) IsAnEvent() bool {
	return i == ObservationStrangeNoiseHeard ||
		i == ObservationTrespassing ||
		i == ObservationIllegalAction ||
		i == ObservationExplosion ||
		i == ObservationCombatSeen ||
		i == ObservationDeath ||
		i == ObservationGunshot ||
		i == ObservationPersonAttacked ||
		i == ObservationUnconscious ||
		i == ObservationDeviceDistraction
}

// NeedsCleanup returns true if the incident needs to be cleaned up by a guard after it has been handled.
func (i Observation) NeedsCleanup() bool {
	return i == ObservationBodyFound || i == ObservationBloodFound || i == ObservationWeaponFound
}

// IsIllegal returns true if the incident is a crime. This is used to determine if an actor next to these incidents
// is considered suspicious.
func (i Observation) IsIllegal() bool {
	return i == ObservationTrespassing ||
		i == ObservationIllegalAction ||
		i == ObservationExplosion ||
		i == ObservationCombatSeen ||
		i == ObservationDeath ||
		i == ObservationUnconscious ||
		i == ObservationBloodFound ||
		i == ObservationWeaponFound ||
		i == ObservationBodyFound ||
		i == ObservationDeviceDistraction
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
	return i == ObservationTrespassing ||
		i == ObservationTrespassingInHostileZone ||
		i == ObservationOpenCarry ||
		i == ObservationIllegalAction ||
		i == ObservationOngoingSuspiciousBehaviour ||
		i == ObservationDraggingBody ||
		i == ObservationNearActiveIllegalIncident ||
		i == ObservationCombatSeen ||
		i == ObservationWearingCompromisedDisguise
}

func (i Observation) IsTrespassing() bool {
	return i == ObservationTrespassing || i == ObservationTrespassingInHostileZone
}

func (i Observation) IsOpenViolence() bool {
	return i == ObservationCombatSeen
}

func (i Observation) IsSpeech() bool {
	pattern := regexp.MustCompile(`^DLG_[a-zA-Z0-9_-]+_[a-zA-Z0-9-]+_[0-9]+$`)
	return pattern.MatchString(string(i))
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

func (i IncidentReport) IsKnownByGuards() bool {
	for _, actor := range i.KnownBy.ToSlice() {
		if actor.Type == ActorTypeGuard {
			return true
		}
	}
	return false
}

func (i IncidentReport) HasActiveHandler() bool {
	if i.RegisteredHandler == nil || i.RegisteredHandler.IsDowned() {
		return false
	}
	return false
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
