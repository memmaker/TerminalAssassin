package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
)

type MapSerializer struct {
	engine            *ConsoleEngine
	paletteForWriting map[common.RGBAColor]rune
	paletteForReading []common.Color
}

func (g *MapSerializer) SaveTiles(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	drawFunc := func(point geometry.Point) rune {
		return currentMap.GetCell(point).TileType.DefinedIcon
	}
	g.writeGrid(file, geometry.Point{X: currentMap.MapWidth, Y: currentMap.MapHeight}, drawFunc)
	return nil
}

func (g *MapSerializer) LoadTiles(files *Files, currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	tileCache := make(map[rune]gridmap.Tile)
	data := g.engine.ExternalData
	setFunc := func(pos geometry.Point, icon rune) {
		if _, ok := tileCache[icon]; !ok {
			tileCache[icon] = data.TileFromIcon(icon)
		}
		currentMap.SetTile(pos, tileCache[icon])
	}
	readErr := g.readGrid(file, geometry.Point{X: currentMap.MapWidth, Y: currentMap.MapHeight}, setFunc)
	if readErr != nil {
		return readErr
	}
	return nil
}
func (g *MapSerializer) writeGrid(file *os.File, gridSize geometry.Point, writeFunc func(point geometry.Point) rune) {
	for y := 0; y < gridSize.Y; y++ {
		for x := 0; x < gridSize.X; x++ {
			file.WriteString(string(writeFunc(geometry.Point{X: x, Y: y})))
		}
		file.WriteString("\n")
	}
	file.WriteString("\n")
}
func (g *MapSerializer) readGrid(file fs.File, gridSize geometry.Point, readFunc func(pos geometry.Point, icon rune)) error {
	for y := 0; y < gridSize.Y; y++ {
		for x := 0; x < gridSize.X; x++ {
			var icon rune
			_, err := fmt.Fscanf(file, "%c", &icon)
			if err != nil {
				return err
			}
			readFunc(geometry.Point{X: x, Y: y}, icon)
		}
		// read and discard newline
		var newline rune
		fmt.Fscanf(file, "%c", &newline)
	}
	return nil
}

func (g *MapSerializer) SaveItemLocations(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	records := make([]rec_files.Record, 0)
	for _, itemAt := range currentMap.Items() {
		record := []rec_files.Field{
			{Name: "ItemAt", Value: itemAt.Pos().String()},
			{Name: "Name", Value: itemAt.Name},
		}
		if itemAt.KeyString != "" {
			record = append(record, rec_files.Field{Name: "Key", Value: itemAt.GetKey()})
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i][0].Value < records[j][0].Value
	})

	return rec_files.Write(file, records)
}

func (g *MapSerializer) LoadItemLocations(files *Files, currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	itemCount := 0
	records := rec_files.Read(file)
	for _, record := range records {
		var pos geometry.Point
		var itemName string
		var keyString string
		for _, field := range record {
			switch field.Name {
			case "ItemAt":
				pos, _ = geometry.NewPointFromString(field.Value)
			case "Name":
				itemName = field.Value
			case "Key":
				keyString = field.Value
			}
		}
		itemFactory := g.engine.GetItemFactory()
		item := itemFactory.DecodeStringToItem(itemName)
		itemRef := &item
		if keyString != "" {
			itemRef.SetKey(keyString)
		}
		currentMap.AddItem(itemRef, pos)
		itemCount++
	}

	println(fmt.Sprintf("Loaded %d items", itemCount))
	return file.Close()
}

