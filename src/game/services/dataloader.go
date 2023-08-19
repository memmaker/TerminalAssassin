package services

import (
    "bufio"
    "fmt"
    "github.com/memmaker/terminal-assassin/game/core"

    //"github.com/memmaker/terminal-assassin/game/services"
    "io/fs"
    "path"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/game/stimuli"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/gridmap"
    "github.com/memmaker/terminal-assassin/mapset"
    rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

type DataSource interface {
    Open(filename string) (fs.File, error)
    ReadDir(path string) ([]fs.DirEntry, error)
}

func NewExternalDataFromDisk(files DataSource) *ExternalData {
    e := &ExternalData{
        items: []*core.Item{},
        tiles: []*gridmap.Tile{},
    }
    e.LoadCoreData(files)
    return e
}

type ExternalData struct {
    items                  []*core.Item
    tiles                  []*gridmap.Tile
    clothing               []*core.Clothing
    defaultFloor           *gridmap.Tile
    defaultClothing        *core.Clothing
    defaultPlayerClothing  *core.Clothing
    defaultWeapon          *core.Item
    defaultItem            *core.Item
    ItemUnlockMap          map[string][]*core.Item
    noClothing             *core.Clothing
    definedStims           map[string]ParametrizedStimuliRecord
    definedEffects         map[string]EffectRecord
    definedTrigger         map[string]ParametrizedTriggerRecord
    definedReactionTrigger map[string]ParametrizedTriggerRecord
}

func (e *ExternalData) GroundTile() gridmap.Tile {
    return *e.defaultFloor
}

func (e *ExternalData) WallTile() gridmap.Tile {
    return *e.defaultFloor
}

func (e *ExternalData) NoClothing() core.Clothing {
    return *e.noClothing
}

func (e *ExternalData) NewEmptyCell() *gridmap.MapCell[*core.Actor, *core.Item, Object] {
    return &gridmap.MapCell[*core.Actor, *core.Item, Object]{
        TileType:      e.GroundTile(),
        IsExplored:    false,
        Stimuli:       nil,
        BakedLighting: common.RGBAColor{},
    }
}

func (e *ExternalData) LoadCoreData(files DataSource) {
    e.tiles = e.LoadHardCodedTiles()
    e.clothing = e.LoadHardCodedClothing()

    coreDir := path.Join("datafiles", "core")
    baseDir := path.Join(coreDir, "base")

    e.loadCoreDataFromDirectory(files, baseDir) // load base data first

    e.setDefaultGear()
    e.setPredefinedUnlocks()

    entries, err := files.ReadDir(coreDir)
    if err != nil {
        println(err.Error())
        return
    }
    for _, entry := range entries {
        if !entry.IsDir() || path.Base(entry.Name()) == "base" {
            continue
        } else {
            println("Loading " + entry.Name())
        }

        dataFilesSubDir := path.Join(coreDir, entry.Name())
        e.loadCoreDataFromDirectory(files, dataFilesSubDir)
    }

}

func (e *ExternalData) setDefaultGear() {
    e.defaultWeapon = e.items[0]
    e.defaultItem = e.items[1]
}

func (e *ExternalData) setPredefinedUnlocks() {
    e.ItemUnlockMap = map[string][]*core.Item{
        "codename 47": {e.items[0], e.items[1], e.items[2], e.items[3], e.items[4]},
    }
}

func (e *ExternalData) loadCoreDataFromDirectory(files DataSource, dataFilesSubDir string) {
    e.definedStims = merge(e.definedStims, e.LoadCustomStimuli(files, dataFilesSubDir))
    e.definedEffects = merge(e.definedEffects, e.LoadCustomEffects(files, dataFilesSubDir))
    e.definedTrigger = merge(e.definedTrigger, e.LoadCustomTriggers(files, dataFilesSubDir))
    e.definedReactionTrigger = merge(e.definedReactionTrigger, e.LoadCustomReactionTriggers(files, dataFilesSubDir))
    e.items = append(e.items, e.LoadListOfCustomItems(files, dataFilesSubDir)...)
    e.tiles = append(e.tiles, e.LoadListOfCustomTiles(files, dataFilesSubDir)...)
    e.clothing = append(e.clothing, e.LoadListOfCustomClothing(files, dataFilesSubDir)...)
}

func (e *ExternalData) LoadCustomReactionTriggers(files DataSource, dataDir string) map[string]ParametrizedTriggerRecord {
    triggerDataFileName := path.Join(dataDir, "reaction_triggers.txt")
    file, err := files.Open(triggerDataFileName)
    if err != nil {
        println(fmt.Sprintf("Could not open custom reaction trigger file %s: %s", triggerDataFileName, err.Error()))
        return map[string]ParametrizedTriggerRecord{}
    }
    defer file.Close()

    records := rec_files.Read(file)
    result := make(map[string]ParametrizedTriggerRecord)
    for _, record := range records {
        trigger := NewTriggerFromRecord(record.ToMap())
        result[trigger.Name()] = trigger
    }
    println(fmt.Sprintf("Loaded %d custom reaction triggers from %s", len(result), triggerDataFileName))
    return result
}

func merge[T any](a, b map[string]T) map[string]T {
    if a == nil {
        return b
    }
    for k, v := range b {
        a[k] = v
    }
    return a
}

func (e *ExternalData) LoadNamedStimuli() []stimuli.Stimulus {
    return []stimuli.Stimulus{
        stimuli.Stim{StimType: stimuli.StimulusFire, StimForce: 10},
    }
}

/*
Effect: BulletHit(POWER)
Stimuli: BulletStimuli(POWER)
Distribution: Blast
Size: 10
DestroyOnApplication: true
*/

/*
TriggerOnRangedShotHit: BulletHit(POWER),
TriggerOnFlightpath:    {Stimuli: defaultBulletStimuli(power)},
TriggerOnMeleeAttack:   {Stimuli: defaultBluntWeaponStimuli(power)},
*/
type ParametrizedTriggerRecord struct {
    Signature string
    EffectMap map[string]string
}

func (p ParametrizedTriggerRecord) Name() string {
    name, _ := core.GetNameAndArgs(p.Signature)
    return name
}

func (p ParametrizedTriggerRecord) Args() []string {
    _, args := core.GetNameAndArgs(p.Signature)
    return args
}
func resolveValue(value string, params map[string]int) int {
    if strings.HasPrefix(value, "$") {
        if v, ok := params[value]; ok {
            return v
        }
    }
    if v, err := strconv.Atoi(value); err == nil {
        return v
    }
    return 0
}

func indexOf(needle []string, haystack string) int {
    for i, v := range needle {
        if v == haystack {
            return i
        }
    }
    return -1
}

func (p ParametrizedTriggerRecord) ToTriggers(context *EvalContext) map[core.ItemEffectTrigger]stimuli.StimEffect {
    effects := map[core.ItemEffectTrigger]stimuli.StimEffect{}
    for triggerName, effectCall := range p.EffectMap {
        trigger := core.NewItemEffectTriggerFromString(triggerName)
        if trigger == core.NoTrigger {
            println("Unknown trigger: " + triggerName)
            continue
        }
        effects[trigger] = resolveEffect(effectCall, context)
    }
    return effects
}

func (p ParametrizedTriggerRecord) ToReactionTriggers(context *EvalContext) map[stimuli.StimulusType]core.StimReaction {
    reaction := map[stimuli.StimulusType]core.StimReaction{}
    for stimName, thresholdAndEffectCall := range p.EffectMap {
        stimType := stimuli.StimulusType(stimName)
        if stimType == stimuli.StimulusNone {
            println("Unknown stimType: " + stimName)
            continue
        }
        // 30, EffectCall(paramOne, paramTwo)
        pattern := regexp.MustCompile(`(\d+),\s([A-Za-z0-9_]+(?:\(.*\))?)`)
        matches := pattern.FindStringSubmatch(thresholdAndEffectCall)
        if len(matches) != 3 {
            println("Invalid reaction trigger: " + thresholdAndEffectCall)
            continue
        }
        threshold, _ := strconv.Atoi(matches[1])
        effectCall := matches[2]
        reaction[stimType] = core.StimReaction{
            ForceThreshold:   threshold,
            EffectOnReaction: resolveEffect(effectCall, context),
        }
    }
    return reaction
}

func (p ParametrizedTriggerRecord) Evaluate(context *EvalContext, args ...int) map[core.ItemEffectTrigger]stimuli.StimEffect {
    argMap := make(map[string]int)
    parameters := p.Args()
    for i, arg := range args {
        if i >= len(parameters) {
            println(fmt.Sprintf("Too many arguments for '%s'", p.Signature))
        }
        argMap[parameters[i]] = arg
    }
    context.ArgMap = argMap
    return p.ToTriggers(context)
}

func (p ParametrizedTriggerRecord) EvaluateReaction(context *EvalContext, args ...int) map[stimuli.StimulusType]core.StimReaction {
    argMap := make(map[string]int)
    parameters := p.Args()
    for i, arg := range args {
        if i >= len(parameters) {
            println(fmt.Sprintf("Too many arguments for '%s'", p.Signature))
        }
        argMap[parameters[i]] = arg
    }
    context.ArgMap = argMap
    return p.ToReactionTriggers(context)
}

func NewTriggerFromRecord(record map[string]string) ParametrizedTriggerRecord {
    signature := record["Signature"]
    delete(record, "Signature")
    return ParametrizedTriggerRecord{
        Signature: signature,
        EffectMap: record,
    }
}

func resolveEffect(call string, context *EvalContext) stimuli.StimEffect {
    effectName, callArgs := core.GetNameAndArgs(call)
    // we need the effect record for this name now..
    effect := context.DefinedEffects[effectName]
    argNames := effect.Args()
    for i, arg := range callArgs {
        context.ArgMap[argNames[i]] = resolveValue(arg, context.ArgMap)
    }
    // then we have to lookup the stimuli record for the effect record
    return effect.Evaluate(context)
}

type EffectRecord struct {
    Signature            string
    Stimuli              string
    Distribution         string
    Distance             string
    Pressure             string
    DestroyOnApplication bool
}

func (r EffectRecord) Name() string {
    name, _ := core.GetNameAndArgs(r.Signature)
    return name
}

func (r EffectRecord) Args() []string {
    _, args := core.GetNameAndArgs(r.Signature)
    return args
}

func (r EffectRecord) Evaluate(context *EvalContext) stimuli.StimEffect {
    stimName, _ := core.GetNameAndArgs(r.Stimuli)
    effect := stimuli.StimEffect{
        Stimuli:              context.DefinedStimuli[stimName].ToStimuli(context),
        Distribution:         stimuli.NewMethodOfDistributionFromString(r.Distribution),
        Distance:             resolveValue(r.Distance, context.ArgMap),
        Pressure:             resolveValue(r.Pressure, context.ArgMap),
        DestroyOnApplication: r.DestroyOnApplication,
    }
    return effect
}

func NewEffectFromRecord(record map[string]string) EffectRecord {
    return EffectRecord{
        Signature:            valueOrEmpty(record, "Signature"),
        Stimuli:              valueOrEmpty(record, "Stimuli"),
        Distribution:         valueOrEmpty(record, "Distribution"),
        Distance:             valueOrEmpty(record, "Distance"),
        Pressure:             valueOrEmpty(record, "Pressure"),
        DestroyOnApplication: valueOrEmpty(record, "DestroyOnApplication") == "true",
    }
}

func valueOrEmpty(record map[string]string, s string) string {
    if v, ok := record[s]; ok {
        return v
    }
    return ""
}

type ParametrizedStimuliRecord struct {
    Signature string
    // StimMap is a map of stimuli type to stimuli force.
    // eg: "FIRE: 10, PIERCING: $POWER"
    StimMap map[string]string
}

func NewStimuliFromRecord(record map[string]string) ParametrizedStimuliRecord {
    signature := record["Signature"]
    delete(record, "Signature")
    return NewParametrizedStimuliRecord(signature, record)
}

func NewParametrizedStimuliRecord(signature string, record map[string]string) ParametrizedStimuliRecord {
    return ParametrizedStimuliRecord{
        Signature: signature,
        StimMap:   record,
    }
}
func (p ParametrizedStimuliRecord) Name() string {
    name, _ := core.GetNameAndArgs(p.Signature)
    return name
}

func (p ParametrizedStimuliRecord) Args() []string {
    _, args := core.GetNameAndArgs(p.Signature)
    return args
}

// ToStimuli converts a ParametrizedStimuliRecord to a list of stimuli.Stimulus.
// The conversion is done by replacing the parameters in the StimMap with the values
// from the params map.
// eg: "FIRE: 10, PIERCING: $POWER" with params["$POWER"] = 5 becomes
// "FIRE: 10, PIERCING: 5"
func (p ParametrizedStimuliRecord) ToStimuli(context *EvalContext) []stimuli.Stimulus {
    var result []stimuli.Stimulus
    for k, v := range p.StimMap {
        stimType := stimuli.StimulusType(k)
        stimValue := resolveValue(v, context.ArgMap)
        result = append(result, stimuli.Stim{StimType: stimType, StimForce: stimValue})
    }
    return result
}

func (p ParametrizedStimuliRecord) ToRecord() []rec_files.Field {
    var result []rec_files.Field
    result = append(result, rec_files.Field{Name: "Signature", Value: p.Signature})
    for k, v := range p.StimMap {
        result = append(result, rec_files.Field{Name: k, Value: v})
    }
    return result
}

func (e *ExternalData) LoadCustomStimuli(files DataSource, dataDir string) map[string]ParametrizedStimuliRecord {
    stimDataFileName := path.Join(dataDir, "stims.txt")
    file, err := files.Open(stimDataFileName)
    if err != nil {
        println(fmt.Sprintf("Could not open custom stim file %s: %s", stimDataFileName, err.Error()))
        return map[string]ParametrizedStimuliRecord{}
    }
    defer file.Close()

    records := rec_files.Read(file)
    result := make(map[string]ParametrizedStimuliRecord)
    for _, record := range records {
        stim := NewStimuliFromRecord(record.ToMap())
        result[stim.Name()] = stim
    }
    println(fmt.Sprintf("Loaded %d custom stimuli from %s", len(result), stimDataFileName))
    return result
}
func (e *ExternalData) LoadCustomEffects(files DataSource, dataDir string) map[string]EffectRecord {
    effectDataFileName := path.Join(dataDir, "effects.txt")
    file, err := files.Open(effectDataFileName)
    if err != nil {
        println(fmt.Sprintf("Could not open custom effect file %s: %s", effectDataFileName, err.Error()))
        return map[string]EffectRecord{}
    }
    defer file.Close()

    records := rec_files.Read(file)
    result := make(map[string]EffectRecord)
    for _, record := range records {
        effect := NewEffectFromRecord(record.ToMap())
        result[effect.Name()] = effect
    }
    println(fmt.Sprintf("Loaded %d custom effects from %s", len(result), effectDataFileName))
    return result
}
func (e *ExternalData) LoadCustomTriggers(files DataSource, dataDir string) map[string]ParametrizedTriggerRecord {
    triggerDataFileName := path.Join(dataDir, "triggers.txt")
    file, err := files.Open(triggerDataFileName)
    if err != nil {
        println(fmt.Sprintf("Could not open custom trigger file %s: %s", triggerDataFileName, err.Error()))
        return map[string]ParametrizedTriggerRecord{}
    }
    defer file.Close()

    records := rec_files.Read(file)
    result := make(map[string]ParametrizedTriggerRecord)
    for _, record := range records {
        trigger := NewTriggerFromRecord(record.ToMap())
        result[trigger.Name()] = trigger
    }
    println(fmt.Sprintf("Loaded %d custom triggers from %s", len(result), triggerDataFileName))
    return result
}

type EvalContext struct {
    ArgMap                 map[string]int
    DefinedEffects         map[string]EffectRecord
    DefinedStimuli         map[string]ParametrizedStimuliRecord
    DefinedTrigger         map[string]ParametrizedTriggerRecord
    DefinedReactionTrigger map[string]ParametrizedTriggerRecord
}

func NewEvalContext(definedEffects map[string]EffectRecord, definedStimuli map[string]ParametrizedStimuliRecord, definedTrigger map[string]ParametrizedTriggerRecord, reactionTrigger map[string]ParametrizedTriggerRecord) *EvalContext {
    return &EvalContext{
        ArgMap:                 map[string]int{},
        DefinedEffects:         definedEffects,
        DefinedStimuli:         definedStimuli,
        DefinedTrigger:         definedTrigger,
        DefinedReactionTrigger: reactionTrigger,
    }
}
func (e *ExternalData) LoadListOfCustomItems(files DataSource, dataDir string) []*core.Item {

    definedItems := make([]*core.Item, 0)
    evalContext := NewEvalContext(e.definedEffects, e.definedStims, e.definedTrigger, e.definedReactionTrigger)

    itemFileName := path.Join(dataDir, "items.txt")
    file, err := files.Open(itemFileName)
    if err != nil {
        println(fmt.Sprintf("Could not open custom item file %s: %s", itemFileName, err.Error()))
        return definedItems
    }
    defer file.Close()

    records := rec_files.Read(file)
    for _, record := range records {
        item := NewItemFromRecord(record.ToMap(), evalContext)
        definedItems = append(definedItems, item)
    }
    /* TODO: Reaction Effects
       {Name: "a can of gasoline", DefinedIcon: GlyphGasCan, RangedAttack: ActionTypeThrow, Type: ItemTypeCommon, MeleeAttack: ActionTypeMeleeAttack, Uses: UnlimitedUses, TriggerEffects: gasolineEffect,
       	ReactionEffects: map[stimuli.StimulusType]StimReaction{
       		stimuli.StimulusFire:           {ForceThreshold: 30, EffectOnReaction: stimuli.EffectExplosion(30, 10, 2)},
       		stimuli.StimulusPiercingDamage: {EffectOnReaction: stimuli.EffectLeak(stimuli.StimulusBurnableLiquid, 10, 4)},
       	},
       },
       {Name: "a gas canister", DefinedIcon: GlyphGasCanister, RangedAttack: ActionTypeThrow, Type: ItemTypeCommon, MeleeAttack: ActionTypeMeleeAttack, Uses: UnlimitedUses, TriggerEffects: bluntEffects(35),
       	ReactionEffects: map[stimuli.StimulusType]StimReaction{
       		stimuli.StimulusFire:           {ForceThreshold: 30, EffectOnReaction: stimuli.EffectExplosion(30, 30, 4)},
       		stimuli.StimulusPiercingDamage: {ForceThreshold: 30, EffectOnReaction: stimuli.EffectExplosion(30, 30, 4)},
       	},
       },
    */
    // sanity check
    for i := 0; i < len(definedItems); i++ {
        definedItems[i] = e.SanitizeItem(definedItems[i])
        //		records[i] = itemList[i].ToRecord()

    }
    //	rec_files.Write(file, records)
    println(fmt.Sprintf("Loaded %d custom items from %s", len(definedItems), itemFileName))

    return definedItems
}
func (e *ExternalData) LoadHardCodedTiles() []*gridmap.Tile {
    groundBG := common.RGBAColor{R: 17 / 255.0, G: 33 / 255.0, B: 219 / 255.0, A: 1.0}
    monoFG := common.RGBAColor{R: 196 / 255.0, G: 197 / 255.0, B: 215 / 255.0, A: 1.0}
    brightFG := common.RGBAColor{R: 252 / 255.0, G: 252 / 255.0, B: 255 / 255.0, A: 1.0}

    mapStyle := common.Style{Foreground: monoFG, Background: groundBG}
    brighterStyle := common.Style{Foreground: brightFG, Background: groundBG}

    tileList := []*gridmap.Tile{

        {
            DefinedIcon:        '¢',
            DefinedDescription: "a brick wall",
            IsWalkable:         false,
            IsTransparent:      false,
            DefinedStyle:       brighterStyle.Reversed(),
        },
        {
            DefinedIcon:        '˚',
            DefinedDescription: "an exit",
            IsWalkable:         true,
            IsTransparent:      true,
            Special:            gridmap.SpecialTilePlayerExit,
            DefinedStyle:       mapStyle,
        },
        {
            DefinedIcon:        core.GlyphGround,
            DefinedDescription: "ground",
            IsWalkable:         true,
            IsTransparent:      true,
            Special:            gridmap.SpecialTileDefaultFloor,
            DefinedStyle:       mapStyle,
        },
        {
            DefinedIcon:        '¡',
            DefinedDescription: "player spawn",
            IsWalkable:         true,
            IsTransparent:      true,
            Special:            gridmap.SpecialTilePlayerSpawn,
            DefinedStyle:       mapStyle,
        },
    }
    e.defaultFloor = tileList[2]
    return tileList
}
func (e *ExternalData) LoadListOfCustomTiles(files DataSource, dataDir string) []*gridmap.Tile {

    tileDataFilename := filepath.Join(dataDir, "tiles.txt")
    var tileList []*gridmap.Tile

    file, err := files.Open(tileDataFilename)
    if err != nil {
        println("Could not open custom tile data file: ", err.Error())
        return tileList
    }
    defer file.Close()

    tileCounter := 0
    records := rec_files.Read(file)
    for _, record := range records {
        tileList = append(tileList, gridmap.NewTileFromRecord(record.ToMap()))
        tileCounter++
    }
    println(fmt.Sprintf("Loaded %d custom tiles from %s", tileCounter, tileDataFilename))

    return tileList
}

func (e *ExternalData) LoadHardCodedClothing() []*core.Clothing {
    clothes := []*core.Clothing{
        {
            Name:    "naked",
            FgColor: common.RGBAColor{R: 195, G: 149, B: 130, A: 1.0}.ToHSV(),
            BgColor: common.RGBAColor{R: 240, G: 184, B: 160, A: 1.0}.ToHSV(),
        },
        {
            Name:    "47's Signature Suit",
            FgColor: common.NewHSVColor(346/360.0, 0.97, 0.50),
            BgColor: common.NewHSVColor(0, 0, 0),
        },
        {
            Name:    "a jacket",
            FgColor: common.NewHSVColor(25/360.0, 0.7, 0.9),
            BgColor: common.NewHSVColor(0, 0, 0),
        },
    }
    e.noClothing = clothes[0]
    e.defaultPlayerClothing = clothes[1]
    e.defaultClothing = clothes[2]
    return clothes
}
func (e *ExternalData) LoadListOfCustomClothing(files DataSource, dataDir string) []*core.Clothing {
    clothingDataFilename := filepath.Join(dataDir, "clothing.txt")
    var clothes []*core.Clothing

    file, err := files.Open(clothingDataFilename)
    if err != nil {
        println("Could not open custom clothing data file: " + err.Error())
        return clothes
    }
    defer file.Close()
    loadedCounter := 0
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if len(line) == 0 {
            continue
        }
        clothesFromLine := core.NewClothingFromString(line)
        clothes = append(clothes, clothesFromLine)
        loadedCounter++
    }
    println(fmt.Sprintf("Loaded %d custom clothing items from %s", loadedCounter, clothingDataFilename))
    return clothes
}

