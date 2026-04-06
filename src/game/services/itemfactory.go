package services

import (
    "fmt"
    "os"
    "path"
    "path/filepath"
    "regexp"
    "time"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/gridmap"
)

func NewFactory(engine Engine) *ItemFactory {
    return &ItemFactory{engine: engine}
}

type ItemFactory struct {
    engine Engine
}

func (f ItemFactory) CreateRemoteDetonator(source *core.Actor, item *core.Item) *core.Item {
    detonateItem := func() {
        f.engine.GetGame().SendTriggerStimuli(source, item, item.Pos(), core.TriggerOnRemoteControl)
    }
    remote := &core.Item{Name: "remote detonator", DefinedIcon: core.GlyphLockpickElectronic, Type: core.ItemTypeTool, InsteadOfUse: detonateItem, Uses: 1, DefinedStyle: item.DefinedStyle}
    remote.OnCooldown = true
    return remote
}

func (f ItemFactory) CreateEmptyPieceOfPaper() *core.Item {
	pieceOfPaperItem := &core.Item{
		Name:         "Note",
        Type:         core.ItemTypeMessage,
        DefinedIcon:  core.GlyphFilledSquare,
        DefinedStyle: common.DefaultStyle.Reversed(),
        Uses:         -1,
    }
    var paperCache []string
    engine := f.engine
    insteadOfUse := func() {
        if paperCache == nil {
            currentMap := engine.GetGame().GetMap()
            mapFolder := currentMap.MapFileName()
            textfile := path.Join(mapFolder, "texts", pieceOfPaperItem.KeyString+".txt")
            files := engine.GetFiles()
            paperCache = files.LoadTextFile(textfile)
        }
        userInterface := engine.GetUI()
        userInterface.ShowAlert(paperCache)
    }
    pieceOfPaperItem.InsteadOfUse = insteadOfUse
    return pieceOfPaperItem
}

// ItemFromNameAndKey constructs an item from the name and key fields of a map record.
// Pattern-based encoded forms (Key(...) etc.) never appear in item_locations.txt,
// so only a direct name lookup and the paper fallback are needed here.
func (f ItemFactory) ItemFromNameAndKey(name string, key string) *core.Item {
    externalData := f.engine.GetData()
    if item, ok := externalData.ItemByName(name); ok {
        result := *item
        if key != "" {
            result.SetKey(key)
        }
        f.applyItemBehavior(&result)
        return &result
    }
    if key != "" {
		if name == "Note" {
            return f.CreatePieceOfPaper(name, key)
        } else if name == "Key card" {
            return core.NewKeyCard(key)
        } else if name == "Key" {
            return core.NewKey(key)
        }
    }
    println("WARNING: Could not find item with name " + name)
    return &core.Item{}
}

func (f ItemFactory) DecodeStringToItem(name string) core.Item {
    externalData := f.engine.GetData()
    keyPattern := regexp.MustCompile(`^Key\((.*)\)$`)
    keyCardPattern := regexp.MustCompile(`^KeyCard\((.*)\)$`)
    paperPattern := regexp.MustCompile(`^Message\((.*)\): (.*)$`)
    if keyPattern.MatchString(name) {
        submatches := keyPattern.FindStringSubmatch(name)
        keyString := submatches[1]
        return *core.NewKey(keyString)
    } else if keyCardPattern.MatchString(name) {
        submatches := keyCardPattern.FindStringSubmatch(name)
        keyString := submatches[1]
        return *core.NewKeyCard(keyString)
    } else if paperPattern.MatchString(name) {
        submatches := paperPattern.FindStringSubmatch(name)
        textFilename := submatches[1]
        paperName := submatches[2]
        paper := f.CreatePieceOfPaper(paperName, textFilename)
        return *paper
    }
    if item, ok := externalData.ItemByName(name); ok {
        result := *item
        f.applyItemBehavior(&result)
        return result
    }
    println("WARNING: Could not find item with name " + name)
    return core.Item{}
}
func (f ItemFactory) StringsToItems(inventory []string) []*core.Item {
    items := make([]*core.Item, len(inventory))
    for i, name := range inventory {
        fromName := f.DecodeStringToItem(name)
        items[i] = &fromName
    }
    return items
}