func (g *MapSerializer) SaveObjects(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	records := make([]rec_files.Record, 0)
	for _, objectAt := range currentMap.Objects() {
		record := []rec_files.Field{
			{Name: "ObjectAt", Value: objectAt.Pos().String()},
			{Name: "Name", Value: objectAt.EncodeAsString()},
			{Name: "FgColor", Value: objectAt.GetStyle().Foreground.EncodeAsString()},
			{Name: "BgColor", Value: objectAt.GetStyle().Background.EncodeAsString()},
		}
		if keyboundObject, ok := objectAt.(services.KeyBound); ok && keyboundObject.GetKey() != "" {
			record = append(record, rec_files.Field{Name: "Key", Value: keyboundObject.GetKey()})
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i][0].Value < records[j][0].Value
	})
	return rec_files.Write(file, records)
}
func (g *MapSerializer) LoadObjects(files *Files, loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	records := rec_files.Read(file)
	factory := g.engine.ObjectFactory

	for _, record := range records {
		var pos geometry.Point
		var objectName string
		var fgColor, bgColor common.Color
		var key string
		for _, field := range record {
			switch field.Name {
			case "ObjectAt":
				pos, _ = geometry.NewPointFromString(field.Value)
			case "Name":
				objectName = field.Value
			case "FgColor":
				fgColor = common.NewColorFromString(field.Value)
			case "BgColor":
				bgColor = common.NewColorFromString(field.Value)
			case "Key":
				key = field.Value
			}
		}
		object := factory.NewObjectFromName(objectName)
		if object == nil {
			println(fmt.Sprintf("Error loading object '%s'", objectName))
			continue
		}
		object.SetStyle(common.Style{Foreground: fgColor, Background: bgColor})
		if keyboundObject, ok := object.(services.KeyBound); ok && key != "" {
			keyboundObject.SetKey(key)
		}
		loadedMap.AddObject(object, pos)
	}

	println(fmt.Sprintf("Loaded %d objects", len(records)))
	return nil
}

func (g *MapSerializer) SaveActors(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	listOfActors := make([]rec_files.Record, 0)
	for _, actorAt := range currentMap.Actors() {
		onDiskActor := NewActorOnDiskFromActor(actorAt)
		listOfActors = append(listOfActors, onDiskActor.ToRecord())
	}
	sort.SliceStable(listOfActors, func(i, j int) bool {
		return listOfActors[i][0].Value < listOfActors[j][0].Value
	})
	return rec_files.Write(file, listOfActors)
}

func NewActorOnDiskFromActor(person *core.Actor) core.ActorOnDisk {
	return core.ActorOnDisk{
		Name:          person.Name,
		Clothing:      person.Clothes.Name,
		Inventory:     services.EncodeItems(person.Inventory),
		ActorType:     person.Type,
		MoveSpeed:     person.AutoMoveSpeed,
		FoVinDegrees:  person.FoVinDegrees,
		VisionRange:   person.MaxVisionRange,
		LookDirection: person.LookDirection,
		Position:      person.MapPos,
	}
}
func (g *MapSerializer) LoadActors(files *Files, loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	data := g.engine.ExternalData
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	actorCounter := 0

	records := rec_files.Read(file)
	for _, record := range records {
		onDiskActor := core.ActorOnDiskFromRecord(record)
		newActor := data.NewActorFromDisk(g.engine.ItemFactory, onDiskActor)
		if newActor.IsDowned() {
			loadedMap.AddDownedActor(newActor, newActor.Pos())
		} else {
			loadedMap.AddActor(newActor, newActor.Pos())
		}
		actorCounter++
	}
	println(fmt.Sprintf("Loaded %d actors", actorCounter))
	return nil
}

func (g *MapSerializer) SaveActorSchedules(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	listOfSchedules := make([]rec_files.Record, 0)
	for _, actorAt := range currentMap.Actors() {
		scheduleAsRecord := actorAt.AI.Schedule.ToRecord(actorAt.Pos())
		if len(scheduleAsRecord) == 0 {
			continue
		}
		listOfSchedules = append(listOfSchedules, scheduleAsRecord)
	}
	sort.SliceStable(listOfSchedules, func(i, j int) bool {
		return listOfSchedules[i][0].Value < listOfSchedules[j][0].Value
	})
	return rec_files.Write(file, listOfSchedules)
}

