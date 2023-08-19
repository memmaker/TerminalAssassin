package states

import (
	"fmt"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
	"path"
	"strconv"
	"time"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/utils"
)

type GameStateGameOver struct {
	engine                          services.Engine
	MissionExitedWithGoalCompletion bool
	debriefingMessage               []core.StyledText
	isDirty                         bool
	CauseOfPlayerDeath              core.CauseOfDeath
}

func (g *GameStateGameOver) ClearOverlay() {

}

func (g *GameStateGameOver) SetDirty() {
	g.isDirty = true
}

func (g *GameStateGameOver) Update(input services.InputInterface) {

}

func (g *GameStateGameOver) Draw(con console.CellInterface) {

}

func (g *GameStateGameOver) Init(engine services.Engine) {
	g.engine = engine
	stats := engine.GetGame().GetStats()
	stats.SecondsNeeded = utils.UTicksToSeconds(engine.CurrentTick())

	g.debriefingMessage = g.createDebriefingMessage(g.MissionExitedWithGoalCompletion)

	userInterface := g.engine.GetUI()
	onQuit := func() {
		g.engine.Reset()
	}
	userInterface.ShowPager("Mission Debriefing", g.debriefingMessage, onQuit)
}

func (g *GameStateGameOver) createDebriefingMessage(success bool) []core.StyledText {
	career := g.engine.GetCareer()
	game := g.engine.GetGame()
	stats := *game.GetStats()
	currentMap := game.GetMap()
	player := currentMap.Player
	lootItems := g.getLootFromInventory(player.Inventory)
	var challengeResults services.ChallengeResults
	var newUnlocks []Unlockable
	if success {
		career.AddMissionStats(currentMap, stats)
		challengeResults = career.CheckForChallengeCompletion(g.engine, stats)
		newUnlocks = g.checkForUnlocks(challengeResults.OldCompletionPercentage, challengeResults.NewCompletionPercentage)
		g.ApplyUnlocks(newUnlocks)
	}

	message := make([]core.StyledText, 0)

	if success {
		message = append(message, core.Text("Your career has been updated.").WithStyle(common.DefaultStyle.WithFg(common.Green)))
		message = append(message, core.Text(fmt.Sprintf("You completed the mission in %s", stats.MissionDuration().Round(time.Millisecond))))
		if len(challengeResults.NewlyCompleted) > 0 {
			message = append(message, core.Text(""))
			message = append(message, core.Text("New challenges:"))

			for _, challenge := range challengeResults.NewlyCompleted {
				challengeMessage := fmt.Sprintf("%s (%d XP) in %s", challenge.Name(), challenge.Reward(), challenge.CompletionTime().Round(time.Millisecond))
				message = append(message, core.Text(challengeMessage).WithStyle(common.DefaultStyle.WithFg(common.Green)))
			}
		}
		if len(challengeResults.FasterCompleted) > 0 {
			message = append(message, core.Text(""))
			message = append(message, core.Text("Faster challenges:"))

			for _, challenge := range challengeResults.FasterCompleted {
				challengeMessage := fmt.Sprintf("%s (%d XP) in %s", challenge.Name(), challenge.Reward(), challenge.CompletionTime().Round(time.Millisecond))
				message = append(message, core.Text(challengeMessage).WithStyle(common.DefaultStyle.WithFg(common.Green)))
			}
		}
	} else {
		message = append(message, core.Text("This mission was a FAILURE.").WithStyle(common.DefaultStyle.WithFg(common.Red)))
		message = append(message, core.Text(fmt.Sprintf("Cause of death: %s", g.CauseOfPlayerDeath.WithKiller())))
	}

	if len(newUnlocks) > 0 {
		message = append(message, core.Text(""))
		message = append(message, core.Text("New unlocks:"))
		for _, unlock := range newUnlocks {
			message = append(message, core.Text(fmt.Sprintf("%s: %s", unlock.UnlockType, unlock.Unlockable)))
		}
	}

	totalKills := 0
	if len(stats.Kills) > 0 {
		directKills := make([]core.KillStatistics, 0)
		indirectKills := make([]core.KillStatistics, 0)
		for _, kill := range stats.Kills {
			if !kill.CauseOfDeath.IsPlayer() {
				continue
			}
			directKills = append(directKills, kill)
		}
		for _, kill := range stats.Kills {
			if kill.CauseOfDeath.IsPlayer() {
				continue
			}
			indirectKills = append(indirectKills, kill)
		}
		if len(directKills) > 0 {
			message = append(message, core.Text(""))
			message = append(message, core.Text("Direct Kills:"))
			for _, kill := range directKills {
				message = append(message, core.Text(fmt.Sprintf("%s (%s) %s (%s)", kill.VictimName, kill.VictimType, kill.CauseOfDeath.WithoutKiller(), kill.KilerClothingDuringKill)))
			}
		}
		if len(indirectKills) > 0 {
			message = append(message, core.Text(""))
			message = append(message, core.Text("Indirect Kills:"))
			for _, kill := range indirectKills {
				message = append(message, core.Text(fmt.Sprintf("%s (%s) %s", kill.VictimName, kill.VictimType, kill.CauseOfDeath.WithKiller())))
			}
		}
	}
	lootSum := 0
	if success {
		if len(lootItems) > 0 {
			message = append(message, core.Text(""))
			message = append(message, core.Text("You repossessed the following valuables:"))
			for _, item := range lootItems {
				itemMessage := fmt.Sprintf("%s ($%d)", item.Name, item.LootValue)
				message = append(message, core.Text(itemMessage))
				lootSum += item.LootValue
			}
		}
		if lootSum > 0 {
			message = append(message, core.Text(""))
			message = append(message, core.Text(fmt.Sprintf("Total loot: $%d", lootSum)))
		}
	}
	totalKills = len(stats.Kills)
	if totalKills > 0 {
		message = append(message, core.Text(fmt.Sprintf("Total kills: %d", totalKills)))
	}
	if success {
		career.Money += uint64(lootSum)
		career.SaveToFile()
	}
	return message
}

