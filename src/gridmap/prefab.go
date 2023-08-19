package gridmap

import (
	"github.com/huandu/go-clone"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Prefab[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}] struct {
	Cells                []MapCell[ActorType, ItemType, ObjectType]
	ActorPositions       map[geometry.Point]ActorType
	DownedActorPositions map[geometry.Point]ActorType
	ItemPositions        map[geometry.Point]ItemType
	ObjectPositions      map[geometry.Point]ObjectType

	DynamicLights map[geometry.Point]*LightSource
	BakedLights   map[geometry.Point]*LightSource
	bounds        geometry.Rect
}

func NewPrefab[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}](width, height int) *Prefab[ActorType, ItemType, ObjectType] {
	return &Prefab[ActorType, ItemType, ObjectType]{
		Cells:                make([]MapCell[ActorType, ItemType, ObjectType], width*height),
		ActorPositions:       make(map[geometry.Point]ActorType),
		DownedActorPositions: make(map[geometry.Point]ActorType),
		ItemPositions:        make(map[geometry.Point]ItemType),
		ObjectPositions:      make(map[geometry.Point]ObjectType),
		DynamicLights:        make(map[geometry.Point]*LightSource),
		BakedLights:          make(map[geometry.Point]*LightSource),
		bounds:               geometry.Rect{Min: geometry.Point{X: 0, Y: 0}, Max: geometry.Point{X: width, Y: height}},
	}
}

func NewPrefabFromMap[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}](loadedMap *GridMap[ActorType, ItemType, ObjectType], bbox geometry.Rect) *Prefab[ActorType, ItemType, ObjectType] {
	prefab := &Prefab[ActorType, ItemType, ObjectType]{
		Cells:                make([]MapCell[ActorType, ItemType, ObjectType], bbox.Size().X*bbox.Size().Y),
		ActorPositions:       make(map[geometry.Point]ActorType),
		DownedActorPositions: make(map[geometry.Point]ActorType),
		ItemPositions:        make(map[geometry.Point]ItemType),
		ObjectPositions:      make(map[geometry.Point]ObjectType),
		DynamicLights:        make(map[geometry.Point]*LightSource),
		BakedLights:          make(map[geometry.Point]*LightSource),
	}
	prefab.SetFromMapRegion(loadedMap, bbox)
	return prefab
}