func (g *MapSerializer) LoadActorSchedules(files *Files, loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	records := rec_files.Read(file)
	scheduleCounter := 0
	for _, record := range records {
		schedule := core.ScheduleFromRecord(record)
		pos, _ := geometry.NewPointFromString(record.ToMap()["ForActorAt"])
		actorAt, isActorAt := loadedMap.TryGetActorAt(pos)
		if isActorAt {
			actorAt.AI.Schedule = *schedule
			scheduleCounter++
		} else {
			println("Error loading actor schedule: no actor at " + pos.String())
		}
	}

	println(fmt.Sprintf("Loaded %d actor schedules", scheduleCounter))
	return nil
}

func (g *MapSerializer) SaveGlobalData(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], newMapFolder string) error {
	globalFile := path.Join(newMapFolder, "global.txt")
	file, err := os.Create(globalFile)
	if err != nil {
		return err
	}
	defer file.Close()
	globalData := NewGlobalDataFromMap(currentMap)
	oldMapFolder := currentMap.MapFileName()

	if oldMapFolder != newMapFolder && oldMapFolder != "" {
		println(fmt.Sprintf("Saving under new map folder %s (Copying manually added files now)", newMapFolder))
		files := g.engine.Files
		g.copyManuallyAddedFiles(files, oldMapFolder, newMapFolder)
	}

	return rec_files.Write(file, []rec_files.Record{globalData.ToRecord()})
}

func (g *MapSerializer) LoadGlobalData(files *Files, mapFolder string) (gridmap.GlobalMapDataOnDisk, error) {
	globalFile := path.Join(mapFolder, "global.txt")
	file, err := files.Open(globalFile)
	if err != nil {
		return gridmap.GlobalMapDataOnDisk{}, err
	}
	defer file.Close()
	records := rec_files.Read(file)
	globalMapDataRecord := records[0]
	globalData := gridmap.NewGlobalMapFromRecord(globalMapDataRecord)

	println("Loading map\n" + globalData.ToString())
	return globalData, nil
}
func (g *MapSerializer) copyFileIfExists(files *Files, oldPath, newPath string) {
	if files.FileExists(oldPath) {
		err := g.copyFile(files, oldPath, newPath)
		if err != nil {
			println(fmt.Sprintf("Error copying file %s to %s: %s", oldPath, newPath, err.Error()))
		}
	}
}

func (g *MapSerializer) copyFile(files *Files, oldPath string, newPath string) error {
	file, err := files.Open(oldPath)
	if err != nil {
		return err
	}
	defer file.Close()
	newFile, err := os.Create(newPath)
	if err != nil {
		return err
	}
	defer newFile.Close()
	_, err = io.Copy(newFile, file)
	if err != nil {
		return err
	}
	println(fmt.Sprintf("Copied file %s to %s", oldPath, newPath))
	return nil
}
func (g *MapSerializer) copyManuallyAddedFiles(files *Files, oldFolder, newFolder string) {
	// optional, manually added files
	g.copyFileIfExists(files,
		path.Join(oldFolder, "challenges.txt"),
		path.Join(newFolder, "challenges.txt"),
	)

	g.copyFileIfExists(files,
		path.Join(oldFolder, "start_locations.txt"),
		path.Join(newFolder, "start_locations.txt"),
	)

	g.copyFileIfExists(files,
		path.Join(oldFolder, "unlocks.txt"),
		path.Join(newFolder, "unlocks.txt"),
	)
	os.MkdirAll(path.Join(newFolder, "briefing"), 0777)
	os.MkdirAll(path.Join(newFolder, "dialogues"), 0777)
	os.MkdirAll(path.Join(newFolder, "scripts"), 0777)

	for _, oldFilename := range files.GetFilesInPath(path.Join(oldFolder, "briefing")) {
		newFilename := path.Join(newFolder, "briefing", path.Base(oldFilename))
		err := g.copyFile(files, oldFilename, newFilename)
		if err != nil {
			println(fmt.Sprintf("Error copying file %s to %s: %s", oldFilename, newFilename, err.Error()))
		}
	}

	for _, oldFilename := range files.GetFilesInPath(path.Join(oldFolder, "dialogues")) {
		newFilename := path.Join(newFolder, "dialogues", path.Base(oldFilename))
		err := g.copyFile(files, oldFilename, newFilename)
		if err != nil {
			println(fmt.Sprintf("Error copying file %s to %s: %s", oldFilename, newFilename, err.Error()))
		}
	}
	for _, oldFilename := range files.GetFilesInPath(path.Join(oldFolder, "scripts")) {
		newFilename := path.Join(newFolder, "scripts", path.Base(oldFilename))
		err := g.copyFile(files, oldFilename, newFilename)
		if err != nil {
			println(fmt.Sprintf("Error copying file %s to %s: %s", oldFilename, newFilename, err.Error()))
		}
	}
}
func (g *MapSerializer) ApplyGlobalMapData(loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], globalData gridmap.GlobalMapDataOnDisk) {
	loadedMap.MapWidth = globalData.Width
	loadedMap.MapHeight = globalData.Height
	loadedMap.PlayerSpawn = globalData.PlayerSpawn
	loadedMap.MaxVisionRange = globalData.MaxVisionRange
	loadedMap.AmbientLight = globalData.AmbientLight
	loadedMap.MaxLightIntensity = globalData.MaxLightIntensity
	loadedMap.MetaData.MissionTitle = globalData.MissionTitle
	loadedMap.TimeOfDay = globalData.TimeOfDay
	loadedMap.AmbienceSoundCue = globalData.AmbienceSoundCue
}

