package core

import (
    "fmt"
    "strconv"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/game/stimuli"
    "github.com/memmaker/terminal-assassin/geometry"
    rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

type ItemType int

// itemTypeNames is the single source of truth for ItemType ↔ string mapping.
// itemTypeByName is the reverse map, built once in init().
// ItemTypeMessage is intentionally omitted — it has no serialised string form.
var itemTypeNames = map[ItemType]string{
    ItemTypeCommon:              "common",
    ItemTypePistol:              "pistol",
    ItemTypeShotgun:             "shotgun",
    ItemTypeSniperRifle:         "sniper_rifle",
    ItemTypeAssaultRifle:        "assault_rifle",
    ItemTypeSubmachineGun:       "submachine_gun",
    ItemTypeMeleeSharp:          "melee_sharp",
    ItemTypeMeleeBlunt:          "melee_blunt",
    ItemTypeTool:                "tool",
    ItemTypeLethalPoison:        "lethal_poison",
    ItemTypeEmeticPoison:        "emetic_poison",
    ItemTypeSleepPoison:         "sleep_poison",
    ItemTypeScrewdriver:         "screwdriver",
    ItemTypeWrench:              "wrench",
    ItemTypeTaser:               "taser",
    ItemTypeDartGun:             "dart_gun",
    ItemTypeRemoteTaser:         "remote_taser",
    ItemTypeCrowbar:             "crowbar",
    ItemTypeKnife:               "knife",
    ItemTypePianoWire:           "piano_wire",
    ItemTypeMechanicalLockpick:  "mechanical_lockpick",
    ItemTypeElectronicLockpick:  "electronic_lockpick",
    ItemTypeCleaner:             "cleaner",
    ItemTypeKey:                 "key",
    ItemTypeKeyCard:             "key_card",
    ItemTypeLoot:                "loot",
    ItemTypeExplosive:           "explosive",
    ItemTypeRemoteExplosive:     "remote_explosive",
    ItemTypeProximityMine:       "proximity_mine",
    ItemTypeLethalPoisonGrenade: "lethal_poison_grenade",
    ItemTypeSleepPoisonGrenade:  "sleep_poison_grenade",
    ItemTypeSmokeGrenade:        "smoke_grenade",
    ItemTypeLethalPoisonMine:    "lethal_poison_mine",
    ItemTypeSleepPoisonMine:     "sleep_poison_mine",
    ItemTypeCamera:              "camera",
    ItemTypeShovel:              "shovel",
    ItemTypeFlashlight:          "flashlight",
}

var itemTypeByName map[string]ItemType

func init() {
    itemTypeByName = make(map[string]ItemType, len(itemTypeNames))
    for k, v := range itemTypeNames {
        itemTypeByName[v] = k
    }
}

func (t ItemType) ToString() string {
    if s, ok := itemTypeNames[t]; ok {
        return s
    }
    return "unknown"
}

func NewItemTypeFromString(text string) ItemType {
    if t, ok := itemTypeByName[text]; ok {
        return t
    }
    return ItemTypeCommon
}

const (
    ItemTypeCommon ItemType = iota
    ItemTypePistol
    ItemTypeShotgun
    ItemTypeSniperRifle
    ItemTypeAssaultRifle
    ItemTypeSubmachineGun
    ItemTypeMeleeSharp
    ItemTypeMeleeBlunt
    ItemTypeTool
    ItemTypeLethalPoison
    ItemTypeEmeticPoison
    ItemTypeSleepPoison
    ItemTypeScrewdriver
    ItemTypeWrench
    ItemTypeTaser
    ItemTypeDartGun
    ItemTypeRemoteTaser
    ItemTypeCrowbar
    ItemTypeKnife
    ItemTypePianoWire
    ItemTypeMechanicalLockpick
    ItemTypeElectronicLockpick
    ItemTypeCleaner
    ItemTypeKey
    ItemTypeKeyCard
    ItemTypeLoot
    ItemTypeExplosive
    ItemTypeRemoteExplosive
    ItemTypeProximityMine
    ItemTypeLethalPoisonGrenade
    ItemTypeSleepPoisonGrenade
    ItemTypeLethalPoisonMine
    ItemTypeSleepPoisonMine
    ItemTypeSmokeGrenade
    ItemTypeMessage
    ItemTypeCamera
    ItemTypeShovel
    ItemTypeFlashlight
)

// itemTypeTraits holds every per-type capability flag and value in one place.
// Adding a new ItemType only requires one entry in itemTraitsTable below.
type itemTypeTraits struct {
    hasMeleeAction          bool
    isMeleeTool             bool // tool-use melee (stimuli via TriggerOnToolUsage)
    meleeDecreaseUses       bool // consumes a use on melee activation
    hasRangedAction         bool
    isThrowable             bool
    canSelfActivate         bool // place / arm at own tile (e.g. proximity mines)
    isRemoteDetonated       bool
    isSpreadShot            bool
    isMeleeWeapon           bool    // physical strike weapon (for IsWeapon / IsMeleeWeapon)
    isRangedWeapon          bool    // firearm (for IsRangedWeapon / IsWeapon)
    isAutomaticRangedWeapon bool    // full-auto (for IsAutomaticRangedWeapon / HasBurstWeapon)
    cooldownSecs            float64 // 0 → default minimum (0.07 s)
    scopeFoV                float64 // 0 → no scope; >0 → scoped FoV angle in degrees
}

// itemTraitsTable is the single source of truth for every ItemType capability.
// All predicate methods below are one-line lookups into this table.
var itemTraitsTable = map[ItemType]itemTypeTraits{
    ItemTypeCommon:              {hasRangedAction: true, isThrowable: true},
    ItemTypePistol:              {hasMeleeAction: true, hasRangedAction: true, isRangedWeapon: true, cooldownSecs: 0.30},
    ItemTypeShotgun:             {hasMeleeAction: true, hasRangedAction: true, isRangedWeapon: true, isSpreadShot: true, cooldownSecs: 1.50},
    ItemTypeSniperRifle:         {hasMeleeAction: true, hasRangedAction: true, isRangedWeapon: true, cooldownSecs: 2.00, scopeFoV: 20.0},
    ItemTypeAssaultRifle:        {hasMeleeAction: true, hasRangedAction: true, isRangedWeapon: true, isAutomaticRangedWeapon: true, cooldownSecs: 0.20},
    ItemTypeSubmachineGun:       {hasMeleeAction: true, hasRangedAction: true, isRangedWeapon: true, isAutomaticRangedWeapon: true, cooldownSecs: 0.10},
    ItemTypeMeleeSharp:          {hasMeleeAction: true, hasRangedAction: true, isThrowable: true, isMeleeWeapon: true},
    ItemTypeMeleeBlunt:          {hasMeleeAction: true, hasRangedAction: true, isThrowable: true, isMeleeWeapon: true},
    ItemTypeTool:                {hasMeleeAction: true, isMeleeTool: true, meleeDecreaseUses: true},
    ItemTypeLethalPoison:        {hasMeleeAction: true, isMeleeTool: true, meleeDecreaseUses: true},
    ItemTypeEmeticPoison:        {hasMeleeAction: true, isMeleeTool: true, meleeDecreaseUses: true},
    ItemTypeSleepPoison:         {hasMeleeAction: true, isMeleeTool: true, meleeDecreaseUses: true},
    ItemTypeScrewdriver:         {hasMeleeAction: true, hasRangedAction: true, isThrowable: true, isMeleeWeapon: true},
    ItemTypeWrench:              {hasMeleeAction: true, hasRangedAction: true, isThrowable: true, isMeleeWeapon: true},
    ItemTypeTaser:               {hasMeleeAction: true, meleeDecreaseUses: true, cooldownSecs: 0.20},
    ItemTypeDartGun:             {hasRangedAction: true, cooldownSecs: 2.00},
    ItemTypeRemoteTaser:         {hasRangedAction: true, isThrowable: true, isRemoteDetonated: true},
    ItemTypeCrowbar:             {hasMeleeAction: true, hasRangedAction: true, isThrowable: true, isMeleeWeapon: true},
    ItemTypeKnife:               {hasMeleeAction: true, hasRangedAction: true, isThrowable: true, isMeleeWeapon: true},
    ItemTypePianoWire:           {hasMeleeAction: true, isMeleeWeapon: true},
    ItemTypeCleaner:             {hasMeleeAction: true, isMeleeTool: true, meleeDecreaseUses: true},
    ItemTypeLoot:                {hasRangedAction: true, isThrowable: true},
    ItemTypeExplosive:           {hasRangedAction: true, isThrowable: true},
    ItemTypeRemoteExplosive:     {hasRangedAction: true, isThrowable: true, isRemoteDetonated: true},
    ItemTypeProximityMine:       {hasRangedAction: true, isThrowable: true, canSelfActivate: true},
    ItemTypeLethalPoisonGrenade: {hasRangedAction: true, isThrowable: true},
    ItemTypeSleepPoisonGrenade:  {hasRangedAction: true, isThrowable: true},
    ItemTypeSmokeGrenade:        {hasRangedAction: true, isThrowable: true},
    ItemTypeLethalPoisonMine:    {hasRangedAction: true, isThrowable: true, canSelfActivate: true},
    ItemTypeSleepPoisonMine:     {hasRangedAction: true, isThrowable: true, canSelfActivate: true},
    ItemTypeFlashlight:          {hasMeleeAction: true, scopeFoV: 60.0},
    // ItemTypeMechanicalLockpick, ItemTypeElectronicLockpick, ItemTypeClothing,
    // ItemTypeKey, ItemTypeKeyCard, ItemTypeMessage, ItemTypeCamera, ItemTypeShovel
    // use all-false / zero defaults.
}

// HasMeleeAction returns true when the item can be used at melee range.
func (t ItemType) HasMeleeAction() bool { return itemTraitsTable[t].hasMeleeAction }

// IsMeleeTool returns true when the melee action is a tool-use (applies stimuli
// via TriggerOnToolUsage) rather than a physical strike.
func (t ItemType) IsMeleeTool() bool { return itemTraitsTable[t].isMeleeTool }

// MeleeDecreaseUses returns true when activating the item in melee should
// consume a use. Physical-strike weapons (guns pistol-whipping, blades) return
// false so that ammo / unlimited-use items are not inadvertently decremented.
func (t ItemType) MeleeDecreaseUses() bool { return itemTraitsTable[t].meleeDecreaseUses }

// HasRangedAction returns true when the item can be used at range.
func (t ItemType) HasRangedAction() bool { return itemTraitsTable[t].hasRangedAction }

// IsThrowable returns true when the ranged action is a throw (including
// throw-remote). Used by the aim system to enable target lock-on.
func (t ItemType) IsThrowable() bool { return itemTraitsTable[t].isThrowable }

// CanSelfActivate returns true when the item can be placed / armed at the
// player's own tile (drop-to-arm, e.g. proximity mines).
func (t ItemType) CanSelfActivate() bool { return itemTraitsTable[t].canSelfActivate }

func (t ItemType) IsRemoteDetonated() bool { return itemTraitsTable[t].isRemoteDetonated }

func (t ItemType) IsSpreadShot() bool { return itemTraitsTable[t].isSpreadShot }

// SpreadShotDegrees is the angular spread (in degrees) for a shotgun blast.
const SpreadShotDegrees = 44

// ShotgunPelletCount is the number of pellets fired per shotgun shot.
const ShotgunPelletCount = uint8(5)

// CooldownSecs returns the per-shot cooldown duration in seconds for this
// item type. This is the delay before the item can be used again after firing
// or striking.
func (t ItemType) CooldownSecs() float64 {
    if c := itemTraitsTable[t].cooldownSecs; c > 0 {
        return c
    }
    return 0.07 // default minimum cooldown for melee, throwables, tools, etc.
}

// HasScope returns true when equipping the item activates a directional narrow
// FoV cone (scoped mode). True for sniper rifles and the flashlight.
func (t ItemType) HasScope() bool { return itemTraitsTable[t].scopeFoV > 0 }

// ScopeFoV returns the field-of-view cone angle in degrees used while in
// scoped mode. Returns 0 for items that have no scope.
func (t ItemType) ScopeFoV() float64 { return itemTraitsTable[t].scopeFoV }

// LockDifficulty controls how many lockpicks a door/safe consumes and how long
// the picking animation takes (Deus-Ex-style consumable picks).
type LockDifficulty int

const (
    LockDifficultyEasy   LockDifficulty = iota // 1 pick, 2 s
    LockDifficultyMedium                       // 2 picks, 3 s
    LockDifficultyHard                         // 4 picks, 4 s
)

// PickCount returns the number of lockpicks that must be in the inventory.
func (d LockDifficulty) PickCount() int {
    switch d {
    case LockDifficultyMedium:
        return 2
    case LockDifficultyHard:
        return 4
    }
    return 1
}

// PickTime returns the animation duration in seconds.
func (d LockDifficulty) PickTime() float64 {
    switch d {
    case LockDifficultyMedium:
        return 3.0
    case LockDifficultyHard:
        return 4.0
    }
    return 2.0
}

func (d LockDifficulty) ToString() string {
    switch d {
    case LockDifficultyMedium:
        return "medium"
    case LockDifficultyHard:
        return "hard"
    }
    return "easy"
}

func NewLockDifficultyFromString(s string) LockDifficulty {
    switch s {
    case "medium":
        return LockDifficultyMedium
    case "hard":
        return LockDifficultyHard
    }
    return LockDifficultyEasy
}

// LockType distinguishes mechanical (key + lockpick) from electronic (keycard + e-pick).
// Shared by Door and Safe.
type LockType bool

const (
    LockTypeMechanical LockType = false
    LockTypeElectronic LockType = true
)

type ItemEffectTrigger uint16

const (
    NoTrigger            ItemEffectTrigger = 0
    TriggerOnMeleeAttack ItemEffectTrigger = 1 << iota
    TriggerOnRangedShotHit
    TriggerOnSneakingWithItem
    TriggerOnItemDropped
    TriggerOnItemImpact
    TriggerAfterItemImpact
    TriggerOnFlightpath
    TriggerOnToolUsage
    TriggerOnTakenToNewCell
    TriggerOnRemoteControl
    TriggerOnActorContact // fires when any actor steps onto the tile holding this item
)

func NewItemEffectTriggerFromString(s string) ItemEffectTrigger {
    switch s {
    case "on_melee_attack":
        return TriggerOnMeleeAttack
    case "on_ranged_shot_hit":
        return TriggerOnRangedShotHit
    case "on_sneaking_with_item":
        return TriggerOnSneakingWithItem
    case "on_item_dropped":
        return TriggerOnItemDropped
    case "on_item_impact":
        return TriggerOnItemImpact
    case "after_item_impact":
        return TriggerAfterItemImpact
    case "on_flightpath":
        return TriggerOnFlightpath
    case "on_tool_usage":
        return TriggerOnToolUsage
    case "on_taken_to_new_cell":
        return TriggerOnTakenToNewCell
    case "on_remote_control":
        return TriggerOnRemoteControl
    case "on_actor_contact":
        return TriggerOnActorContact
    default:
        return NoTrigger
    }
}

type Item struct {
    Name        string
    DefinedIcon rune
    MapPos      geometry.Point
    HeldBy      *Actor
    IsBig       bool

    Type      ItemType
    KeyString string

    TriggerEffects  map[ItemEffectTrigger]stimuli.StimEffect
    ReactionEffects map[stimuli.StimulusType]StimReaction

    InsteadOfUse    func()
    InsteadOfPickup func(actor *Actor, item *Item)

    Uses            int
    ProjectileRange int
    OnCooldown      bool
    IsDestroyed     bool
    Buried          bool
    AudioCue        string
    NoiseRadius     int
    StartPosition   geometry.Point
    LootValue       int
}

func (i *Item) Icon() rune {
    return i.DefinedIcon
}

// IsSilenced returns true when this weapon makes no significant noise (noise_radius <= 0).
func (i *Item) IsSilenced() bool {
    return i.NoiseRadius <= 0
}

func (i *Item) Description() string {
    if (i.Type == ItemTypeKey || i.Type == ItemTypeKeyCard) && i.KeyString != "" {
        return fmt.Sprintf("%s (%s)", i.Name, i.KeyString)
    }
    return i.Name
}

func (i *Item) SetPos(point geometry.Point) {
    i.MapPos = point
}

type StimReaction struct {
    ForceThreshold   int
    EffectOnReaction stimuli.StimEffect
}

func (i *Item) Pos() geometry.Point {
    if i.HeldBy != nil {
        return i.HeldBy.Pos()
    }
    return i.MapPos
}

func (i *Item) Style(st common.Style) common.Style {
    itemStyle := common.Style{Foreground: CurrentTheme.ItemForeground, Background: st.Background}
    switch i.Type {
    case ItemTypeEmeticPoison:
        itemStyle = itemStyle.WithFg(CurrentTheme.EmeticPoisonForeground)
    case ItemTypeLethalPoison, ItemTypeLethalPoisonGrenade, ItemTypeLethalPoisonMine:
        itemStyle = itemStyle.WithFg(CurrentTheme.LethalPoisonForeground)
    case ItemTypeSleepPoison, ItemTypeSleepPoisonGrenade, ItemTypeSleepPoisonMine:
        itemStyle = itemStyle.WithFg(CurrentTheme.SleepPoisonForeground)
    }
    return itemStyle
}
func (i *Item) IsLegalForActor(actor *Actor) bool {
    if actor.Type == ActorTypeGuard || actor.IsTarget {
        return true
    }
    return !i.IsObviousWeapon()
}

func (i *Item) HasUsesLeft() bool {
    if i.Uses == -1 {
        return true
    }
    return i.Uses > 0
}

// HasMeleePiercingDamage reports whether this item deals piercing damage on a melee hit.
func (i *Item) HasMeleePiercingDamage() bool {
    effect, ok := i.TriggerEffects[TriggerOnMeleeAttack]
    if !ok {
        return false
    }
    for _, stim := range effect.Stimuli {
        if stim.Type() == stimuli.StimulusPiercingDamage {
            return true
        }
    }
    return false
}

const UnlimitedUses = -1

func (i *Item) DeepCopy() *Item {
    newItem := *i
    newItem.TriggerEffects = make(map[ItemEffectTrigger]stimuli.StimEffect, len(i.TriggerEffects))
    for k, v := range i.TriggerEffects {
        newItem.TriggerEffects[k] = v
    }
    newItem.ReactionEffects = make(map[stimuli.StimulusType]StimReaction, len(i.ReactionEffects))
    for k, v := range i.ReactionEffects {
        newItem.ReactionEffects[k] = v
    }
    newItem.InsteadOfUse = i.InsteadOfUse
    newItem.InsteadOfPickup = i.InsteadOfPickup
    newItem.HeldBy = i.HeldBy
    return &newItem
}

func (i *Item) WasMoved() bool {
    return i.StartPosition != i.MapPos
}

func (i *Item) DecreaseUsesLeft() {
    if i.Uses == UnlimitedUses {
        return
    }
    if i.Uses > 0 {
        i.Uses--
    }
}

// IsLockpickType returns true for mechanical or electronic lockpicks.
// These items stack in a single inventory slot using Uses as the item counter.
func (i *Item) IsLockpickType() bool {
    return i.Type == ItemTypeMechanicalLockpick || i.Type == ItemTypeElectronicLockpick
}

func (i *Item) IsWeapon() bool                { return i.IsRangedWeapon() || i.IsMeleeWeapon() }
func (i *Item) IsMeleeWeapon() bool           { return itemTraitsTable[i.Type].isMeleeWeapon }
func (i *Item) IsObviousWeapon() bool         { return i.IsRangedWeapon() || i.Type == ItemTypeKnife }
func (i *Item) IsRangedWeapon() bool          { return itemTraitsTable[i.Type].isRangedWeapon }
func (i *Item) IsAutomaticRangedWeapon() bool { return itemTraitsTable[i.Type].isAutomaticRangedWeapon }

// IsMine returns true for proximity-triggered explosive/poison mines.
func (i *Item) IsMine() bool {
    return i.Type == ItemTypeProximityMine || i.Type == ItemTypeLethalPoisonMine || i.Type == ItemTypeSleepPoisonMine
}

func (i *Item) SetKey(key string) {
    i.KeyString = key
}

func (i *Item) GetKey() string {
    return i.KeyString
}

func (i *Item) ToRecord() []rec_files.Field {
    fields := []rec_files.Field{
        {Name: "name", Value: i.Name},
        {Name: "icon", Value: string(i.DefinedIcon)},
        {Name: "type", Value: i.Type.ToString()},
        {Name: "projectile_range", Value: strconv.Itoa(i.ProjectileRange)},
    }
    if i.IsBig {
        fields = append(fields, rec_files.Field{Name: "is_big", Value: "true"})
    }
    if i.Uses != UnlimitedUses {
        fields = append(fields, rec_files.Field{Name: "uses", Value: strconv.Itoa(i.Uses)})
    }
    if i.NoiseRadius != 0 {
        fields = append(fields, rec_files.Field{Name: "noise_radius", Value: strconv.Itoa(i.NoiseRadius)})
    }
    if i.AudioCue != "" {
        fields = append(fields, rec_files.Field{Name: "audio_cue", Value: i.AudioCue})
    }
    if i.LootValue != 0 {
        fields = append(fields, rec_files.Field{Name: "loot_value", Value: strconv.Itoa(i.LootValue)})
    }
    if i.KeyString != "" {
        fields = append(fields, rec_files.Field{Name: "key_string", Value: i.KeyString})
    }
    return fields
}

func NewEmptyKey() *Item {
    return &Item{Name: "Key", DefinedIcon: GlyphKey, Type: ItemTypeKey, Uses: UnlimitedUses}
}

func NewEmptyKeyCard() *Item {
    return &Item{Name: "Key card", DefinedIcon: GlyphKeyCard, Type: ItemTypeKeyCard, Uses: UnlimitedUses}
}

func NewKeyCard(key string) *Item {
    emptyKeyCard := NewEmptyKeyCard()
    emptyKeyCard.SetKey(key)
    return emptyKeyCard
}

func NewKey(key string) *Item {
    emptyKey := NewEmptyKey()
    emptyKey.SetKey(key)
    return emptyKey
}
