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

func (t ItemType) ToString() string {
    switch t {
    case ItemTypeCommon:
        return "common"
    case ItemTypePistol:
        return "pistol"
    case ItemTypeShotgun:
        return "shotgun"
    case ItemTypeSniperRifle:
        return "sniper_rifle"
    case ItemTypeAssaultRifle:
        return "assault_rifle"
    case ItemTypeSubmachineGun:
        return "submachine_gun"
    case ItemTypeMeleeSharp:
        return "melee_sharp"
    case ItemTypeMeleeBlunt:
        return "melee_blunt"
    case ItemTypeTool:
        return "tool"
    case ItemTypeLethalPoison:
        return "lethal_poison"
    case ItemTypeEmeticPoison:
        return "emetic_poison"
    case ItemTypeScrewdriver:
        return "screwdriver"
    case ItemTypeWrench:
        return "wrench"
    case ItemTypeTaser:
        return "taser"
    case ItemTypeDartGun:
        return "dart_gun"
    case ItemTypeRemoteTaser:
        return "remote_taser"
    case ItemTypeCrowbar:
        return "crowbar"
    case ItemTypePianoWire:
        return "piano_wire"
    case ItemTypeMechanicalLockpick:
        return "mechanical_lockpick"
    case ItemTypeElectronicalLockpick:
        return "electronical_lockpick"
    case ItemTypeCleaner:
        return "cleaner"
    case ItemTypeClothing:
        return "clothing"
    case ItemTypeKey:
        return "key"
    case ItemTypeKeyCard:
        return "key_card"
    case ItemTypeKnife:
        return "knife"
    case ItemTypeLoot:
        return "loot"
    case ItemTypeExplosive:
        return "explosive"
    case ItemTypeRemoteExplosive:
        return "remote_explosive"
    case ItemTypeProximityMine:
        return "proximity_mine"
    case ItemTypeSleepPoison:
        return "sleep_poison"
    case ItemTypeLethalPoisonGrenade:
        return "lethal_poison_grenade"
    case ItemTypeSleepPoisonGrenade:
        return "sleep_poison_grenade"
    case ItemTypeLethalPoisonMine:
        return "lethal_poison_mine"
    case ItemTypeSleepPoisonMine:
        return "sleep_poison_mine"
    case ItemTypeCamera:
        return "camera"
    case ItemTypeShovel:
        return "shovel"
    case ItemTypeFlashlight:
        return "flashlight"
    default:
        return "unknown"
    }
}
func NewItemTypeFromString(text string) ItemType {
    switch text {
    case "common":
        return ItemTypeCommon
    case "pistol":
        return ItemTypePistol
    case "shotgun":
        return ItemTypeShotgun
    case "sniper_rifle":
        return ItemTypeSniperRifle
    case "assault_rifle":
        return ItemTypeAssaultRifle
    case "submachine_gun":
        return ItemTypeSubmachineGun
    case "melee_sharp":
        return ItemTypeMeleeSharp
    case "melee_blunt":
        return ItemTypeMeleeBlunt
    case "tool":
        return ItemTypeTool
    case "lethal_poison":
        return ItemTypeLethalPoison
    case "emetic_poison":
        return ItemTypeEmeticPoison
    case "screwdriver":
        return ItemTypeScrewdriver
    case "wrench":
        return ItemTypeWrench
    case "taser":
        return ItemTypeTaser
    case "dart_gun":
        return ItemTypeDartGun
    case "remote_taser":
        return ItemTypeRemoteTaser
    case "crowbar":
        return ItemTypeCrowbar
    case "piano_wire":
        return ItemTypePianoWire
    case "mechanical_lockpick":
        return ItemTypeMechanicalLockpick
    case "electronical_lockpick":
        return ItemTypeElectronicalLockpick
    case "cleaner":
        return ItemTypeCleaner
    case "clothing":
        return ItemTypeClothing
    case "key":
        return ItemTypeKey
    case "key_card":
        return ItemTypeKeyCard
    case "loot":
        return ItemTypeLoot
    case "knife":
        return ItemTypeKnife
    case "explosive":
        return ItemTypeExplosive
    case "remote_explosive":
        return ItemTypeRemoteExplosive
    case "proximity_mine":
        return ItemTypeProximityMine
    case "sleep_poison":
        return ItemTypeSleepPoison
    case "lethal_poison_grenade":
        return ItemTypeLethalPoisonGrenade
    case "sleep_poison_grenade":
        return ItemTypeSleepPoisonGrenade
    case "lethal_poison_mine":
        return ItemTypeLethalPoisonMine
    case "sleep_poison_mine":
        return ItemTypeSleepPoisonMine
    case "camera":
        return ItemTypeCamera
    case "shovel":
        return ItemTypeShovel
    case "flashlight":
        return ItemTypeFlashlight
    default:
        return ItemTypeCommon
    }
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
    ItemTypeElectronicalLockpick
    ItemTypeCleaner
    ItemTypeClothing
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
    ItemTypeMessage
    ItemTypeCamera
    ItemTypeShovel
    ItemTypeFlashlight
)