func (g *MapSerializer) SaveBakedLights(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	records := make([]rec_files.Record, len(currentMap.BakedLights))
	index := 0
	for _, lightAt := range currentMap.BakedLights {
		records[index] = lightAt.ToRecord()
		index++
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i][0].Value < records[j][0].Value
	})
	return rec_files.Write(file, records)
}

func (g *MapSerializer) LoadBakedLights(files *Files, loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	records := rec_files.Read(file)
	for _, record := range records {
		light := gridmap.NewLightSourceFromRecord(record)
		loadedMap.AddBakedLightSource(light.Pos, light)
	}

	println(fmt.Sprintf("Loaded %d baked lights", len(records)))

	return nil
}

func (g *MapSerializer) SaveDynamicLights(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	records := make([]rec_files.Record, len(currentMap.DynamicLights))
	index := 0
	for _, lightAt := range currentMap.DynamicLights {
		records[index] = lightAt.ToRecord()
		index++
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i][0].Value < records[j][0].Value
	})
	return rec_files.Write(file, records)
}

func (g *MapSerializer) LoadDynamicLights(files *Files, loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	records := rec_files.Read(file)
	for _, record := range records {
		light := gridmap.NewLightSourceFromRecord(record)
		loadedMap.AddDynamicLightSource(light.Pos, light)
	}

	println(fmt.Sprintf("Loaded %d dynamic lights", len(records)))
	return nil
}

func (g *MapSerializer) colorToPaletteIndex(color common.RGBAColor) rune {
	if index, ok := g.paletteForWriting[color]; ok {
		return index
	}
	index := rune(len(g.paletteForWriting) + 33)
	g.paletteForWriting[color] = index
	return index
}
func (g *MapSerializer) paletteIndexToColor(index rune) common.Color {
	return g.paletteForReading[int(index-33)]
}
func (g *MapSerializer) SaveTileColors(loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename + ".fg")
	if err != nil {
		return err
	}
	defer file.Close()

	writeFunc := func(pos geometry.Point) rune {
		tileStyle := loadedMap.GetCell(pos).TileType.DefinedStyle
		return g.colorToPaletteIndex(tileStyle.Foreground.ToRGB())
	}
	g.writeGrid(file, geometry.Point{loadedMap.MapWidth, loadedMap.MapHeight}, writeFunc)

	bgFile, err := os.Create(filename + ".bg")
	if err != nil {
		return err
	}
	defer bgFile.Close()

	bgWriteFunc := func(pos geometry.Point) rune {
		tileStyle := loadedMap.GetCell(pos).TileType.DefinedStyle
		return g.colorToPaletteIndex(tileStyle.Background.ToRGB())
	}
	g.writeGrid(bgFile, geometry.Point{loadedMap.MapWidth, loadedMap.MapHeight}, bgWriteFunc)

	return nil
}

