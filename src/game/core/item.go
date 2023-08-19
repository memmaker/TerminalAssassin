package core

import (
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
	ItemTypeScrewdriver
	ItemTypeWrench
	ItemTypeTaser
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
	ItemTypeMessage
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

	Type         ItemType
	MeleeAttack  ItemActionType
	RangedAttack ItemActionType
	SelfUse      ItemActionType

	KeyString string

	TriggerEffects  map[ItemEffectTrigger]stimuli.StimEffect
	ReactionEffects map[stimuli.StimulusType]StimReaction

	InsteadOfUse    func()
	InsteadOfPickup func(actor *Actor, item *Item)

	Uses                    int
	ProjectileRange         int
	SpreadInDegrees         int
	ProjectileCount         uint8
	Scope                   ScopeInfo
	IsSilenced              bool
	OnCooldown              bool
	DelayBetweenShotsInSecs float64
	IsDestroyed             bool
	DefinedStyle            common.Style
	SilencedCue             string
	AudioCue                string
	NoiseRadius             int
	StartPosition           geometry.Point
	LootValue               int
}

func (i *Item) Icon() rune {
	return i.DefinedIcon
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
		itemStyle = itemStyle.WithFg(ColorFromCode(ColorPoisonEmetic))
	case ItemTypeLethalPoison:
		itemStyle = itemStyle.WithFg(ColorFromCode(ColorPoisonLethal))
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

type ScopeInfo struct {
	Range        int
	FoVinDegrees float64
}

type ItemActionType int

func (t ItemActionType) ToString() string {
	switch t {
	case ActionTypeMeleeAttack:
		return "melee_attack"
	case ActionTypeShot:
		return "shot"
	case ActionTypeSpreadShot:
		return "spread_shot"
	case ActionTypeThrow:
		return "throw"
	case ActionTypeThrowRemote:
		return "throw_remote"
	case ActionTypeTool:
		return "tool"
	case ActionTypeMeleeUse:
		return "melee_use"
	default:
		return ""
	}
}

func NewItemActionTypeFromString(s string) ItemActionType {
	switch s {
	case "melee_attack":
		return ActionTypeMeleeAttack
	case "shot":
		return ActionTypeShot
	case "spread_shot":
		return ActionTypeSpreadShot
	case "throw":
		return ActionTypeThrow
	case "throw_remote":
		return ActionTypeThrowRemote
	case "tool":
		return ActionTypeTool
	case "melee_use":
		return ActionTypeMeleeUse
	default:
		return NoAction
	}
}

const (
	NoAction ItemActionType = iota
	ActionTypeMeleeAttack
	ActionTypeShot
	ActionTypeSpreadShot
	ActionTypeThrow
	ActionTypeThrowRemote
	ActionTypeTool
	ActionTypeMeleeUse
)

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
	return []rec_files.Field{
		{Name: "name", Value: i.Name},
		{Name: "icon", Value: string(i.DefinedIcon)},
		{Name: "type", Value: i.Type.ToString()},
		{Name: "is_big", Value: strconv.FormatBool(i.IsBig)},
		{Name: "uses", Value: strconv.Itoa(i.Uses)},
		{Name: "melee_attack", Value: i.MeleeAttack.ToString()},
		{Name: "ranged_attack", Value: i.RangedAttack.ToString()},
		{Name: "self_use", Value: i.SelfUse.ToString()},
		{Name: "projectile_range", Value: strconv.Itoa(i.ProjectileRange)},
		{Name: "spread_in_degrees", Value: strconv.Itoa(i.SpreadInDegrees)},
		{Name: "projectile_count", Value: strconv.Itoa(int(i.ProjectileCount))},
		{Name: "delay_between_shots_in_secs", Value: strconv.FormatFloat(i.DelayBetweenShotsInSecs, 'f', 2, 64)},
		{Name: "noise_radius", Value: strconv.Itoa(i.NoiseRadius)},
		{Name: "audio_cue", Value: i.AudioCue},
		{Name: "is_silenced", Value: strconv.FormatBool(i.IsSilenced)},
		{Name: "silenced_cue", Value: i.SilencedCue},
		{Name: "key_string", Value: i.KeyString},
		{Name: "loot_value", Value: strconv.Itoa(i.LootValue)},
		{Name: "scope_range", Value: strconv.Itoa(i.Scope.Range)},
		{Name: "scope_fov", Value: strconv.FormatFloat(i.Scope.FoVinDegrees, 'f', 2, 64)},
		{Name: "style_fg", Value: i.DefinedStyle.Foreground.EncodeAsString()},
		{Name: "style_bg", Value: i.DefinedStyle.Background.EncodeAsString()},
	}
}

func NewEmptyKey() *Item {
	return &Item{Name: "Key", DefinedIcon: GlyphKey, Type: ItemTypeKey, DefinedStyle: common.DefaultStyle, Uses: UnlimitedUses}
}

func NewEmptyKeyCard() *Item {
	return &Item{Name: "Key card", DefinedIcon: GlyphKeyCard, Type: ItemTypeKeyCard, DefinedStyle: common.DefaultStyle, Uses: UnlimitedUses}
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