func (p *Prefab[ActorType, ItemType, ObjectType]) RotateCW() {
	oldWidth, oldHeight := p.bounds.Size().X, p.bounds.Size().Y
	newWidth, _ := oldHeight, oldWidth
	newBounds := geometry.NewRect(p.bounds.Min.Y, p.bounds.Min.X, p.bounds.Max.Y, p.bounds.Max.X)
	newCells := make([]MapCell[ActorType, ItemType, ObjectType], oldWidth*oldHeight)
	newActorPositions := make(map[geometry.Point]ActorType)
	newDownedActorPositions := make(map[geometry.Point]ActorType)
	newItemPositions := make(map[geometry.Point]ItemType)
	newObjectPositions := make(map[geometry.Point]ObjectType)
	newDynamicLights := make(map[geometry.Point]*LightSource)
	newBakedLights := make(map[geometry.Point]*LightSource)
	for y := p.bounds.Min.Y; y < p.bounds.Max.Y; y++ {
		for x := p.bounds.Min.X; x < p.bounds.Max.X; x++ {
			oldPos := geometry.Point{X: x, Y: y}
			newPos := geometry.Point{X: -y, Y: x}
			xForIndex := x - p.bounds.Min.X
			yForIndex := y - p.bounds.Min.Y
			newXforIndex := (-y) - newBounds.Min.X
			newYforIndex := x - newBounds.Min.Y

			newCells[newXforIndex+newYforIndex*newWidth] = p.Cells[xForIndex+yForIndex*oldWidth]

			if actor, ok := p.ActorPositions[oldPos]; ok {
				newActorPositions[newPos] = actor
			}
			if actor, ok := p.DownedActorPositions[oldPos]; ok {
				newDownedActorPositions[newPos] = actor
			}
			if item, ok := p.ItemPositions[oldPos]; ok {
				newItemPositions[newPos] = item
			}
			if object, ok := p.ObjectPositions[oldPos]; ok {
				newObjectPositions[newPos] = object
			}
			if dLight, ok := p.DynamicLights[oldPos]; ok {
				newDynamicLights[newPos] = dLight
			}
			if bLight, ok := p.BakedLights[oldPos]; ok {
				newBakedLights[newPos] = bLight
			}
		}
	}
	p.Cells = newCells
	p.ActorPositions = newActorPositions
	p.DownedActorPositions = newDownedActorPositions
	p.ItemPositions = newItemPositions
	p.ObjectPositions = newObjectPositions
	p.DynamicLights = newDynamicLights
	p.BakedLights = newBakedLights
	p.bounds = newBounds
}
func (p *Prefab[ActorType, ItemType, ObjectType]) SetMapRegion(loadedMap *GridMap[ActorType, ItemType, ObjectType], placementPos geometry.Point) {
	width := p.bounds.Size().X
	for y := p.bounds.Min.Y; y < p.bounds.Max.Y; y++ {
		for x := p.bounds.Min.X; x < p.bounds.Max.X; x++ {
			mapPos := geometry.Point{X: placementPos.X + x, Y: placementPos.Y + y}
			prefabPos := geometry.Point{X: x, Y: y}
			xForIndex := x - p.bounds.Min.X
			yForIndex := y - p.bounds.Min.Y
			loadedMap.Cells[mapPos.X+mapPos.Y*loadedMap.MapWidth] = p.Cells[xForIndex+yForIndex*width]

			if actor, ok := p.ActorPositions[prefabPos]; ok {
				clonedActor := clone.Clone(actor).(ActorType)
				clonedActor.SetPos(mapPos)
				loadedMap.MoveActor(clonedActor, mapPos)
			}
			if downedActor, ok := p.DownedActorPositions[prefabPos]; ok {
				clonedActor := clone.Clone(downedActor).(ActorType)
				clonedActor.SetPos(mapPos)
				loadedMap.MoveDownedActor(clonedActor, mapPos)
			}
			if item, ok := p.ItemPositions[prefabPos]; ok {
				clonedItem := clone.Clone(item).(ItemType)
				clonedItem.SetPos(mapPos)
				loadedMap.MoveItem(clonedItem, mapPos)
			}
			if object, ok := p.ObjectPositions[prefabPos]; ok {
				clonedObject := clone.Clone(object).(ObjectType)
				clonedObject.SetPos(mapPos)
				loadedMap.MoveObject(clonedObject, mapPos)
			}
			if light, ok := p.DynamicLights[prefabPos]; ok {
				clonedLight := clone.Clone(light).(*LightSource)
				clonedLight.Pos = mapPos
				loadedMap.DynamicLights[mapPos] = clonedLight
			}
			if light, ok := p.BakedLights[prefabPos]; ok {
				clonedLight := clone.Clone(light).(*LightSource)
				clonedLight.Pos = mapPos
				loadedMap.BakedLights[mapPos] = clonedLight
			}
		}
	}
}
func (p *Prefab[ActorType, ItemType, ObjectType]) SetFromMapRegion(loadedMap *GridMap[ActorType, ItemType, ObjectType], bbox geometry.Rect) {
	p.bounds = bbox.Sub(bbox.Mid())
	width := p.bounds.Size().X
	for x := bbox.Min.X; x < bbox.Max.X; x++ {
		for y := bbox.Min.Y; y < bbox.Max.Y; y++ {
			mapPos := geometry.Point{X: x, Y: y}
			prefabX := x - bbox.Mid().X
			prefabY := y - bbox.Mid().Y
			prefabPos := geometry.Point{X: prefabX, Y: prefabY}
			xForIndex := x - bbox.Min.X
			yForIndex := y - bbox.Min.Y
			p.Cells[xForIndex+yForIndex*width] = loadedMap.Cells[x+y*loadedMap.MapWidth]

			if actor, ok := loadedMap.TryGetActorAt(mapPos); ok {
				clonedActor := clone.Clone(actor).(ActorType)
				clonedActor.SetPos(prefabPos)
				p.ActorPositions[prefabPos] = clonedActor
			}
			if downedActor, ok := loadedMap.TryGetDownedActorAt(mapPos); ok {
				clonedActor := clone.Clone(downedActor).(ActorType)
				clonedActor.SetPos(prefabPos)
				p.DownedActorPositions[prefabPos] = clonedActor
			}
			if item, ok := loadedMap.TryGetItemAt(mapPos); ok {
				clonedItem := clone.Clone(item).(ItemType)
				clonedItem.SetPos(prefabPos)
				p.ItemPositions[prefabPos] = clonedItem
			}
			if object, ok := loadedMap.TryGetObjectAt(mapPos); ok {
				clonedObject := clone.Clone(object).(ObjectType)
				clonedObject.SetPos(prefabPos)
				p.ObjectPositions[prefabPos] = clonedObject
			}
			if light, ok := loadedMap.DynamicLights[mapPos]; ok {
				clonedLight := clone.Clone(light).(*LightSource)
				clonedLight.Pos = prefabPos
				p.DynamicLights[prefabPos] = clonedLight
			}
			if light, ok := loadedMap.BakedLights[mapPos]; ok {
				clonedLight := clone.Clone(light).(*LightSource)
				clonedLight.Pos = prefabPos
				p.BakedLights[prefabPos] = clonedLight
			}
		}
	}
}

func (p *Prefab[ActorType, ItemType, ObjectType]) Draw(con console.CellInterface, screenPos geometry.Point) {
	for y := p.bounds.Min.Y; y < p.bounds.Max.Y; y++ {
		for x := p.bounds.Min.X; x < p.bounds.Max.X; x++ {
			xForIndex := x - p.bounds.Min.X
			yForIndex := y - p.bounds.Min.Y
			cell := p.Cells[xForIndex+yForIndex*p.bounds.Size().X]
			drawPos := geometry.Point{X: screenPos.X + x, Y: screenPos.Y + y}

			style := cell.TileType.Style()
			icon := cell.TileType.Icon()

			if actor, ok := p.ActorPositions[geometry.Point{X: x, Y: y}]; ok {
				icon = actor.Icon()
			}
			if downedActor, ok := p.DownedActorPositions[geometry.Point{X: x, Y: y}]; ok {
				icon = downedActor.Icon()
			}
			if item, ok := p.ItemPositions[geometry.Point{X: x, Y: y}]; ok {
				icon = item.Icon()
			}
			if object, ok := p.ObjectPositions[geometry.Point{X: x, Y: y}]; ok {
				icon = object.Icon()
			}
			if _, ok := p.DynamicLights[geometry.Point{X: x, Y: y}]; ok {
				icon = '*'
			}
			if _, ok := p.BakedLights[geometry.Point{X: x, Y: y}]; ok {
				icon = '*'
			}

			con.SetSquare(drawPos, common.Cell{
				Style: style,
				Rune:  icon,
			})
		}
	}
}
