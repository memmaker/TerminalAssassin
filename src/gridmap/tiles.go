package gridmap

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

type SpecialTileType uint64

func (t SpecialTileType) ToString() string {
	switch t {
	case SpecialTileDefaultFloor:
		return "defaultFloor"
	case SpecialTileToilet:
		return "toilet"
	case SpecialTilePlayerSpawn:
		return "playerSpawn"
	case SpecialTilePlayerExit:
		return "playerExit"
	case SpecialTileTreeLike:
		return "treeLike"
	case SpecialTileTypeFood:
		return "food"
	case SpecialTileTypePowerOutlet:
		return "powerOutlet"
	case SpecialTileLethal:
		return "lethal"
	default:
		return "none"
	}
}

func NewSpecialTileTypeFromString(text string) SpecialTileType {
	switch text {
	case "defaultFloor":
		return SpecialTileDefaultFloor
	case "toilet":
		return SpecialTileToilet
	case "playerSpawn":
		return SpecialTilePlayerSpawn
	case "playerExit":
		return SpecialTilePlayerExit
	case "treeLike":
		return SpecialTileTreeLike
	case "food":
		return SpecialTileTypeFood
	case "powerOutlet":
		return SpecialTileTypePowerOutlet
	case "lethal":
		return SpecialTileLethal
	default:
		return SpecialTileNone
	}
}

// These are markers, so we can identify special types of tiles programmatically.
const (
	SpecialTileNone SpecialTileType = iota
	SpecialTileDefaultFloor
	SpecialTileToilet
	SpecialTilePlayerSpawn
	SpecialTilePlayerExit
	SpecialTileTreeLike
	SpecialTileTypeFood
	SpecialTileTypePowerOutlet
	SpecialTileLethal
)

type Tile struct {
	DefinedIcon        rune
	DefinedDescription string
	DefinedStyle       common.Style
	IsWalkable         bool
	IsTransparent      bool
	Special            SpecialTileType
}

func (t Tile) Icon() rune {
	return t.DefinedIcon
}

func (t Tile) Description() string {
	return t.DefinedDescription
}

func (t Tile) Style() common.Style {
	return t.DefinedStyle
}

func (t Tile) WithBGColor(color common.Color) Tile {
	t.DefinedStyle = t.DefinedStyle.WithBg(color)
	return t
}

func (t Tile) WithFGColor(color common.Color) Tile {
	t.DefinedStyle = t.DefinedStyle.WithFg(color)
	return t
}

func (t Tile) EncodeAsString() string {
	return fmt.Sprintf("%c: %s", t.DefinedIcon, t.DefinedDescription)
}

func (t Tile) ToTextMap() []rec_files.Field {
	return []rec_files.Field{
		{Name: "icon", Value: string(t.DefinedIcon)},
		{Name: "description", Value: t.DefinedDescription},
		{Name: "styleFG", Value: t.DefinedStyle.Foreground.EncodeAsString()},
		{Name: "styleBG", Value: t.DefinedStyle.Background.EncodeAsString()},
		{Name: "walkable", Value: fmt.Sprintf("%t", t.IsWalkable)},
		{Name: "transparent", Value: fmt.Sprintf("%t", t.IsTransparent)},
		{Name: "special", Value: t.Special.ToString()},
	}
}

func (t Tile) IsLethal() bool {
	return t.Special == SpecialTileLethal
}

func (t Tile) ToString() string {
	if t.Special != SpecialTileNone {
		return fmt.Sprintf("%s (%s)", t.DefinedDescription, t.Special.ToString())
	}
	return t.DefinedDescription
}

func NewTileFromRecord(record map[string]string) *Tile {
	return &Tile{
		DefinedIcon:        []rune(record["icon"])[0],
		DefinedDescription: record["description"],
		DefinedStyle: common.Style{
			Foreground: common.NewColorFromString(record["styleFG"]),
			Background: common.NewColorFromString(record["styleBG"]),
		},
		IsWalkable:    record["walkable"] == "true",
		IsTransparent: record["transparent"] == "true",
		Special:       NewSpecialTileTypeFromString(record["special"]),
	}
}

type MapCell[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}] struct {
	TileType      Tile
	IsExplored    bool
	Stimuli       map[stimuli.StimulusType]stimuli.Stimulus
	BakedLighting common.RGBAColor
	Actor         *ActorType
	DownedActor   *ActorType
	Item          *ItemType
	Object        *ObjectType
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithStimulus(s stimuli.Stimulus) *MapCell[ActorType, ItemType, ObjectType] {
	c.Stimuli[s.Type()] = s
	return &c
}

func (c MapCell[ActorType, ItemType, ObjectType]) HasStim(stimulusType stimuli.StimulusType) bool {
	_, ok := c.Stimuli[stimulusType]
	return ok
}

func (c MapCell[ActorType, ItemType, ObjectType]) HasStims() bool {
	return len(c.Stimuli) > 0
}

func (c MapCell[ActorType, ItemType, ObjectType]) ForceOfStim(stimulusType stimuli.StimulusType) int {
	if stim, ok := c.Stimuli[stimulusType]; ok {
		return stim.Force()
	}
	return 0
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithItemHereRemoved(itemHere ItemType) MapCell[ActorType, ItemType, ObjectType] {
	if c.Item != nil && *c.Item == itemHere {
		c.Item = nil
	}
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithObjectHereRemoved(obj ObjectType) MapCell[ActorType, ItemType, ObjectType] {
	if c.Object != nil && *c.Object == obj {
		c.Object = nil
	}
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithItemRemoved() MapCell[ActorType, ItemType, ObjectType] {
	c.Item = nil
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithObjectRemoved() MapCell[ActorType, ItemType, ObjectType] {
	c.Object = nil
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithDownedActor(a ActorType) MapCell[ActorType, ItemType, ObjectType] {
	c.DownedActor = &a
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithActor(actor ActorType) MapCell[ActorType, ItemType, ObjectType] {
	c.Actor = &actor
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithObject(obj ObjectType) MapCell[ActorType, ItemType, ObjectType] {
	c.Object = &obj
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithActorHereRemoved(actorHere ActorType) MapCell[ActorType, ItemType, ObjectType] {
	if c.Actor != nil && *c.Actor == actorHere {
		c.Actor = nil
	}
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithItem(item ItemType) MapCell[ActorType, ItemType, ObjectType] {
	c.Item = &item
	return c
}
