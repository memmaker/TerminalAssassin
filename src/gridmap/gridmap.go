package gridmap

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/stimuli"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/mapset"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

type MapMetaData struct {
	MissionTitle string
	HashAsHex    string
	FileName     string
}

type MapObject interface {
	Pos() geometry.Point
	Icon() rune
	SetPos(geometry.Point)
}

type FoVMode uint8

const (
	FoVModeNormal FoVMode = iota
	FoVModeScoped
)

type MapActor interface {
	MapObject
	FoV() *geometry.FOV
	FoVMode() FoVMode
	FoVSource() geometry.Point
	VisionRange() int
	NameOfClothing() string
	ShiftSchedulesBy(point geometry.Point, mapSize geometry.Point)
}
type MapObjectWithProperties[ActorType interface {
	comparable
	MapActor
}] interface {
	MapObject
	IsWalkable(person ActorType) bool
	IsTransparent() bool
	IsPassableForProjectile() bool
}
type LightSource struct {
	Pos          geometry.Point
	Radius       int
	Color        common.RGBAColor
	MaxIntensity float64
}

func (s LightSource) ToRecord() []rec_files.Field {
	return []rec_files.Field{
		{Name: "Pos", Value: s.Pos.String()},
		{Name: "Radius", Value: strconv.Itoa(s.Radius)},
		{Name: "Color", Value: s.Color.EncodeAsString()},
		{Name: "Max_Intensity", Value: strconv.FormatFloat(s.MaxIntensity, 'f', 2, 64)},
	}
}

func NewLightSourceFromRecord(record []rec_files.Field) *LightSource {
	var result LightSource
	for _, field := range record {
		switch field.Name {
		case "Pos":
			result.Pos, _ = geometry.NewPointFromString(field.Value)
		case "Radius":
			result.Radius, _ = strconv.Atoi(field.Value)
		case "Color":
			result.Color = common.NewColorFromString(field.Value).ToRGB()
		case "Max_Intensity":
			result.MaxIntensity, _ = strconv.ParseFloat(field.Value, 64)
		}
	}
	return &result
}

type ZoneType int

func (t ZoneType) ToString() string {
	switch t {
	case ZoneTypePublic:
		return "Public"
	case ZoneTypePrivate:
		return "Private"
	case ZoneTypeHighSecurity:
		return "High Security"
	case ZoneTypeDropOff:
		return "Drop Off"
	}
	return "Unknown"
}

func NewZoneTypeFromString(str string) ZoneType {
	switch str {
	case "Public":
		return ZoneTypePublic
	case "Private":
		return ZoneTypePrivate
	case "High Security":
		return ZoneTypeHighSecurity
	case "Drop Off":
		return ZoneTypeDropOff
	}
	return ZoneTypePublic
}

const (
	ZoneTypePublic ZoneType = iota
	ZoneTypePrivate
	ZoneTypeHighSecurity
	ZoneTypeDropOff
)

type ZoneInfo struct {
	Name            string
	Type            ZoneType
	AmbienceCue     string
	AllowedClothing mapset.Set[string]
}

const PublicZoneName = "Public Space"

func (i ZoneInfo) IsDropOff() bool {
	return i.Type == ZoneTypeDropOff
}

func (i ZoneInfo) IsHighSecurity() bool {
	return i.Type == ZoneTypeHighSecurity || i.Type == ZoneTypeDropOff
}

func (i ZoneInfo) IsPublic() bool {
	return i.Type == ZoneTypePublic
}

func (i ZoneInfo) IsPrivate() bool {
	return i.Type == ZoneTypePrivate
}

func (i ZoneInfo) ToRecord() []rec_files.Field {
	fields := []rec_files.Field{
		{Name: "Name", Value: i.Name},
		{Name: "Type", Value: i.Type.ToString()},
		{Name: "Ambience_Cue", Value: i.AmbienceCue},
	}
	clothingList := i.AllowedClothing.ToSlice()
	sort.Strings(clothingList)
	for _, clothing := range clothingList {
		fields = append(fields, rec_files.Field{Name: "Allowed_Clothing", Value: clothing})
	}
	return fields
}

func (i ZoneInfo) ToString() string {
	return fmt.Sprintf("%s (%s)", i.Name, i.Type.ToString())
}
func NewZoneFromRecord(record []rec_files.Field) *ZoneInfo {
	newZone := &ZoneInfo{
		AllowedClothing: mapset.NewSet[string](),
	}
	for _, field := range record {
		switch field.Name {
		case "Name":
			newZone.Name = strings.TrimSpace(field.Value)
		case "Type":
			newZone.Type = NewZoneTypeFromString(strings.TrimSpace(field.Value))
		case "Ambience_Cue":
			newZone.AmbienceCue = strings.TrimSpace(field.Value)
		case "Allowed_Clothing":
			newZone.AllowedClothing.Add(strings.TrimSpace(field.Value))
		}
	}
	return newZone
}
func NewZone(name string) *ZoneInfo {
	return &ZoneInfo{
		Name:            name,
		AllowedClothing: mapset.NewSet[string](),
	}
}
func NewPublicZone(name string) *ZoneInfo {
	return &ZoneInfo{
		Name:            name,
		AllowedClothing: mapset.NewSet[string](),
		Type:            ZoneTypePublic,
	}
}

type GlobalMapDataOnDisk struct {
	MissionTitle      string
	Width             int
	Height            int
	PlayerSpawn       geometry.Point
	AmbientLight      common.RGBAColor
	MaxLightIntensity float64
	MaxVisionRange    int
	TimeOfDay         time.Time
	AmbienceSoundCue  string
}

func (d GlobalMapDataOnDisk) ToString() string {
	return fmt.Sprintf("Mission Title: %s\nWidth: %d\nHeight: %d\nPlayer Spawn: %s\nAmbient Light: %v\nMax Light Intensity: %f\nMax Vision Range: %d\nTime of Day: %s\nAmbience Sound Cue: %s",
		d.MissionTitle, d.Width, d.Height, d.PlayerSpawn.String(), d.AmbientLight, d.MaxLightIntensity, d.MaxVisionRange, d.TimeOfDay.String(), d.AmbienceSoundCue)
}