func (e *ExternalData) HasItemUnlock(command string) bool {
    _, ok := e.ItemUnlockMap[command]
    return ok
}

func (e *ExternalData) SanitizeItem(item *core.Item) *core.Item {
    if item.DelayBetweenShotsInSecs < 0.067 {
        item.DelayBetweenShotsInSecs = 0.067
    }
    item.DefinedStyle = common.Style{Foreground: common.HSVColor{
        H: 242.0 / 360.0,
        S: 0.07,
        V: 0.06,
    }, Background: common.Transparent}
    return item
}

func (e *ExternalData) DefaultPlayerClothing() core.Clothing {
    return *e.defaultPlayerClothing
}

func (e *ExternalData) DefaultWeapon() *core.Item {
    return e.defaultWeapon
}

func (e *ExternalData) DefaultItem() *core.Item {
    return e.defaultItem
}
func (e *ExternalData) Clothing() []*core.Clothing {
    return e.clothing
}
func (e *ExternalData) Items() []*core.Item {
    return e.items
}
func (e *ExternalData) Tiles() []*gridmap.Tile {
    return e.tiles
}

func (e *ExternalData) DefaultClothing() core.Clothing {
    return *e.defaultClothing
}
func (e *ExternalData) TileFromIcon(icon rune) gridmap.Tile {
    for _, tile := range e.tiles {
        if tile.DefinedIcon == icon {
            return *tile
        }
    }
    return *e.defaultFloor
}
func (e *ExternalData) NameToClothing(name string) core.Clothing {
    for _, clothing := range e.clothing {
        if clothing.Name == name {
            return *clothing
        }
    }
    return *e.noClothing
}

