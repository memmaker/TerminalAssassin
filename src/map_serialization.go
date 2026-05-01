package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

type MapSerializer struct {
    engine *ConsoleEngine
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
        if itemAt.Buried {
            record = append(record, rec_files.Field{Name: "Buried", Value: "true"})
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
        var buried bool
        for _, field := range record {
            switch field.Name {
            case "ItemAt":
                pos, _ = geometry.NewPointFromString(field.Value)
            case "Name":
                itemName = field.Value
            case "Key":
                keyString = field.Value
            case "Buried":
                buried = field.Value == "true"
            }
        }
        itemFactory := g.engine.GetItemFactory()
        itemRef := itemFactory.ItemFromNameAndKey(itemName, keyString)
        itemRef.Buried = buried
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
        }
        if keyboundObject, ok := objectAt.(services.KeyBound); ok && keyboundObject.GetKey() != "" {
            record = append(record, rec_files.Field{Name: "Key", Value: keyboundObject.GetKey()})
        }
        if diffHolder, ok := objectAt.(services.LockDifficultyHolder); ok {
            record = append(record, rec_files.Field{Name: "Difficulty", Value: diffHolder.GetLockDifficulty().ToString()})
        }
        if contentHolder, ok := objectAt.(services.ContentHolder); ok {
            for _, itemName := range contentHolder.GetContents() {
                record = append(record, rec_files.Field{Name: "Content", Value: itemName})
            }
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
        var key string
        var contents []string
        var difficulty string
        for _, field := range record {
            switch field.Name {
            case "ObjectAt":
                pos, _ = geometry.NewPointFromString(field.Value)
            case "Name":
                objectName = field.Value
            case "FgColor", "BgColor":
                // legacy field — ignored, colors come from the theme now
            case "Key":
                key = field.Value
            case "Difficulty":
                difficulty = field.Value
            case "Content":
                contents = append(contents, field.Value)
            }
        }
        object := factory.NewObjectFromName(objectName)
        if object == nil {
            println(fmt.Sprintf("Error loading object '%s'", objectName))
            continue
        }
        if keyboundObject, ok := object.(services.KeyBound); ok && key != "" {
            keyboundObject.SetKey(key)
        }
        if diffHolder, ok := object.(services.LockDifficultyHolder); ok && difficulty != "" {
            diffHolder.SetLockDifficulty(core.NewLockDifficultyFromString(difficulty))
        }
        if contentHolder, ok := object.(services.ContentHolder); ok && len(contents) > 0 {
            contentHolder.SetContents(contents)
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
        Inventory:     services.EncodeItems(person.Inventory),
        ActorType:     person.Type,
        IsTarget:      person.IsTarget,
        Team:          person.Team,
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

func (g *MapSerializer) SaveSchedules(currentMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], mapFolder string) error {
    schedulesFile, err := os.Create(path.Join(mapFolder, "schedules.txt"))
    if err != nil {
        return err
    }
    defer schedulesFile.Close()

    actorLinksFile, err := os.Create(path.Join(mapFolder, "actor_schedules.txt"))
    if err != nil {
        return err
    }
    defer actorLinksFile.Close()

    var schedulesAsRecords []rec_files.Record
    var links []rec_files.Record
    actors := currentMap.Actors()
    sort.SliceStable(actors, func(i, j int) bool { return actors[i].Name < actors[j].Name })

    for _, actor := range actors {
        if actor.AI.Schedule != "" {
            links = append(links, []rec_files.Field{
                {Name: "ForActorWithName", Value: actor.Name},
                {Name: "StartSchedule", Value: actor.AI.Schedule},
            })
        }
    }

    for _, sched := range g.engine.GetGame().GetMap().ListOfSchedules() {
        schedulesAsRecords = append(schedulesAsRecords, sched.ToRecords()...)
    }

    if err := rec_files.Write(schedulesFile, schedulesAsRecords); err != nil {
        return err
    }
    return rec_files.Write(actorLinksFile, links)
}

func (g *MapSerializer) LoadActorSchedules(files *Files, loadedMap *gridmap.GridMap[*core.Actor, *core.Item, services.Object], mapFolder string) error {
    // --- Load schedule definitions ---
    schedulesFilename := path.Join(mapFolder, "schedules.txt")
    schedulesFile, err := files.Open(schedulesFilename)
    if err != nil {
        return err
    }
    defer schedulesFile.Close()

    for _, sched := range gridmap.SchedulesFromTaskRecords(rec_files.Read(schedulesFile)) {
        loadedMap.AddSchedule(sched)
    }

    // --- Load actor→schedule links ---
    linksFilename := path.Join(mapFolder, "actor_schedules.txt")
    linksFile, err := files.Open(linksFilename)
    if err != nil {
        return err
    }
    defer linksFile.Close()

    actorsByName := make(map[string]*core.Actor)
    for _, a := range loadedMap.Actors() {
        actorsByName[a.Name] = a
    }

    scheduleCounter := 0
    for _, record := range rec_files.Read(linksFile) {
        actorName, scheduleName := gridmap.ActorScheduleLinkFromRecord(record)
        actor, hasActor := actorsByName[actorName]
        if !hasActor {
            println("Error loading actor schedule: no actor named '" + actorName + "'")
            continue
        }
        actor.AI.Schedule = scheduleName
        scheduleCounter++
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
    serializer := MapSerializer{engine: g}

    globalErr := serializer.SaveGlobalData(currentMap, mapFolder)
    if globalErr != nil {
        return globalErr
    }

	tileErr := serializer.SaveTiles(currentMap, path.Join(mapFolder, "tilemap.txt"))
	if tileErr != nil {
		return tileErr
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

    scheduleErr := serializer.SaveSchedules(currentMap, mapFolder)
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

	return nil
}

func (g *ConsoleEngine) LoadMap(mapFolder string) (*gridmap.GridMap[*core.Actor, *core.Item, services.Object], error) {
    serializer := MapSerializer{engine: g}
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

	tileErr := serializer.LoadTiles(files, loadedMap, path.Join(mapFolder, "tilemap.txt"))
	if tileErr != nil {
		return nil, tileErr
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

    scheduleErr := serializer.LoadActorSchedules(files, loadedMap, mapFolder)
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
