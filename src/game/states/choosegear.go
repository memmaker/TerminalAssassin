package states

import (
	"fmt"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
	"path"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

func (g *GameStateMainMenu) openPlanningMenu() {
	g.showGear = true
	g.isDirty = true
	userInterface := g.engine.GetUI()
	title := "Planning"
	startLocations := g.loadUnlockedStartLocations()
	menuItems := []services.MenuItem{
		{
			Label:     "Start Location",
			Handler:   g.openStartLocationMenu(startLocations),
			Condition: func() bool { return len(startLocations) > 0 },
		},
		{
			Label:   "Weapon",
			Handler: g.openWeaponMenu,
		},
		{
			Label:   "Clothes",
			Handler: g.openClothesMenu,
		},
		{
			Label:   "Gear Slot #1",
			Handler: g.openToolsMenuForSlotOne,
		},
		{
			Label:   "Gear Slot #2",
			Handler: g.openToolsMenuForSlotTwo,
		},
		{
			Label: "Back",
			Handler: func() {
				g.showGear = false
				userInterface.PopModal()
			},
		},
	}
	userInterface.OpenFixedWidthStackedMenu(title, menuItems)
}

func (g *GameStateMainMenu) openStartLocationMenu(locations []core.StartLocation) func() {
	return func() {
		menuItems := make([]services.MenuItem, 0)
		game := g.engine.GetGame()
		gear := game.GetMissionPlan()
		data := g.engine.GetData()
		userInterface := g.engine.GetUI()
		for _, startLocation := range locations {
			curLocation := startLocation
			clothes := data.NameToClothing(curLocation.Clothes)
			menuItems = append(menuItems, services.MenuItem{
				Label: curLocation.ToString(),
				Handler: func() {
					gear.SetSpecialStartLocation(curLocation.Name, curLocation.Location, &clothes)
					game.UpdateHUD()
				}})
		}
		userInterface.OpenFixedWidthAutoCloseMenu("Locations", menuItems)
	}
}

func (g *GameStateMainMenu) loadUnlockedStartLocations() []core.StartLocation {
	career := g.engine.GetCareer()
	missionMap := g.engine.GetGame().GetMap()
	mapHash := missionMap.MapHash()
	files := g.engine.GetFiles()
	mapFolder := missionMap.MapFileName()
	availableLocationsFilename := path.Join(mapFolder, "start_locations.txt")
	file, err := files.Open(availableLocationsFilename)
	if err != nil {
		return []core.StartLocation{}
	}
	records := rec_files.Read(file)
	availableLocations := make(map[string]core.StartLocation)
	for _, record := range records {
		startLoc := core.NewStartLocationFromRecord(record, missionMap.GetNamedLocation)
		availableLocations[startLoc.Name] = startLoc
	}
	println(fmt.Sprintf("Found %d available start locations", len(availableLocations)))
	result := make([]core.StartLocation, 0)
	if mapStats, ok := career.MapStatistics[mapHash]; ok {
		if mapStats.UnlockedLocations == nil {
			return result
		}
		for _, locName := range mapStats.UnlockedLocations.ToSlice() {
			if startingLocation, isUnlockedAndAvailable := availableLocations[locName]; isUnlockedAndAvailable {
				result = append(result, startingLocation)
			}
		}
	}
	return result
}

func (g *GameStateMainMenu) RenderChosenGear(con console.CellInterface) {
	m := g.engine.GetGame()
	gear := m.GetMissionPlan()
	gridHeight := g.engine.ScreenGridHeight()

	_, startLocationName := gear.Location()
	if startLocationName == "" {
		startLocationName = "(Provided by the agency)"
	}

	weaponName := "None"
	if gear.Weapon() != nil {
		weaponName = fmt.Sprintf("%s%s", string(gear.Weapon().Icon()), gear.Weapon().Name)
	}

	clothesName := "None"
	if gear.Clothes() != nil {
		clothesName = fmt.Sprintf("%s%s", string(core.GlyphClothing), gear.Clothes().Name)
	}

	gearOneName := "None"
	if gear.GearOne() != nil {
		gearOneName = fmt.Sprintf("%s%s", string(gear.GearOne().Icon()), gear.GearOne().Name)
	}

	gearTwoName := "None"
	if gear.GearTwo() != nil {
		gearTwoName = fmt.Sprintf("%s%s", string(gear.GearTwo().Icon()), gear.GearTwo().Name)
	}
	locationString := fmt.Sprintf("Location    : %s", startLocationName)
	weaponString := fmt.Sprintf("Weapon      : %s", weaponName)
	clothesString := fmt.Sprintf("Clothes     : %s", clothesName)
	gearOneString := fmt.Sprintf("Gear Slot #1: %s", gearOneName)
	gearTwoString := fmt.Sprintf("Gear Slot #2: %s", gearTwoName)

	startAt := geometry.Point{X: 7, Y: gridHeight - 3}
	g.writeText(con, locationString, startAt.Sub(geometry.Point{X: 0, Y: 8}))
	g.writeText(con, weaponString, startAt.Sub(geometry.Point{X: 0, Y: 6}))
	g.writeText(con, clothesString, startAt.Sub(geometry.Point{X: 0, Y: 4}))
	g.writeText(con, gearOneString, startAt.Sub(geometry.Point{X: 0, Y: 2}))
	g.writeText(con, gearTwoString, startAt)
}

func (g *GameStateMainMenu) writeText(con console.CellInterface, text string, pointToPlace geometry.Point) {
	for x, char := range text {
		drawPos := pointToPlace.Add(geometry.Point{X: x})
		cellAt := con.AtSquare(drawPos)
		con.SetSquare(drawPos, common.Cell{Rune: char, Style: cellAt.Style.WithFg(common.White)})
	}
}

func (g *GameStateMainMenu) openWeaponMenu() {
	menuItems := make([]services.MenuItem, 0)
	career := g.engine.GetCareer()
	game := g.engine.GetGame()
	plan := game.GetMissionPlan()
	//data := g.engine.GetData()
	userInterface := g.engine.GetUI()
	itemFactory := g.engine.GetItemFactory()
	for itemName := range career.UnlockedItems {
		item := itemFactory.DecodeStringToItem(itemName)
		if !item.IsWeapon() {
			continue
		}
		menuItems = append(menuItems, services.MenuItem{
			Label: item.Name,
			Handler: func() {
				plan.SetWeapon(&item)
				game.UpdateHUD()
			}})
	}
	userInterface.OpenFixedWidthAutoCloseMenu("Weapons", menuItems)
}

func (g *GameStateMainMenu) openClothesMenu() {
	m := g.engine.GetGame()
	userInterface := g.engine.GetUI()
	gear := m.GetMissionPlan()
	career := g.engine.GetCareer()
	menuItems := make([]services.MenuItem, 0)

	for _, c := range career.UnlockedClothes {
		clothes := c
		menuItems = append(menuItems, services.MenuItem{
			Label: clothes.Name,
			Handler: func() {
				gear.SetClothes(clothes)
				m.UpdateHUD()
			}})
	}

	userInterface.OpenFixedWidthAutoCloseMenu("Clothes", menuItems)
}

func (g *GameStateMainMenu) openToolsMenuForSlotOne() {
	m := g.engine.GetGame()
	career := g.engine.GetCareer()
	userInterface := g.engine.GetUI()
	gear := m.GetMissionPlan()
	itemFactory := g.engine.GetItemFactory()
	menuItems := make([]services.MenuItem, 0)

	for itemName := range career.UnlockedItems {
		tool := itemFactory.DecodeStringToItem(itemName)
		if tool.IsWeapon() ||
			(gear.GearTwo() != nil && tool.Name == (*(gear.GearTwo())).Name) {
			continue
		}
		menuItems = append(menuItems, services.MenuItem{
			Label: tool.Name,
			Handler: func() {
				gear.SetSlotOne(&tool)
				m.UpdateHUD()
			}})
	}

	userInterface.OpenFixedWidthAutoCloseMenu("items", menuItems)
}

func (g *GameStateMainMenu) openToolsMenuForSlotTwo() {
	m := g.engine.GetGame()
	userInterface := g.engine.GetUI()
	career := g.engine.GetCareer()
	gear := m.GetMissionPlan()
	itemFactory := g.engine.GetItemFactory()
	menuItems := make([]services.MenuItem, 0)

	for itemName := range career.UnlockedItems {
		tool := itemFactory.DecodeStringToItem(itemName)
		if tool.IsWeapon() ||
			(gear.GearOne() != nil && tool.Name == (*(gear.GearOne())).Name) {
			continue
		}
		menuItems = append(menuItems, services.MenuItem{
			Label: tool.Name,
			Handler: func() {
				gear.SetSlotTwo(&tool)
				m.UpdateHUD()
			}})
	}

	userInterface.OpenFixedWidthAutoCloseMenu("items", menuItems)
}