func (g *GameStateGameOver) getLootFromInventory(inventory *core.InventoryComponent) []*core.Item {
	lootItems := make([]*core.Item, 0)
	for _, item := range inventory.Items {
		if item.LootValue > 0 {
			lootItems = append(lootItems, item)
		}
	}
	return lootItems

}

type Unlockable struct {
	UnlockType string
	Unlockable string
}

func (g *GameStateGameOver) checkForUnlocks(oldPercentage float64, newPercentage float64) []Unlockable {
	files := g.engine.GetFiles()
	currentMap := g.engine.GetGame().GetMap()
	mapFolder := currentMap.MapFileName()
	unlockFile := path.Join(mapFolder, "unlocks.txt")
	file, err := files.Open(unlockFile)
	if err != nil {
		return nil
	}
	records := rec_files.Read(file)
	println(fmt.Sprintf("Found %d unlockables", len(records)))
	// scale to 100
	oldPercentage *= 100
	newPercentage *= 100
	result := make([]Unlockable, 0)
	for _, record := range records {
		var percentNeeded int
		var unlockable string
		var unlockType string
		for _, field := range record {
			if field.Name == "Completion_Percent" {
				percentNeeded, _ = strconv.Atoi(field.Value)
			} else {
				unlockType = field.Name
				unlockable = field.Value
			}
		}
		if percentNeeded > int(oldPercentage) && percentNeeded <= int(newPercentage) {
			result = append(result, Unlockable{
				UnlockType: unlockType,
				Unlockable: unlockable,
			})
		}
	}
	return result
}

func (g *GameStateGameOver) ApplyUnlocks(unlocks []Unlockable) {
	career := g.engine.GetCareer()
	data := g.engine.GetData()
	for _, unlock := range unlocks {
		switch unlock.UnlockType {
		case "Start":
			currentMap := g.engine.GetGame().GetMap()
			mapHash := currentMap.MapHash()
			career.AddUnlockedLocation(mapHash, unlock.Unlockable)
		case "Clothing":
			clothingName := unlock.Unlockable
			unlockedClothes := data.NameToClothing(clothingName)
			career.UnlockedClothes[clothingName] = &unlockedClothes
		case "Item":
			career.UnlockedItems.Add(unlock.Unlockable)
		}
	}
}