func (e *ExternalData) NewActorFromDisk(factory *ItemFactory, diskData core.ActorOnDisk) *core.Actor {
    newActor := &core.Actor{
        Name:           diskData.Name,
        Type:           diskData.ActorType,
        MapPos:         diskData.Position,
        LastPos:        diskData.Position,
        Clothes:        e.ClothingFromName(diskData.Clothing),
        MovementMode:   core.MovementModeWalking,
        AutoMoveSpeed:  diskData.MoveSpeed,
        FoVinDegrees:   diskData.FoVinDegrees,
        MaxVisionRange: diskData.VisionRange,
        LookDirection:  diskData.LookDirection,
        Path:           make([]geometry.Point, 0),
    }
    newActor.Fov = geometry.NewFOV(geometry.NewRect(-newActor.MaxVisionRange, -newActor.MaxVisionRange, newActor.MaxVisionRange+1, newActor.MaxVisionRange+1))
    newActor.AI = core.NewEmptyAIComponent()
    newActor.Script = &core.ScriptComponent{}
    newActor.Inventory = newInventory(newActor, factory.StringsToItems(diskData.Inventory))
    newActor.Dialogue = &core.DialogueComponent{Conversations: make(map[string]*core.Conversation), SpokenSpeech: mapset.NewSet[string](), HeardSpeech: mapset.NewSet[string]()}
    return newActor
}

