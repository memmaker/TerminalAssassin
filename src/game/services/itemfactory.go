package services

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"path"
	"regexp"
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
		Name:         "piece of paper",
		Type:         core.ItemTypeCommon,
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
			textfile := path.Join(mapFolder, "texts", pieceOfPaperItem.KeyString)
			files := engine.GetFiles()
			paperCache = files.LoadTextFile(textfile)
		}
		userInterface := engine.GetUI()
		userInterface.ShowAlert(paperCache)
	}
	pieceOfPaperItem.InsteadOfUse = insteadOfUse
	return pieceOfPaperItem
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
	for _, item := range externalData.Items() {
		if item.Name == name {
			return *item
		}
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