func (g *MapSerializer) LoadTileColors(files *Files, loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	fgFile, fgErr := files.Open(filename + ".fg")
	if fgErr != nil {
		return fgErr
	}
	defer fgFile.Close()
	fgReadFunc := func(pos geometry.Point, index rune) {
		color := g.paletteIndexToColor(index)
		cell := loadedMap.GetCell(pos)
		cell.TileType.DefinedStyle.Foreground = color
		loadedMap.SetCell(pos, cell)
	}
	fgReadErr := g.readGrid(fgFile, geometry.Point{loadedMap.MapWidth, loadedMap.MapHeight}, fgReadFunc)
	if fgReadErr != nil {
		return fgReadErr
	}

	bgFile, bgErr := files.Open(filename + ".bg")
	if bgErr != nil {
		return bgErr
	}
	defer bgFile.Close()
	bgReadFunc := func(pos geometry.Point, index rune) {
		color := g.paletteIndexToColor(index)
		cell := loadedMap.GetCell(pos)
		cell.TileType.DefinedStyle.Background = color
		loadedMap.SetCell(pos, cell)
	}
	return g.readGrid(bgFile, geometry.Point{loadedMap.MapWidth, loadedMap.MapHeight}, bgReadFunc)
}

func (g *MapSerializer) SaveZones(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	records := make([]rec_files.Record, len(currentMap.ListOfZones)-1)
	for zoneIndex, zoneInfo := range currentMap.ListOfZones {
		if zoneIndex == 0 {
			continue
		}
		record := zoneInfo.ToRecord()
		records[zoneIndex-1] = record
	}
	return rec_files.Write(file, records)
}
func (g *MapSerializer) SaveZoneMap(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writeFunc := func(pos geometry.Point) rune {
		zoneAt := currentMap.ZoneAt(pos)
		return rune(indexOf(zoneAt, currentMap.ListOfZones) + 32)
	}
	g.writeGrid(file, geometry.Point{X: currentMap.MapWidth, Y: currentMap.MapHeight}, writeFunc)
	return nil
}

func indexOf(at *gridmap.ZoneInfo, zones []*gridmap.ZoneInfo) int {
	for index, zone := range zones {
		if zone == at {
			return index
		}
	}
	return 0
}

func (g *MapSerializer) LoadZones(files *Files, currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	records := rec_files.Read(file)
	zoneCounter := 0
	for _, record := range records {
		zoneInfo := gridmap.NewZoneFromRecord(record)
		currentMap.AddZone(zoneInfo)
		zoneCounter++
	}
	println(fmt.Sprintf("Loaded %d zones", zoneCounter))
	return file.Close()
}
func (g *MapSerializer) LoadZoneMap(files *Files, currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := files.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	readFunc := func(pos geometry.Point, r rune) {
		zoneIndex := int(r) - 32
		if zoneIndex >= 0 && zoneIndex < len(currentMap.ListOfZones) {
			zoneToSet := currentMap.ListOfZones[zoneIndex]
			currentMap.SetZone(pos, zoneToSet)
		}
	}
	return g.readGrid(file, geometry.Point{X: currentMap.MapWidth, Y: currentMap.MapHeight}, readFunc)
}
func (g *MapSerializer) SaveNamedLocations(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	// we need to save currentMap.NamedLocations
	records := make([]rec_files.Record, 0)
	for name, location := range currentMap.NamedLocations {
		records = append(records, []rec_files.Field{
			{Name: "Name", Value: name},
			{Name: "Location", Value: location.String()},
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i][0].Value < records[j][0].Value
	})
	return rec_files.Write(file, records)
}

func (g *MapSerializer) LoadNamedLocations(files *Files, currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], filename string) error {
	currentMap.NamedLocations = make(map[string]geometry.Point, 0)
	file, err := files.Open(filename)
	if err != nil {
		return err
	}

	records := rec_files.Read(file)
	for _, record := range records {
		var name string
		var location geometry.Point
		for _, field := range record {
			if field.Name == "Name" {
				name = field.Value
			} else if field.Name == "Location" {
				location, _ = geometry.NewPointFromString(field.Value)
			}
		}
		currentMap.NamedLocations[name] = location
	}

	println(fmt.Sprintf("Loaded %d named locations", len(currentMap.NamedLocations)))
	return file.Close()
}