func (d GlobalMapDataOnDisk) ToRecord() []rec_files.Field {
	return []rec_files.Field{
		{Name: "Mission_Title", Value: d.MissionTitle},
		{Name: "Width", Value: strconv.Itoa(d.Width)},
		{Name: "Height", Value: strconv.Itoa(d.Height)},
		{Name: "Player_Spawn", Value: d.PlayerSpawn.String()},
		{Name: "Ambient_Light", Value: d.AmbientLight.EncodeAsString()},
		{Name: "Max_Light_Intensity", Value: strconv.FormatFloat(d.MaxLightIntensity, 'f', 2, 64)},
		{Name: "Max_Vision_Range", Value: strconv.Itoa(d.MaxVisionRange)},
		{Name: "Time_of_Day", Value: d.TimeOfDay.Format(time.RFC3339)},
		{Name: "Ambience_Sound_Cue", Value: d.AmbienceSoundCue},
	}
}
func NewGlobalMapFromRecord(record []rec_files.Field) GlobalMapDataOnDisk {
	var result GlobalMapDataOnDisk
	for _, field := range record {
		switch field.Name {
		case "Mission_Title":
			result.MissionTitle = strings.TrimSpace(field.Value)
		case "Width":
			result.Width, _ = strconv.Atoi(field.Value)
		case "Height":
			result.Height, _ = strconv.Atoi(field.Value)
		case "Player_Spawn":
			result.PlayerSpawn, _ = geometry.NewPointFromString(field.Value)
		case "Ambient_Light":
			result.AmbientLight = common.NewColorFromString(field.Value).ToRGB()
		case "Max_Light_Intensity":
			result.MaxLightIntensity, _ = strconv.ParseFloat(field.Value, 64)
		case "Max_Vision_Range":
			result.MaxVisionRange, _ = strconv.Atoi(field.Value)
		case "Time_of_Day":
			result.TimeOfDay, _ = time.Parse(time.RFC3339, field.Value)
		case "Ambience_Sound_Cue":
			result.AmbienceSoundCue = strings.TrimSpace(field.Value)
		}
	}
	return result
}

