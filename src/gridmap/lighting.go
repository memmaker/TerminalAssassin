package gridmap

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/geometry"
)

// AddDynamicLightSource adds a light source to the map. It will automatically call UpdateDynamicLights.
func (m *GridMap[ActorType, ItemType, ObjectType]) AddDynamicLightSource(pos geometry.Point, light *LightSource) {
	if m.IsDynamicLightSource(pos) {
		return
	}
	m.DynamicLights[pos] = light
}

// AddBakedLightSource adds a light source to the map. It will automatically call UpdateBakedLights and UpdateDynamicLights.
func (m *GridMap[ActorType, ItemType, ObjectType]) AddBakedLightSource(pos geometry.Point, light *LightSource) {
	if m.IsBakedLightSource(pos) {
		return
	}
	m.BakedLights[pos] = light
}
func (m *GridMap[ActorType, ItemType, ObjectType]) SetAmbientLight(color common.RGBAColor) {
	m.AmbientLight = color
	m.ApplyAmbientLight()
	m.UpdateBakedLights()
	m.UpdateDynamicLights()
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsDynamicLightSource(pos geometry.Point) bool {
	_, ok := m.DynamicLights[pos]
	return ok
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsBakedLightSource(pos geometry.Point) bool {
	_, ok := m.BakedLights[pos]
	return ok
}

// MoveLightSource moves a light source to a new position. It will automatically call UpdateDynamicLights.
func (m *GridMap[ActorType, ItemType, ObjectType]) MoveLightSource(lightSource *LightSource, to geometry.Point) {
	if m.IsDynamicLightSource(to) {
		return
	}
	delete(m.DynamicLights, lightSource.Pos)
	lightSource.Pos = to
	m.DynamicLights[to] = lightSource
	m.UpdateDynamicLights()
}

func (m *GridMap[ActorType, ItemType, ObjectType]) LightAt(p geometry.Point) common.Color {
	if dynamicLightAt, ok := m.dynamicallyLitCells[p]; ok {
		return dynamicLightAt
	}
	return m.Cells[p.X+p.Y*m.MapWidth].BakedLighting
}

// we use the value stored in cell.Lighting for lighting the tile later on..
func (m *GridMap[ActorType, ItemType, ObjectType]) UpdateDynamicLights() {
	for key, _ := range m.dynamicallyLitCells {
		delete(m.dynamicallyLitCells, key)
	}
	if len(m.DynamicLights) == 0 {
		return
	}
	setLightAt := func(point geometry.Point, light common.RGBAColor) {
		m.dynamicallyLitCells[point] = light
	}
	getBaseLight := func(point geometry.Point) common.RGBAColor {
		return m.Cells[point.X+point.Y*m.MapWidth].BakedLighting
	}
	//println("Updating dynamic lights: ", len(m.DynamicLights))
	m.updateLightMap(getBaseLight, m.DynamicLights, setLightAt)
	m.DynamicLightsChanged = false
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ApplyAmbientLight() {
	for i := range m.Cells {
		m.Cells[i].BakedLighting = m.AmbientLight
	}
}
func (m *GridMap[ActorType, ItemType, ObjectType]) UpdateBakedLights() {
	setLightAt := func(point geometry.Point, light common.RGBAColor) {
		m.Cells[point.X+point.Y*m.MapWidth].BakedLighting = light
	}
	getBaseLight := func(point geometry.Point) common.RGBAColor {
		return m.AmbientLight
	}
	//println("Updating baked lights: ", len(m.BakedLights))
	m.updateLightMap(getBaseLight, m.BakedLights, setLightAt)
}
func (m *GridMap[ActorType, ItemType, ObjectType]) updateLightMap(getBaseLight func(p geometry.Point) common.RGBAColor, lightSources map[geometry.Point]*LightSource, setLightAt func(p geometry.Point, light common.RGBAColor)) {
	lightAt := make(map[geometry.Point]common.RGBAColor)
	isTransparent := func(p geometry.Point) bool {
		return m.IsTransparent(p) && !m.IsActorAt(p)
	}
	for _, lightSource := range lightSources {
		for _, nodePos := range m.lightfov.SSCVisionMap(lightSource.Pos, lightSource.Radius, isTransparent, true) {

			//for _, node := range m.lightfov.LightMap(&MapLighter[VictimType, ItemType, ObjectType]{gridmap: m, sources: lightSources}, []geometry.Point{lightSource.Pos}) {
			pos := nodePos
			dist := geometry.Distance(lightSource.Pos, pos)
			//pos := node.P
			//dist := node.Cost
			if dist < 0 {
				dist = 0
			}
			if _, hasValue := lightAt[pos]; !hasValue {
				lightAt[pos] = getBaseLight(pos)
			}
			colorOfLight := lightAt[pos]

			intensityWithFalloff := common.Clamp((float64(lightSource.Radius)-dist)/dist, 0, lightSource.MaxIntensity)
			sourceLightColor := lightSource.Color.MultiplyWithScalar(intensityWithFalloff)
			lightAt[pos] = colorOfLight.AddRGB(sourceLightColor)
		}
	}
	for pos, light := range lightAt {
		setLightAt(pos, light)
	}
}

type MapLighter[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}] struct {
	gridmap *GridMap[ActorType, ItemType, ObjectType]
	sources map[geometry.Point]*LightSource
}

func (m *MapLighter[ActorType, ItemType, ObjectType]) Cost(src geometry.Point, from geometry.Point, to geometry.Point) float64 {
	if src == from {
		return 1
		return geometry.Distance(from, to)
	}
	currentMap := m.gridmap
	switch {
	case !currentMap.Cells[to.Y*currentMap.MapWidth+to.X].TileType.IsTransparent || !currentMap.Cells[from.Y*currentMap.MapWidth+from.X].TileType.IsTransparent:
		return 1000
	case currentMap.IsActorAt(from):
		return geometry.Distance(from, to) + 2
	}

	return geometry.Distance(from, to)
}

// needed for lighting
func (m *MapLighter[ActorType, ItemType, ObjectType]) MaxCost(src geometry.Point) float64 {
	if light, ok := m.sources[src]; ok {
		return float64(light.Radius) + 0.5
	}
	return 0
}