// HasMeleeAction returns true when the item can be used at melee range.
func (t ItemType) HasMeleeAction() bool {
    switch t {
    case ItemTypePistol, ItemTypeShotgun, ItemTypeSniperRifle, ItemTypeAssaultRifle,
        ItemTypeSubmachineGun, ItemTypeMeleeSharp, ItemTypeMeleeBlunt, ItemTypeTool,
        ItemTypeLethalPoison, ItemTypeEmeticPoison, ItemTypeSleepPoison,
        ItemTypeScrewdriver, ItemTypeWrench, ItemTypeTaser, ItemTypeCrowbar,
        ItemTypeKnife, ItemTypePianoWire, ItemTypeCleaner, ItemTypeFlashlight:
        return true
    }
    return false
}

// IsMeleeTool returns true when the melee action is a tool-use (applies stimuli
// via TriggerOnToolUsage) rather than a physical strike.
func (t ItemType) IsMeleeTool() bool {
    switch t {
    case ItemTypeLethalPoison, ItemTypeEmeticPoison, ItemTypeSleepPoison,
        ItemTypeTool, ItemTypeCleaner:
        return true
    }
    return false
}

// MeleeDecreaseUses returns true when activating the item in melee should
// consume a use. Physical-strike weapons (guns pistol-whipping, blades) return
// false so that ammo / unlimited-use items are not inadvertently decremented.
func (t ItemType) MeleeDecreaseUses() bool {
    switch t {
    case ItemTypeTaser, ItemTypeLethalPoison, ItemTypeEmeticPoison,
        ItemTypeSleepPoison, ItemTypeTool, ItemTypeCleaner:
        return true
    }
    return false
}

// HasRangedAction returns true when the item can be used at range.
func (t ItemType) HasRangedAction() bool {
    switch t {
    case ItemTypePistol, ItemTypeShotgun, ItemTypeSniperRifle, ItemTypeAssaultRifle,
        ItemTypeSubmachineGun, ItemTypeDartGun,
        ItemTypeKnife, ItemTypeMeleeSharp, ItemTypeMeleeBlunt, ItemTypeScrewdriver,
        ItemTypeWrench, ItemTypeCrowbar,
        ItemTypeExplosive, ItemTypeRemoteExplosive, ItemTypeProximityMine,
        ItemTypeLethalPoisonGrenade, ItemTypeSleepPoisonGrenade,
        ItemTypeLethalPoisonMine, ItemTypeSleepPoisonMine,
        ItemTypeRemoteTaser, ItemTypeLoot, ItemTypeCommon:
        return true
    }
    return false
}

// IsThrowable returns true when the ranged action is a throw (including
// throw-remote). Used by the aim system to enable target lock-on.
func (t ItemType) IsThrowable() bool {
    switch t {
    case ItemTypeKnife, ItemTypeMeleeSharp, ItemTypeMeleeBlunt, ItemTypeScrewdriver,
        ItemTypeWrench, ItemTypeCrowbar,
        ItemTypeExplosive, ItemTypeRemoteExplosive, ItemTypeProximityMine,
        ItemTypeLethalPoisonGrenade, ItemTypeSleepPoisonGrenade,
        ItemTypeLethalPoisonMine, ItemTypeSleepPoisonMine,
        ItemTypeRemoteTaser, ItemTypeLoot, ItemTypeCommon:
        return true
    }
    return false
}

// CanSelfActivate returns true when the item can be placed / armed at the
// player's own tile (drop-to-arm, e.g. proximity mines).
func (t ItemType) CanSelfActivate() bool {
    switch t {
    case ItemTypeProximityMine, ItemTypeLethalPoisonMine, ItemTypeSleepPoisonMine:
        return true
    }
    return false
}