func newInventory(actor *core.Actor, items []*core.Item) *core.InventoryComponent {
    for _, item := range items {
        item.HeldBy = actor
    }
    return &core.InventoryComponent{
        Items: items,
    }
}

func (e *ExternalData) ClothingFromName(clothing string) core.Clothing {
    if clothing == "" {
        return *e.defaultClothing
    }
    for _, c := range e.Clothing() {
        if c.Name == clothing {
            return *c
        }
    }
    return *e.defaultClothing
}

func EncodeItems(inventory *core.InventoryComponent) []string {
    names := make([]string, len(inventory.Items))
    for i, item := range inventory.Items {
        names[i] = EncodeItemAsString(item)
    }
    return names
}

func NewItemFromRecord(record map[string]string, context *EvalContext) *core.Item {
    item := &core.Item{}
    item.Name = record["name"]
    runes := []rune(record["icon"])
    item.DefinedIcon = runes[0]
    item.Type = core.NewItemTypeFromString(record["type"])
    item.IsBig = record["is_big"] == "true"
    item.Uses, _ = strconv.Atoi(record["uses"])
    item.MeleeAttack = core.NewItemActionTypeFromString(record["melee_attack"])
    item.RangedAttack = core.NewItemActionTypeFromString(record["ranged_attack"])
    item.SelfUse = core.NewItemActionTypeFromString(record["self_use"])
    item.ProjectileRange, _ = strconv.Atoi(record["projectile_range"])
    item.SpreadInDegrees, _ = strconv.Atoi(record["spread_in_degrees"])
    item.ProjectileCount = uint8(core.MustParseInt(record["projectile_count"]))
    item.DelayBetweenShotsInSecs, _ = strconv.ParseFloat(record["delay_between_shots_in_secs"], 64)
    item.NoiseRadius, _ = strconv.Atoi(record["noise_radius"])
    item.AudioCue = record["audio_cue"]
    item.IsSilenced = record["is_silenced"] == "true"
    item.SilencedCue = record["silenced_cue"]
    item.KeyString = record["key_string"]
    item.LootValue, _ = strconv.Atoi(record["loot_value"])
    item.Scope.Range, _ = strconv.Atoi(record["scope_range"])
    item.Scope.FoVinDegrees, _ = strconv.ParseFloat(record["scope_fov"], 64)
    item.DefinedStyle.Foreground = common.NewColorFromString(record["style_fg"])
    item.DefinedStyle.Background = common.NewColorFromString(record["style_bg"])

    // Evaluate TriggerEffects
    name, callArgs := core.GetNameAndArgs(record["trigger_effects"])
    if trigger, ok := context.DefinedTrigger[name]; ok {
        intArgs := make([]int, len(callArgs))
        for i, arg := range callArgs {
            intArgs[i] = core.MustParseInt(arg)
        }
        item.TriggerEffects = trigger.Evaluate(context, intArgs...)
    }
    // Evaluate ReactionEffects
    name, callArgs = core.GetNameAndArgs(record["reaction_effects"])
    if reaction, ok := context.DefinedReactionTrigger[name]; ok {
        intArgs := make([]int, len(callArgs))
        for i, arg := range callArgs {
            intArgs[i] = core.MustParseInt(arg)
        }
        item.ReactionEffects = reaction.EvaluateReaction(context, intArgs...)
    }
    return item
}