type GridMap[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}] struct {
	Cells           []MapCell[ActorType, ItemType, ObjectType]
	AllActors       []ActorType
	AllDownedActors []ActorType
	removedActors   []ActorType
	AllItems        []ItemType
	AllObjects      []ObjectType

	MetaData    MapMetaData
	PlayerSpawn geometry.Point

	DynamicLights     map[geometry.Point]*LightSource
	BakedLights       map[geometry.Point]*LightSource
	MapWidth          int
	MapHeight         int
	lightfov          *geometry.FOV
	AmbientLight      common.RGBAColor
	MaxLightIntensity float64
	MaxVisionRange    int

	pathfinder  *geometry.PathRange
	ListOfZones []*ZoneInfo
	ZoneMap     []*ZoneInfo
	Player      ActorType

	maxLOSRange          geometry.Rect
	dynamicallyLitCells  map[geometry.Point]common.Color
	DynamicLightsChanged bool
	TimeOfDay            time.Time

	NamedLocations   map[string]geometry.Point
	AmbienceSoundCue string
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddZone(zone *ZoneInfo) {
	m.ListOfZones = append(m.ListOfZones, zone)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetTile(position geometry.Point, mapTile Tile) {
	if !m.Contains(position) {
		return
	}
	index := position.Y*m.MapWidth + position.X
	m.Cells[index].TileType = mapTile
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetZone(position geometry.Point, zone *ZoneInfo) {
	if !m.Contains(position) {
		return
	}
	m.ZoneMap[position.Y*m.MapWidth+position.X] = zone
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveItemAt(position geometry.Point) {
	m.RemoveItem(m.ItemAt(position))
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveObjectAt(position geometry.Point) {
	m.RemoveObject(m.ObjectAt(position))
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveObject(obj ObjectType) {
	m.Cells[obj.Pos().Y*m.MapWidth+obj.Pos().X] = m.Cells[obj.Pos().Y*m.MapWidth+obj.Pos().X].WithObjectHereRemoved(obj)
	for i := len(m.AllObjects) - 1; i >= 0; i-- {
		if m.AllObjects[i] == obj {
			m.AllObjects = append(m.AllObjects[:i], m.AllObjects[i+1:]...)
			return
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IterAll(f func(p geometry.Point, c MapCell[ActorType, ItemType, ObjectType])) {
	for y := 0; y < m.MapHeight; y++ {
		for x := 0; x < m.MapWidth; x++ {
			f(geometry.Point{X: x, Y: y}, m.Cells[y*m.MapWidth+x])
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IterWindow(window geometry.Rect, f func(p geometry.Point, c MapCell[ActorType, ItemType, ObjectType])) {
	for y := window.Min.Y; y < window.Max.Y; y++ {
		for x := window.Min.X; x < window.Max.X; x++ {
			mapPos := geometry.Point{X: x, Y: y}
			if !m.Contains(mapPos) {
				continue
			}
			f(mapPos, m.Cells[y*m.MapWidth+x])
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetPlayerSpawn(position geometry.Point) {
	m.PlayerSpawn = position
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SaveToDisk(path string) error {
	file, _ := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	encodeErr := enc.Encode(m)
	encodedMap := buf.Bytes()
	writeCount, writeErr := file.Write(encodedMap)
	if encodeErr == nil && writeErr == nil {
		println("Map saved to file: " + path + ", length: " + strconv.Itoa(writeCount))
		return nil
	} else {
		println("Error saving map to file: " + path)
		if encodeErr != nil {
			println(encodeErr.Error())
			return encodeErr
		} else if writeErr != nil {
			println(writeErr.Error())
			return writeErr
		}
	}
	return nil
}

func (m *GridMap[ActorType, ItemType, ObjectType]) CellAt(location geometry.Point) MapCell[ActorType, ItemType, ObjectType] {
	return m.Cells[m.MapWidth*location.Y+location.X]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ItemAt(location geometry.Point) ItemType {
	return *m.Cells[m.MapWidth*location.Y+location.X].Item
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsItemAt(location geometry.Point) bool {
	return m.Cells[m.MapWidth*location.Y+location.X].Item != nil
}
func (m *GridMap[ActorType, ItemType, ObjectType]) SetActorToDowned(a ActorType) {
	m.RemoveActor(a)
	if m.IsDownedActorAt(a.Pos()) && m.DownedActorAt(a.Pos()) != a {
		m.displaceDownedActor(a)
		return
	}
	m.Cells[a.Pos().Y*m.MapWidth+a.Pos().X] = m.Cells[a.Pos().Y*m.MapWidth+a.Pos().X].WithDownedActor(a)
}
func (m *GridMap[ActorType, ItemType, ObjectType]) SetActorToRemoved(person ActorType) {
	m.RemoveActor(person)
	m.RemoveDownedActor(person)
	m.removedActors = append(m.removedActors, person)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MoveItem(item ItemType, to geometry.Point) {
	m.Cells[item.Pos().Y*m.MapWidth+item.Pos().X] = m.Cells[item.Pos().Y*m.MapWidth+item.Pos().X].WithItemHereRemoved(item)
	item.SetPos(to)
	m.Cells[to.Y*m.MapWidth+to.X] = m.Cells[to.Y*m.MapWidth+to.X].WithItem(item)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetRandomFreeNeighbor(location geometry.Point) geometry.Point {
	freeNearbyPositions := m.GetFreeCellsForDistribution(location, 1, m.IsCurrentlyPassable)
	if len(freeNearbyPositions) == 0 {
		return location
	}
	return freeNearbyPositions[0]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetFreeCellsForDistribution(position geometry.Point, neededCellCount int, freePredicate func(p geometry.Point) bool) []geometry.Point {
	foundFreeCells := mapset.NewSet[geometry.Point]()
	currentPosition := position
	openList := mapset.NewSet[geometry.Point]()
	closedList := mapset.NewSet[geometry.Point]()
	closedList.Add(currentPosition)

	for _, neighbor := range m.GetFilteredNeighbors(currentPosition, m.IsTileWalkable) {
		openList.Add(neighbor)
	}
	for foundFreeCells.Cardinality() < neededCellCount {
		freeNeighbors := m.GetFilteredNeighbors(currentPosition, freePredicate)
		for _, neighbor := range freeNeighbors {
			foundFreeCells.Add(neighbor)
		}
		// pop from open list
		pop, _ := openList.Pop()
		currentPosition = pop
		for _, neighbor := range m.GetFilteredNeighbors(currentPosition, m.IsTileWalkable) {
			if !closedList.Contains(neighbor) {
				openList.Add(neighbor)
			}
		}
		closedList.Add(currentPosition)
	}

	freeCells := foundFreeCells.ToSlice()
	sort.Slice(freeCells, func(i, j int) bool {
		return geometry.DistanceSquared(freeCells[i], position) < geometry.DistanceSquared(freeCells[j], position)
	})
	return freeCells
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsNextToTileWithSpecial(pos geometry.Point, specialType SpecialTileType) bool {
	for _, neighbor := range m.GetAllCardinalNeighbors(pos) {
		if m.CellAt(neighbor).TileType.Special == specialType {
			return true
		}
	}
	return false
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MapHash() string {
	return m.MetaData.HashAsHex
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MapFileName() string {
	return m.MetaData.FileName
}
func (m *GridMap[ActorType, ItemType, ObjectType]) Neighbors(point geometry.Point) []geometry.Point {
	return m.GetFilteredCardinalNeighbors(point, func(p geometry.Point) bool {
		return m.Contains(p) && m.IsTileWalkable(p)
	})
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Cost(point geometry.Point, point2 geometry.Point) int {
	return geometry.DistanceManhattan(point, point2)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveItem(item ItemType) {
	m.Cells[item.Pos().Y*m.MapWidth+item.Pos().X] = m.Cells[item.Pos().Y*m.MapWidth+item.Pos().X].WithItemHereRemoved(item)
	for i := len(m.AllItems) - 1; i >= 0; i-- {
		if m.AllItems[i] == item {
			m.AllItems = append(m.AllItems[:i], m.AllItems[i+1:]...)
			return
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetAllCardinalNeighbors(pos geometry.Point) []geometry.Point {
	neighbors := geometry.Neighbors{}
	allCardinalNeighbors := neighbors.Cardinal(pos, func(p geometry.Point) bool {
		return m.Contains(p)
	})
	return allCardinalNeighbors
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsSpecialAt(pos geometry.Point, specialValue SpecialTileType) bool {
	special := m.CellAt(pos).TileType.Special
	return special == specialValue
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsStimulusOnTile(pos geometry.Point, stimType stimuli.StimulusType) bool {
	if !m.Contains(pos) {
		return false
	}
	_, ok := m.CellAt(pos).Stimuli[stimType]
	return ok
}

func (m *GridMap[ActorType, ItemType, ObjectType]) WavePropagationFrom(pos geometry.Point, size int, pressure int) map[int][]geometry.Point {

	soundAnimationMap := make(map[int][]geometry.Point)
	m.pathfinder.DijkstraMap(m, []geometry.Point{pos}, size)
	for _, v := range m.pathfinder.DijkstraIterNodes {
		cost := v.Cost
		point := v.P
		if soundAnimationMap[cost] == nil {
			soundAnimationMap[cost] = make([]geometry.Point, 0)
		}
		soundAnimationMap[cost] = append(soundAnimationMap[cost], point)
	}
	return soundAnimationMap
}

func (m *GridMap[ActorType, ItemType, ObjectType]) PropagateElectroStimFromWaterTileAt(atLocation geometry.Point, electroStim stimuli.Stimulus) {

	electroTiles := m.GetConnected(atLocation, func(p geometry.Point) bool {
		return m.IsStimulusOnTile(p, stimuli.StimulusWater)
	})
	if len(electroTiles) == 0 {
		electroTiles = append(electroTiles, atLocation)
	}
	for _, tile := range electroTiles {
		m.AddStimulusToTile(tile, electroStim)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetConnected(startLocation geometry.Point, traverse func(p geometry.Point) bool) []geometry.Point {
	results := make([]geometry.Point, 0)
	for _, node := range m.pathfinder.BreadthFirstMap(MapPather{neighborPredicate: traverse, allNeighbors: m.GetAllCardinalNeighbors}, []geometry.Point{startLocation}, 100) {
		results = append(results, node.P)
	}
	return results
}
func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveStimulusFromTile(p geometry.Point, s stimuli.StimulusType) {
	if !m.Contains(p) {
		return
	}
	delete(m.Cells[m.MapWidth*p.Y+p.X].Stimuli, s)
	m.onStimRemoved(p, s)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveAllStimuliFromTile(pos geometry.Point) {
	for stimType := range m.CellAt(pos).Stimuli {
		m.RemoveStimulusFromTile(pos, stimType)
	}
}

// update for entities:
// call update for every updatable entity (genMap, AllItems, AllObjects, tiles)
// default: just return
// entities have an internal schedule, waiting for ticks to happen

func (m *GridMap[ActorType, ItemType, ObjectType]) AddStimulusToTile(p geometry.Point, s stimuli.Stimulus) {
	if !m.Contains(p) {
		return
	}
	cell := m.Cells[m.MapWidth*p.Y+p.X]
	if cell.Stimuli == nil {
		m.Cells[m.MapWidth*p.Y+p.X].Stimuli = make(map[stimuli.StimulusType]stimuli.Stimulus)
	}
	if cell.Stimuli[s.Type()] == nil {
		m.Cells[m.MapWidth*p.Y+p.X].Stimuli[s.Type()] = s
		return
	}
	currentStim := cell.Stimuli[s.Type()]
	m.Cells[m.MapWidth*p.Y+p.X].Stimuli[s.Type()] = currentStim.WithForce(currentStim.Force() + s.Force())
	m.onStimAdded(p, s)
}
func NewEmptyMap[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}](width, height, maxVisionRange int) *GridMap[ActorType, ItemType, ObjectType] {
	pathRange := geometry.NewPathRange(geometry.NewRect(0, 0, width, height))
	publicSpaceZone := NewPublicZone(PublicZoneName)
	m := &GridMap[ActorType, ItemType, ObjectType]{
		Cells:               make([]MapCell[ActorType, ItemType, ObjectType], width*height),
		AllActors:           make([]ActorType, 0),
		AllDownedActors:     make([]ActorType, 0),
		AllItems:            make([]ItemType, 0),
		AllObjects:          make([]ObjectType, 0),
		DynamicLights:       map[geometry.Point]*LightSource{},
		BakedLights:         map[geometry.Point]*LightSource{},
		dynamicallyLitCells: map[geometry.Point]common.Color{},
		NamedLocations:      map[string]geometry.Point{},
		lightfov:            geometry.NewFOV(geometry.NewRect(0, 0, width, height)),
		ListOfZones:         []*ZoneInfo{publicSpaceZone},
		ZoneMap:             NewZoneMap(publicSpaceZone, width, height),
		MapWidth:            width,
		MapHeight:           height,
		MaxLightIntensity:   3,
		AmbientLight:        DefaultAmbientLight,
		TimeOfDay:           time.Now(),
		pathfinder:          pathRange,
		maxLOSRange:         geometry.NewRect(-maxVisionRange, -maxVisionRange, maxVisionRange+1, maxVisionRange+1),
		MaxVisionRange:      maxVisionRange,
	}
	m.Fill(MapCell[ActorType, ItemType, ObjectType]{
		TileType: Tile{
			DefinedIcon:        ' ',
			DefinedDescription: "empty space",
			DefinedStyle:       common.Style{Foreground: common.White, Background: common.Blue},
			IsWalkable:         true,
			IsTransparent:      true,
			Special:            SpecialTileNone,
		},
		IsExplored:    false,
		Stimuli:       nil,
		BakedLighting: common.RGBAColor{},
	})
	m.ApplyAmbientLight()
	return m
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetCell(p geometry.Point) MapCell[ActorType, ItemType, ObjectType] {
	return m.Cells[p.X+p.Y*m.MapWidth]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetCell(p geometry.Point, cell MapCell[ActorType, ItemType, ObjectType]) {
	m.Cells[p.X+p.Y*m.MapWidth] = cell
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetActor(p geometry.Point) ActorType {
	return *m.Cells[p.X+p.Y*m.MapWidth].Actor
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveActor(actor ActorType) {
	m.Cells[actor.Pos().X+actor.Pos().Y*m.MapWidth] = m.Cells[actor.Pos().X+actor.Pos().Y*m.MapWidth].WithActorHereRemoved(actor)
	for i := len(m.AllActors) - 1; i >= 0; i-- {
		if m.AllActors[i] == actor {
			m.AllActors = append(m.AllActors[:i], m.AllActors[i+1:]...)
			break
		}
	}
}

// MoveActor Should only be called my the model, so we can ensure that a HUD IsFinished will follow
func (m *GridMap[ActorType, ItemType, ObjectType]) MoveActor(actor ActorType, newPos geometry.Point) {
	m.Cells[actor.Pos().X+actor.Pos().Y*m.MapWidth] = m.Cells[actor.Pos().X+actor.Pos().Y*m.MapWidth].WithActorHereRemoved(actor)
	actor.SetPos(newPos)
	m.Cells[newPos.X+newPos.Y*m.MapWidth] = m.Cells[newPos.X+newPos.Y*m.MapWidth].WithActor(actor)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MoveObject(obj ObjectType, newPos geometry.Point) {
	m.Cells[obj.Pos().X+obj.Pos().Y*m.MapWidth] = m.Cells[obj.Pos().X+obj.Pos().Y*m.MapWidth].WithObjectHereRemoved(obj)
	obj.SetPos(newPos)
	m.Cells[newPos.X+newPos.Y*m.MapWidth] = m.Cells[newPos.X+newPos.Y*m.MapWidth].WithObject(obj)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Fill(mapCell MapCell[ActorType, ItemType, ObjectType]) {
	for i := range m.Cells {
		m.Cells[i] = mapCell
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsTransparent(p geometry.Point) bool {
	if !m.Contains(p) {
		return false
	}

	if objectAt, ok := m.TryGetObjectAt(p); ok && !objectAt.IsTransparent() {
		return false
	}

	return m.GetCell(p).TileType.IsTransparent
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsTileWalkable(point geometry.Point) bool {
	if !m.Contains(point) {
		return false
	}
	return m.GetCell(point).TileType.IsWalkable
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetMaxLightIntensity(f float64) {
	m.MaxLightIntensity = f
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Contains(dest geometry.Point) bool {
	return dest.X >= 0 && dest.X < m.MapWidth && dest.Y >= 0 && dest.Y < m.MapHeight
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsActorAt(location geometry.Point) bool {
	return m.Cells[location.X+location.Y*m.MapWidth].Actor != nil
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ActorAt(location geometry.Point) ActorType {
	return *m.Cells[location.X+location.Y*m.MapWidth].Actor
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsDownedActorAt(location geometry.Point) bool {
	return m.Cells[location.X+location.Y*m.MapWidth].DownedActor != nil
}

func (m *GridMap[ActorType, ItemType, ObjectType]) DownedActorAt(location geometry.Point) ActorType {
	return *m.Cells[location.X+location.Y*m.MapWidth].DownedActor
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsObjectAt(location geometry.Point) bool {
	return m.Cells[location.X+location.Y*m.MapWidth].Object != nil
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ObjectAt(location geometry.Point) ObjectType {
	return *m.Cells[location.X+location.Y*m.MapWidth].Object
}
func (m *GridMap[ActorType, ItemType, ObjectType]) ZoneAt(p geometry.Point) *ZoneInfo {
	return m.ZoneMap[m.MapWidth*p.Y+p.X]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetFilteredCardinalNeighbors(pos geometry.Point, filter func(geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	filtered := neighbors.Cardinal(pos, filter)
	return filtered
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Actors() []ActorType {
	return m.AllActors
}

func (m *GridMap[ActorType, ItemType, ObjectType]) DownedActors() []ActorType {
	return m.AllDownedActors
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Items() []ItemType {
	return m.AllItems
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Objects() []ObjectType {
	return m.AllObjects
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetFilteredNeighbors(pos geometry.Point, filter func(geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	filtered := neighbors.All(pos, filter)
	return filtered
}

func (m *GridMap[ActorType, ItemType, ObjectType]) displaceDownedActor(a ActorType) {
	free := m.GetFreeCellsForDistribution(a.Pos(), 1, func(p geometry.Point) bool {
		return !m.IsDownedActorAt(p) && m.IsWalkable(p)
	})
	freePos := free[0]
	m.MoveDownedActor(a, freePos)
}
func (m *GridMap[ActorType, ItemType, ObjectType]) GetJPSPath(start geometry.Point, end geometry.Point, isWalkable func(geometry.Point) bool, buffer []geometry.Point) []geometry.Point {
	if !isWalkable(end) {
		end = m.getNearestFreeNeighbor(start, end, isWalkable)
	}
	//println(fmt.Sprintf("JPS from %v to %v", start, end))
	buffer = m.pathfinder.JPSPath(buffer, start, end, isWalkable, false)
	return buffer
}
func (m *GridMap[ActorType, ItemType, ObjectType]) getNearestFreeNeighbor(origin, pos geometry.Point, isFree func(geometry.Point) bool) geometry.Point {
	dist := math.MaxInt32
	nearest := pos
	for _, neighbor := range m.NeighborsCardinal(pos, isFree) {
		d := geometry.DistanceManhattan(origin, neighbor)
		if d < dist {
			dist = d
			nearest = neighbor
		}
	}
	return nearest
}

func (m *GridMap[ActorType, ItemType, ObjectType]) getCurrentlyPassableNeighbors(pos geometry.Point) []geometry.Point {
	neighbors := geometry.Neighbors{}
	freeNeighbors := neighbors.All(pos, func(p geometry.Point) bool {
		return m.Contains(p) && m.IsCurrentlyPassable(p)
	})
	return freeNeighbors
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsCurrentlyPassable(p geometry.Point) bool {
	if !m.Contains(p) {
		return false
	}
	return m.IsWalkable(p) && (!m.IsActorAt(p)) //&& !knownAsBlocked
}
func (m *GridMap[ActorType, ItemType, ObjectType]) CurrentlyPassableAndSafeForActor(person ActorType) func(p geometry.Point) bool {
	return func(p geometry.Point) bool {
		if !m.Contains(p) ||
			(m.IsActorAt(p) && m.ActorAt(p) != person) {
			return false
		}
		return m.IsWalkableFor(p, person) && !m.IsObviousHazardAt(p)
	}
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsWalkable(p geometry.Point) bool {
	if !m.Contains(p) {
		return false
	}
	var noActor ActorType
	if m.IsObjectAt(p) && (!m.ObjectAt(p).IsWalkable(noActor)) {
		return false
	}
	cellAt := m.GetCell(p)
	return cellAt.TileType.IsWalkable
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsObviousHazardAt(p geometry.Point) bool {
	return m.IsStimulusOnTile(p, stimuli.StimulusFire) || m.IsLethalTileAt(p)
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsWalkableFor(p geometry.Point, person ActorType) bool {
	if !m.Contains(p) {
		return false
	}

	if m.IsObjectAt(p) && (!m.ObjectAt(p).IsWalkable(person)) {
		return false
	}

	cellAt := m.GetCell(p)
	return cellAt.TileType.IsWalkable

}

func (m *GridMap[ActorType, ItemType, ObjectType]) UpdateFieldOfView(person ActorType) {
	visionRange := person.VisionRange()
	visionRangeSquared := visionRange * visionRange

	var fovRange = geometry.NewRect(-visionRange, -visionRange, visionRange+1, visionRange+1)
	person.FoV().SetRange(fovRange.Add(person.FoVSource()).Intersect(geometry.NewRect(0, 0, m.MapWidth, m.MapHeight)))

	visionMap := person.FoV().SSCVisionMap(person.FoVSource(), visionRange, func(p geometry.Point) bool {
		return m.IsTransparent(p) && geometry.DistanceSquared(p, person.FoVSource()) <= visionRangeSquared
	}, false)
	if person != m.Player {
		return
	}
	for _, p := range visionMap {
		if !m.IsExplored(p) {
			m.SetExplored(p)
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsTrespassing(person ActorType) bool {
	ourPos := person.Pos()
	zoneAt := m.ZoneAt(ourPos)
	if zoneAt == nil || zoneAt.IsPublic() {
		return false
	}
	return !zoneAt.AllowedClothing.Contains(person.NameOfClothing())
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsInHostileZone(person ActorType) bool {
	ourPos := person.Pos()
	zoneAt := m.ZoneAt(ourPos)
	if zoneAt == nil || zoneAt.IsPublic() {
		return false
	}
	return !zoneAt.AllowedClothing.Contains(person.NameOfClothing()) && zoneAt.IsHighSecurity()
}

func (m *GridMap[ActorType, ItemType, ObjectType]) CurrentlyPassableForActor(person ActorType) func(p geometry.Point) bool {
	return func(p geometry.Point) bool {
		if !m.Contains(p) ||
			(m.IsActorAt(p) && m.ActorAt(p) != person) {
			return false
		}
		return m.IsWalkableFor(p, person)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsExplored(pos geometry.Point) bool {
	return m.GetCell(pos).IsExplored
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetExplored(pos geometry.Point) {
	m.Cells[pos.X+pos.Y*m.MapWidth].IsExplored = true
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetAllSpecialTilePositions(tile SpecialTileType) []geometry.Point {
	result := make([]geometry.Point, 0)
	for index, c := range m.Cells {
		if c.TileType.Special == tile {
			x := index % m.MapWidth
			y := index / m.MapWidth
			result = append(result, geometry.Point{X: x, Y: y})
		}
	}
	return result
}
func (m *GridMap[ActorType, ItemType, ObjectType]) GetNeighborWithSpecial(pos geometry.Point, specialType SpecialTileType) geometry.Point {
	neighbors := m.GetAllCardinalNeighbors(pos)
	for _, n := range neighbors {
		if m.GetCell(n).TileType.Special == specialType {
			return n
		}
	}
	return pos
}
func (m *GridMap[ActorType, ItemType, ObjectType]) GetNearestSpecialTile(pos geometry.Point, tile SpecialTileType) geometry.Point {
	result := pos
	allPositions := m.GetAllSpecialTilePositions(tile)

	m.pathfinder.DijkstraMap(m.currentlyPassablePather(), []geometry.Point{pos}, 100)

	sort.Slice(allPositions, func(i, j int) bool {
		return m.pathfinder.DijkstraMapAt(allPositions[i]) < m.pathfinder.DijkstraMapAt(allPositions[j])
	})
	if len(allPositions) > 0 {
		result = allPositions[0]
	}
	return result
}

func (m *GridMap[ActorType, ItemType, ObjectType]) currentlyPassablePather() MapPather {
	return MapPather{
		allNeighbors:      m.getCurrentlyPassableNeighbors,
		neighborPredicate: func(pos geometry.Point) bool { return true },
		pathCostFunc:      func(from geometry.Point, to geometry.Point) int { return 1 },
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SwapDownedPositions(downedActorOne ActorType, downedActorTwo ActorType) {
	posTwo := downedActorTwo.Pos()
	posOne := downedActorOne.Pos()
	downedActorOne.SetPos(posTwo)
	downedActorTwo.SetPos(posOne)
	m.Cells[posOne.X+posOne.Y*m.MapWidth] = m.Cells[posOne.X+posOne.Y*m.MapWidth].WithDownedActor(downedActorTwo)
	m.Cells[posTwo.X+posTwo.Y*m.MapWidth] = m.Cells[posTwo.X+posTwo.Y*m.MapWidth].WithDownedActor(downedActorOne)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SwapPositions(actorOne ActorType, actorTwo ActorType) {
	posTwo := actorTwo.Pos()
	posOne := actorOne.Pos()
	actorOne.SetPos(posTwo)
	actorTwo.SetPos(posOne)
	m.Cells[posOne.X+posOne.Y*m.MapWidth] = m.Cells[posOne.X+posOne.Y*m.MapWidth].WithActor(actorTwo)
	m.Cells[posTwo.X+posTwo.Y*m.MapWidth] = m.Cells[posTwo.X+posTwo.Y*m.MapWidth].WithActor(actorOne)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetNeighborWithStim(pos geometry.Point, stimType stimuli.StimulusType) geometry.Point {
	neighbors := m.GetAllCardinalNeighbors(pos)
	for _, n := range neighbors {
		if m.GetCell(n).HasStim(stimType) {
			return n
		}
	}
	return pos
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetStimAt(location geometry.Point, stimulusType stimuli.StimulusType) stimuli.Stimulus {
	return m.GetCell(location).Stimuli[stimulusType]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ForceOfStimulusOnTile(location geometry.Point, stimulusType stimuli.StimulusType) int {
	if val, ok := m.GetCell(location).Stimuli[stimulusType]; ok {
		return val.Force()
	}
	return 0
}

var DefaultAmbientLight = common.RGBAColor{R: 0.8, G: 0.8, B: 0.8, A: 1.0}

func NewZoneMap(zone *ZoneInfo, width int, height int) []*ZoneInfo {
	zoneMap := make([]*ZoneInfo, width*height)
	for i := 0; i < width*height; i++ {
		zoneMap[i] = zone
	}
	return zoneMap
}

type MapPather struct {
	neighborPredicate func(pos geometry.Point) bool
	allNeighbors      func(pos geometry.Point) []geometry.Point
	pathCostFunc      func(from geometry.Point, to geometry.Point) int
}

func (m MapPather) Neighbors(point geometry.Point) []geometry.Point {
	neighbors := make([]geometry.Point, 0)
	for _, p := range m.allNeighbors(point) {
		if m.neighborPredicate(p) {
			neighbors = append(neighbors, p)
		}
	}
	return neighbors
}
func (m MapPather) Cost(from geometry.Point, to geometry.Point) int {
	return m.pathCostFunc(from, to)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsTileWithSpecialAt(pos geometry.Point, special SpecialTileType) bool {
	return m.GetCell(pos).TileType.Special == special
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MoveDownedActor(actor ActorType, newPos geometry.Point) {
	m.Cells[actor.Pos().Y*m.MapWidth+actor.Pos().X].DownedActor = nil
	actor.SetPos(newPos)
	m.Cells[newPos.Y*m.MapWidth+newPos.X] = m.Cells[newPos.Y*m.MapWidth+newPos.X].WithDownedActor(actor)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveDownedActor(actor ActorType) bool {

	for i := len(m.AllDownedActors) - 1; i >= 0; i-- {
		if m.AllDownedActors[i] == actor {
			m.AllDownedActors = append(m.AllDownedActors[:i], m.AllDownedActors[i+1:]...)
			return true
		}
	}
	return false
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Apply(f func(cell MapCell[ActorType, ItemType, ObjectType]) MapCell[ActorType, ItemType, ObjectType]) {
	for i, cell := range m.Cells {
		m.Cells[i] = f(cell)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IterAllLights(f func(p geometry.Point, l *LightSource)) {
	for p, l := range m.BakedLights {
		f(p, l)
	}
	for p, l := range m.DynamicLights {
		f(p, l)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) onStimAdded(p geometry.Point, s stimuli.Stimulus) {
	if s.Type() == stimuli.StimulusFire {
		m.AddDynamicLightSource(p, &LightSource{Pos: p, Color: common.RGBAColor{R: 2.0, G: 0.4, B: 0.0, A: 1.0}, Radius: 3, MaxIntensity: 4})
		m.DynamicLightsChanged = true
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) onStimRemoved(p geometry.Point, s stimuli.StimulusType) {
	if s == stimuli.StimulusFire {
		m.RemoveDynamicLightAt(p)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveDynamicLightAt(p geometry.Point) {
	delete(m.DynamicLights, p)
	m.UpdateDynamicLights()
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetNearestDropOffPosition(pos geometry.Point) geometry.Point {
	nearestLocation := geometry.Point{X: 1, Y: 1}
	shortestDistance := math.MaxInt
	for index, c := range m.ZoneMap {
		xPos := index % m.MapWidth
		yPos := index / m.MapWidth
		curPos := geometry.Point{X: xPos, Y: yPos}
		curDist := geometry.DistanceManhattan(curPos, pos)
		isItemHere := m.IsItemAt(curPos)
		isValidZone := c.IsDropOff()
		if !isItemHere && isValidZone && curDist < shortestDistance {
			nearestLocation = curPos
			shortestDistance = curDist
		}
	}
	return nearestLocation
}

func (m *GridMap[ActorType, ItemType, ObjectType]) FindNearestItem(pos geometry.Point, predicate func(item ItemType) bool) ItemType {
	var nearestItem ItemType
	nearestDistance := math.MaxInt
	for _, item := range m.AllItems {
		if predicate(item) {
			curDist := geometry.DistanceManhattan(item.Pos(), pos)
			if curDist < nearestDistance {
				nearestItem = item
				nearestDistance = curDist
			}
		}
	}
	return nearestItem
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsNamedLocationAt(positionInWorld geometry.Point) bool {
	for _, loc := range m.NamedLocations {
		if loc == positionInWorld {
			return true
		}
	}
	return false
}
func (m *GridMap[ActorType, ItemType, ObjectType]) ZoneNames() []string {
	result := make([]string, 0)
	for _, zone := range m.ListOfZones {
		result = append(result, zone.Name)
	}
	return result
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Resize(width int, height int, emptyTile Tile) {
	oldWidth := m.MapWidth
	oldHeight := m.MapHeight

	newCells := make([]MapCell[ActorType, ItemType, ObjectType], width*height)
	newZoneMap := make([]*ZoneInfo, width*height)
	for i := 0; i < width*height; i++ {
		newZoneMap[i] = m.ListOfZones[0]
	}
	for i := 0; i < width*height; i++ {
		newCells[i] = MapCell[ActorType, ItemType, ObjectType]{
			TileType:      emptyTile,
			IsExplored:    true,
			Stimuli:       nil,
			BakedLighting: common.RGBAColor{},
		}
	}

	// copy over the old cells into the center of the new map
	for y := 0; y < oldHeight; y++ {
		if y >= height {
			break
		}
		for x := 0; x < oldWidth; x++ {
			if x >= width {
				break
			}
			destIndex := y*width + x
			srcIndex := y*oldWidth + x
			newCells[destIndex] = m.Cells[srcIndex]
			newZoneMap[destIndex] = m.ZoneMap[srcIndex]
		}
	}

	m.Cells = newCells
	m.ZoneMap = newZoneMap
	m.MapWidth = width
	m.MapHeight = height

	m.ApplyAmbientLight()
	m.UpdateBakedLights()
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ShiftMapBy(offset geometry.Point) {
	newCells := make([]MapCell[ActorType, ItemType, ObjectType], m.MapWidth*m.MapHeight)
	newZoneMap := make([]*ZoneInfo, m.MapWidth*m.MapHeight)
	for y := 0; y < m.MapHeight; y++ {
		for x := 0; x < m.MapWidth; x++ {
			xDest := (x + offset.X) % m.MapWidth
			yDest := (y + offset.Y) % m.MapHeight
			if xDest < 0 {
				xDest += m.MapWidth
			}
			if yDest < 0 {
				yDest += m.MapHeight
			}
			newCells[yDest*m.MapWidth+xDest] = m.Cells[y*m.MapWidth+x]
			newZoneMap[yDest*m.MapWidth+xDest] = m.ZoneMap[y*m.MapWidth+x]
		}
	}
	m.Cells = newCells
	m.ZoneMap = newZoneMap

	for _, item := range m.AllItems {
		oldPos := item.Pos()
		newPos := oldPos.AddWrapped(offset, m.MapSize())
		m.MoveItem(item, newPos)
	}

	for _, actor := range m.AllActors {
		oldPos := actor.Pos()
		newPos := oldPos.AddWrapped(offset, m.MapSize())
		actor.ShiftSchedulesBy(offset, m.MapSize())
		m.MoveActor(actor, newPos)
	}

	for _, actor := range m.AllDownedActors {
		oldPos := actor.Pos()
		newPos := oldPos.AddWrapped(offset, m.MapSize())
		actor.ShiftSchedulesBy(offset, m.MapSize())
		m.MoveDownedActor(actor, newPos)
	}

	for _, object := range m.AllObjects {
		oldPos := object.Pos()
		newPos := oldPos.AddWrapped(offset, m.MapSize())
		m.MoveObject(object, newPos)
	}
	m.PlayerSpawn = m.PlayerSpawn.AddWrapped(offset, m.MapSize())

	newBakedLights := make(map[geometry.Point]*LightSource, len(m.BakedLights))
	for oldPos, light := range m.BakedLights {
		newPos := oldPos.AddWrapped(offset, m.MapSize())
		light.Pos = newPos
		newBakedLights[newPos] = light
	}
	m.BakedLights = newBakedLights

	newDynamicLights := make(map[geometry.Point]*LightSource, len(m.DynamicLights))
	for oldPos, light := range m.DynamicLights {
		newPos := oldPos.AddWrapped(offset, m.MapSize())
		light.Pos = newPos
		newDynamicLights[newPos] = light
	}

	for name, loc := range m.NamedLocations {
		newPos := loc.AddWrapped(offset, m.MapSize())
		m.NamedLocations[name] = newPos
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MapSize() geometry.Point {
	return geometry.Point{X: m.MapWidth, Y: m.MapHeight}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) NeighborsAll(pos geometry.Point, filter func(p geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	return neighbors.All(pos, filter)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) NeighborsCardinal(pos geometry.Point, filter func(p geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	return neighbors.Cardinal(pos, filter)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetNearestWalkableNeighbor(start geometry.Point, dest geometry.Point) geometry.Point {
	minDist := math.MaxInt
	var minPos geometry.Point
	for _, neighbor := range m.NeighborsCardinal(dest, m.IsWalkable) {
		dist := geometry.DistanceManhattan(neighbor, start)
		if dist < minDist {
			minDist = dist
			minPos = neighbor
		}
	}
	return minPos
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsPassableForProjectile(p geometry.Point) bool {
	isTileWalkable := m.IsTileWalkable(p)
	isActorOnTile := m.IsActorAt(p)
	isObjectOnTile := m.IsObjectAt(p)
	isObjectBlocking := false
	if isObjectOnTile {
		objectOnTile := m.ObjectAt(p)
		isObjectBlocking = !objectOnTile.IsPassableForProjectile()
	}
	return isTileWalkable && !isActorOnTile && !isObjectBlocking
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsPathPassableForProjectile(source geometry.Point, target geometry.Point) bool {
	los := m.LineOfSight(source, target)
	return los[len(los)-1] == target
}

func (m *GridMap[ActorType, ItemType, ObjectType]) LineOfSight(source geometry.Point, target geometry.Point) []geometry.Point {
	los := geometry.LineOfSight(source, target, func(p geometry.Point) bool {
		return p == source || p == target || m.IsPassableForProjectile(p)
	})
	return los
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetAllExplored() {
	for y := 0; y < m.MapHeight; y++ {
		for x := 0; x < m.MapWidth; x++ {
			m.Cells[y*m.MapWidth+x].IsExplored = true
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RandomSpawnPosition() geometry.Point {
	for {
		x := rand.Intn(m.MapWidth)
		y := rand.Intn(m.MapHeight)
		pos := geometry.Point{X: x, Y: y}
		if m.IsCurrentlyPassable(pos) {
			return pos
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetNamedLocation(name string, point geometry.Point) {
	m.NamedLocations[name] = point
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetNamedLocation(name string) geometry.Point {
	return m.NamedLocations[name]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsNamedLocation(pos geometry.Point) bool {
	for _, location := range m.NamedLocations {
		if location == pos {
			return true
		}
	}
	return false
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetNamedLocationByPos(pos geometry.Point) string {
	for name, location := range m.NamedLocations {
		if location == pos {
			return name
		}
	}
	return ""
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RenameLocation(oldName string, newName string) {
	if pos, ok := m.NamedLocations[oldName]; ok {
		delete(m.NamedLocations, oldName)
		m.NamedLocations[newName] = pos
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveNamedLocation(namedLocation string) {
	delete(m.NamedLocations, namedLocation)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsLethalTileAt(p geometry.Point) bool {
	return m.CellAt(p).TileType.Special == SpecialTileLethal
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RandomPosAround(pos geometry.Point) geometry.Point {
	neighbors := m.NeighborsAll(pos, func(p geometry.Point) bool {
		return m.Contains(p)
	})
	if len(neighbors) == 0 {
		return pos
	}
	neighbors = append(neighbors, pos)
	return neighbors[rand.Intn(len(neighbors))]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) TryGetActorAt(pos geometry.Point) (ActorType, bool) {
	var noActor ActorType
	isActorAt := m.IsActorAt(pos)
	if !isActorAt {
		return noActor, false
	}
	return m.ActorAt(pos), isActorAt
}

func (m *GridMap[ActorType, ItemType, ObjectType]) TryGetDownedActorAt(pos geometry.Point) (ActorType, bool) {
	var noActor ActorType
	isDownedActorAt := m.IsDownedActorAt(pos)
	if !isDownedActorAt {
		return noActor, false
	}
	return m.DownedActorAt(pos), isDownedActorAt
}

func (m *GridMap[ActorType, ItemType, ObjectType]) TryGetObjectAt(pos geometry.Point) (ObjectType, bool) {
	var noObject ObjectType
	isObjectAt := m.IsObjectAt(pos)
	if !isObjectAt {
		return noObject, false
	}
	return m.ObjectAt(pos), isObjectAt
}

func (m *GridMap[ActorType, ItemType, ObjectType]) TryGetItemAt(pos geometry.Point) (ItemType, bool) {
	var noItem ItemType
	isItemAt := m.IsItemAt(pos)
	if !isItemAt {
		return noItem, false
	}
	return m.ItemAt(pos), isItemAt
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddActor(actor ActorType, spawnPos geometry.Point) {
	m.AllActors = append(m.AllActors, actor)
	m.MoveActor(actor, spawnPos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddDownedActor(actor ActorType, spawnPos geometry.Point) {
	m.AllDownedActors = append(m.AllDownedActors, actor)
	m.MoveDownedActor(actor, spawnPos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddObject(object ObjectType, spawnPos geometry.Point) {
	m.AllObjects = append(m.AllObjects, object)
	m.MoveObject(object, spawnPos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddItem(item ItemType, spawnPos geometry.Point) {
	m.AllItems = append(m.AllItems, item)
	m.MoveItem(item, spawnPos)
}