func (f ItemFactory) ComplexItems() []*core.Item {
    return []*core.Item{
        core.NewEmptyKeyCard(),
        core.NewEmptyKey(),
        f.CreateEmptyPieceOfPaper(),
    }
}

func (f ItemFactory) CreatePieceOfPaper(name string, filename string) *core.Item {
    emptyPaper := f.CreateEmptyPieceOfPaper()
    emptyPaper.Name = name
    emptyPaper.KeyString = filename
    return emptyPaper
}

// need a way to convert items to strings
func EncodeItemAsString(item *core.Item) string {
    if item.Type == core.ItemTypeKeyCard {
        return fmt.Sprintf("KeyCard(%s)", item.KeyString)
    } else if item.Type == core.ItemTypeKey {
        return fmt.Sprintf("Key(%s)", item.KeyString)
    } else if item.Type == core.ItemTypeMessage {
        return fmt.Sprintf("Message(%s): %s", item.KeyString, item.Name)
    }
    return item.Name
}

// applyItemBehavior wires up engine-bound runtime behaviour (InsteadOfUse etc.)
// for any item type that requires it, based solely on the item's Type field.
// Call this whenever an item is produced from external data so the data file
// provides the static definition while the factory provides the behaviour.
func (f ItemFactory) applyItemBehavior(item *core.Item) {
    switch item.Type {
    case core.ItemTypeCamera:
        f.applyCameraBehavior(item)
    case core.ItemTypeShovel:
        f.applyShovelBehavior(item)
    }
}

// applyShovelBehavior wires up the shovel's InsteadOfUse action.
// When used, it searches for buried items at the player's FoV source position
// and uncovers the first one found.
func (f ItemFactory) applyShovelBehavior(item *core.Item) {
    engine := f.engine
    item.InsteadOfUse = func() {
        currentMap := engine.GetGame().GetMap()
        player := currentMap.Player
        digPos := player.FoVSource()

        if !currentMap.IsItemAt(digPos) {
            engine.GetGame().PrintMessage("Nothing to find here.")
            return
        }
        itemAtPos := currentMap.ItemAt(digPos)
        if itemAtPos == nil || !itemAtPos.Buried {
            engine.GetGame().PrintMessage("Nothing to find here.")
            return
        }
        itemAtPos.Buried = false
        engine.GetGame().PrintMessage("You uncover " + itemAtPos.Name + ".")
        engine.GetGame().UpdateHUD()
    }
}

// applyCameraBehavior attaches the camera flash + screenshot + metadata
// InsteadOfUse closure to an item whose static properties come from the data file.
func (f ItemFactory) applyCameraBehavior(item *core.Item) {
    engine := f.engine

    // Each camera item gets its own flash light instance so concurrent
    // uses (e.g. if multiple cameras exist) don't share state.
    cameraFlashLight := &gridmap.LightSource{
        Radius:       8,
        Color:        common.RGBAColor{R: 1, G: 1, B: 1, A: 1.0},
        MaxIntensity: 20,
    }

    item.InsteadOfUse = func() {
        currentMap := engine.GetGame().GetMap()
        playerPos := currentMap.Player.Pos()

        // Camera flash — same mechanism as a gun muzzle flash, just brighter.
        if engine.GetGame().GetConfig().LightSources {
            engine.ScheduleInTicks(1, func() {
                cameraFlashLight.Pos = playerPos
                currentMap.AddDynamicLightSource(playerPos, cameraFlashLight)
                currentMap.UpdateDynamicLights()
                engine.Schedule(0.150, func() {
                    currentMap.RemoveDynamicLightAt(playerPos)
                    currentMap.UpdateDynamicLights()
                })
            })
        }

        // Collect scene metadata.
        metadata := CapturePhotoMetadata(engine)

        // Derive timestamped output paths.
        timestamp := time.Now().Format("2006-01-02_15-04-05")
        photoDir := "photos"
        if err := os.MkdirAll(photoDir, 0o755); err != nil {
            println("Camera: could not create photos directory:", err.Error())
            return
        }
        basePath := filepath.Join(photoDir, "photo_"+timestamp)

        // Request screenshot (captured on the next render frame).
        engine.RequestScreenshot(basePath + ".png")

        // Save JSON sidecar.
        SavePhotoMetadata(metadata, basePath+".json")
    }
}