func (t ItemType) IsRemoteDetonated() bool {
    return t == ItemTypeRemoteExplosive || t == ItemTypeRemoteTaser
}

func (t ItemType) IsSpreadShot() bool {
    return t == ItemTypeShotgun
}

// SpreadShotDegrees is the angular spread (in degrees) for a shotgun blast.
const SpreadShotDegrees = 44

// ShotgunPelletCount is the number of pellets fired per shotgun shot.
const ShotgunPelletCount = uint8(5)

// CooldownSecs returns the per-shot cooldown duration in seconds for this
// item type. This is the delay before the item can be used again after firing
// or striking.
func (t ItemType) CooldownSecs() float64 {
    switch t {
    case ItemTypePistol:
        return 0.30
    case ItemTypeSubmachineGun:
        return 0.10
    case ItemTypeAssaultRifle:
        return 0.20
    case ItemTypeShotgun:
        return 1.50
    case ItemTypeSniperRifle:
        return 2.00
    case ItemTypeDartGun:
        return 2.00
    case ItemTypeTaser:
        return 0.20
    }
    return 0.07 // default minimum cooldown for melee, throwables, tools, etc.
}

// HasScope returns true when equipping the item activates a directional narrow
// FoV cone (scoped mode). True for sniper rifles and the flashlight.
func (t ItemType) HasScope() bool {
    return t == ItemTypeSniperRifle || t == ItemTypeFlashlight
}

// ScopeFoV returns the field-of-view cone angle in degrees used while in
// scoped mode. Returns 0 for items that have no scope.
func (t ItemType) ScopeFoV() float64 {
    switch t {
    case ItemTypeSniperRifle:
        return 20.0
    case ItemTypeFlashlight:
        return 60.0
    }
    return 0
}

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
    DefinedStyle    common.Style
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
    itemStyle := i.DefinedStyle.WithBg(st.Background)
    switch i.Type {
    case ItemTypeEmeticPoison:
        itemStyle = itemStyle.WithFg(ColorFromCode(ColorEmetic))
    case ItemTypeLethalPoison, ItemTypeLethalPoisonGrenade, ItemTypeLethalPoisonMine:
        itemStyle = itemStyle.WithFg(ColorFromCode(ColorLethal))
    case ItemTypeSleepPoison, ItemTypeSleepPoisonGrenade, ItemTypeSleepPoisonMine:
        itemStyle = itemStyle.WithFg(ColorFromCode(ColorSleep))
    }
    return itemStyle
}
func (i *Item) IsLegalForActor(actor *Actor) bool {
    if actor.Type == ActorTypeGuard || actor.Type == ActorTypeEnforcer || actor.Type == ActorTypeTarget {
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
    newItem.DefinedStyle = i.DefinedStyle
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

func (i *Item) IsWeapon() bool {
    return i.IsRangedWeapon() || i.IsMeleeWeapon()

}
func (i *Item) IsMeleeWeapon() bool {
    return i.Type == ItemTypeMeleeSharp ||
        i.Type == ItemTypeMeleeBlunt ||
        i.Type == ItemTypeScrewdriver ||
        i.Type == ItemTypeWrench ||
        i.Type == ItemTypeCrowbar ||
        i.Type == ItemTypePianoWire ||
        i.Type == ItemTypeKnife
}

func (i *Item) IsObviousWeapon() bool {
    return i.IsRangedWeapon() || i.Type == ItemTypeKnife
}

func (i *Item) IsRangedWeapon() bool {
    return i.Type == ItemTypePistol ||
        i.Type == ItemTypeShotgun ||
        i.Type == ItemTypeSniperRifle ||
        i.IsAutomaticRangedWeapon()
}

func (i *Item) IsAutomaticRangedWeapon() bool {
    return i.Type == ItemTypeAssaultRifle ||
        i.Type == ItemTypeSubmachineGun
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
    return &Item{Name: "Key", DefinedIcon: GlyphKey, Type: ItemTypeKey, DefinedStyle: common.DefaultStyle.Reversed(), Uses: UnlimitedUses}
}

func NewEmptyKeyCard() *Item {
    return &Item{Name: "Key card", DefinedIcon: GlyphKeyCard, Type: ItemTypeKeyCard, DefinedStyle: common.DefaultStyle.Reversed(), Uses: UnlimitedUses}
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