func (g *MapSerializer) SaveCurrentPalette(filename string) error {
	palFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	paletteAsList := make([]common.RGBAColor, len(g.paletteForWriting))
	for color, index := range g.paletteForWriting {
		paletteAsList[index-33] = color
	}
	for _, color := range paletteAsList {
		fmt.Fprintln(palFile, color.EncodeAsString())
	}
	return palFile.Close()
}

func (g *MapSerializer) LoadPalette(files *Files, filename string) error {
	g.paletteForReading = make([]common.Color, 0)
	palFile, err := files.Open(filename)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(palFile)
	paletteIndex := 0
	for scanner.Scan() {
		line := scanner.Text()
		color := common.NewColorFromString(line)
		g.paletteForReading = append(g.paletteForReading, color)
		paletteIndex++
	}
	return palFile.Close()
}

func NewGlobalDataFromMap(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object]) gridmap.GlobalMapDataOnDisk {
	return gridmap.GlobalMapDataOnDisk{
		Width:             currentMap.MapWidth,
		Height:            currentMap.MapHeight,
		PlayerSpawn:       currentMap.PlayerSpawn,
		MaxVisionRange:    currentMap.MaxVisionRange,
		AmbientLight:      currentMap.AmbientLight,
		MaxLightIntensity: currentMap.MaxLightIntensity,
		MissionTitle:      currentMap.MetaData.MissionTitle,
		TimeOfDay:         currentMap.TimeOfDay,
		AmbienceSoundCue:  currentMap.AmbienceSoundCue,
	}
}

func (g *ConsoleEngine) SaveMap(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], mapFolder string) error {
	serializer := MapSerializer{engine: g, paletteForWriting: make(map[common.RGBAColor]rune)}

	globalErr := serializer.SaveGlobalData(currentMap, mapFolder)
	if globalErr != nil {
		return globalErr
	}

	tileErr := serializer.SaveTiles(currentMap, path.Join(mapFolder, "tilemap.txt"))
	if tileErr != nil {
		return tileErr
	}

	tileColorErr := serializer.SaveTileColors(currentMap, path.Join(mapFolder, "tile_colors"))
	if tileColorErr != nil {
		return tileColorErr
	}

	itemErr := serializer.SaveItemLocations(currentMap, path.Join(mapFolder, "item_locations.txt"))
	if itemErr != nil {
		return itemErr
	}

	objectErr := serializer.SaveObjects(currentMap, path.Join(mapFolder, "objects.txt"))
	if objectErr != nil {
		return objectErr
	}

	actorErr := serializer.SaveActors(currentMap, path.Join(mapFolder, "actors.txt"))
	if actorErr != nil {
		return actorErr
	}

	scheduleErr := serializer.SaveActorSchedules(currentMap, path.Join(mapFolder, "actor_schedules.txt"))
	if scheduleErr != nil {
		return scheduleErr
	}

	bakedLightsErr := serializer.SaveBakedLights(currentMap, path.Join(mapFolder, "baked_lights.txt"))
	if bakedLightsErr != nil {
		return bakedLightsErr
	}

	dynamicLightsErr := serializer.SaveDynamicLights(currentMap, path.Join(mapFolder, "dynamic_lights.txt"))
	if dynamicLightsErr != nil {
		return dynamicLightsErr
	}

	saveZoneErr := serializer.SaveZones(currentMap, path.Join(mapFolder, "zones.txt"))
	if saveZoneErr != nil {
		return saveZoneErr
	}

	zoneMapErr := serializer.SaveZoneMap(currentMap, path.Join(mapFolder, "zone_map.txt"))
	if zoneMapErr != nil {
		return zoneMapErr
	}

	namedLocationsErr := serializer.SaveNamedLocations(currentMap, path.Join(mapFolder, "named_locations.txt"))
	if namedLocationsErr != nil {
		return namedLocationsErr
	}

	paletteErr := serializer.SaveCurrentPalette(path.Join(mapFolder, "palette.txt"))
	if paletteErr != nil {
		return paletteErr
	}
	return nil
}

func (g *ConsoleEngine) LoadMap(mapFolder string) (*gridmap.GridMap[*core.Actor, *core.Item, services.Object], error) {
	serializer := MapSerializer{engine: g, paletteForWriting: make(map[common.RGBAColor]rune)}
	files := g.Files
	globalData, globalErr := serializer.LoadGlobalData(files, mapFolder)
	if globalErr != nil {
		println("Error loading global data: " + globalErr.Error())
		return gridmap.NewEmptyMap[*core.Actor, *core.Item, services.Object](g.MapWindowWidth(), g.MapWindowHeight(), g.Config.MaxVisionRange), globalErr
	}

	loadedMap := gridmap.NewEmptyMap[*core.Actor, *core.Item, services.Object](globalData.Width, globalData.Height, globalData.MaxVisionRange)
	loadedMap.MetaData.FileName = mapFolder
	hash := sha256.Sum256([]byte(mapFolder))
	loadedMap.MetaData.HashAsHex = hex.EncodeToString(hash[:])

	serializer.ApplyGlobalMapData(loadedMap, globalData)

	paletteErr := serializer.LoadPalette(files, path.Join(mapFolder, "palette.txt"))
	if paletteErr != nil {
		return nil, paletteErr
	}

	tileErr := serializer.LoadTiles(files, loadedMap, path.Join(mapFolder, "tilemap.txt"))
	if tileErr != nil {
		return nil, tileErr
	}

	tileColorErr := serializer.LoadTileColors(files, loadedMap, path.Join(mapFolder, "tile_colors"))
	if tileColorErr != nil {
		println("Error loading tile colors: " + tileColorErr.Error())
	}
	itemErr := serializer.LoadItemLocations(files, loadedMap, path.Join(mapFolder, "item_locations.txt"))
	if itemErr != nil {
		println("Error loading item locations: " + itemErr.Error())
	}

	objectErr := serializer.LoadObjects(files, loadedMap, path.Join(mapFolder, "objects.txt"))
	if objectErr != nil {
		println("Error loading objects: " + objectErr.Error())
	}

	actorErr := serializer.LoadActors(files, loadedMap, path.Join(mapFolder, "actors.txt"))
	if actorErr != nil {
		println("Error loading actors: " + actorErr.Error())
	}

	scheduleErr := serializer.LoadActorSchedules(files, loadedMap, path.Join(mapFolder, "actor_schedules.txt"))
	if scheduleErr != nil {
		println("Error loading actor schedules: " + scheduleErr.Error())
	}

	bakedLightsErr := serializer.LoadBakedLights(files, loadedMap, path.Join(mapFolder, "baked_lights.txt"))
	if bakedLightsErr != nil {
		println("Error loading baked lights: " + bakedLightsErr.Error())
	}

	dynamicLightsErr := serializer.LoadDynamicLights(files, loadedMap, path.Join(mapFolder, "dynamic_lights.txt"))
	if dynamicLightsErr != nil {
		println("Error loading dynamic lights: " + dynamicLightsErr.Error())
	}

	zonesErr := serializer.LoadZones(files, loadedMap, path.Join(mapFolder, "zones.txt"))
	if zonesErr != nil {
		println("Error loading zones: " + zonesErr.Error())
	}

	zoneMapErr := serializer.LoadZoneMap(files, loadedMap, path.Join(mapFolder, "zone_map.txt"))
	if zoneMapErr != nil {
		println("Error loading zone map: " + zoneMapErr.Error())
	}

	namedLocationsErr := serializer.LoadNamedLocations(files, loadedMap, path.Join(mapFolder, "named_locations.txt"))
	if namedLocationsErr != nil {
		println("Error loading named locations: " + namedLocationsErr.Error())
	}

	return loadedMap, nil
}
