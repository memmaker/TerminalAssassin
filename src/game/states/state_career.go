package states

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/utils"
	"sort"
	"strconv"
	"time"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/mapset"
)

type GameStateCareerViewer struct {
	engine  services.Engine
	isDirty bool
}

func (g *GameStateCareerViewer) ClearOverlay() {

}

func (g *GameStateCareerViewer) SetDirty() {
	g.isDirty = true
}

func (g *GameStateCareerViewer) Print(text string) {

}

func (g *GameStateCareerViewer) UpdateHUD() {

}

func (g *GameStateCareerViewer) Update(input services.InputInterface) {

}

func (g *GameStateCareerViewer) Init(engine services.Engine) {
	g.engine = engine
	userInterface := g.engine.GetUI()
	userInterface.ShowPager("Career Data", g.renderCareerData(), func() {
		g.engine.GetGame().PushState(&GameStateMainMenu{})
	})
}

func (g *GameStateCareerViewer) Draw(con console.CellInterface) {
	//g.pager.DrawOrigin(con)
}

func (g *GameStateCareerViewer) renderCareerData() []core.StyledText {
	career := g.engine.GetCareer()
	totalBodyCount := 0
	totalPlayTime := time.Duration(0)
	lines := make([]core.StyledText, 0)
	lines = append(lines, core.Text("Agent: "+career.PlayerName))
	lines = append(lines, core.Text("XP   : "+strconv.FormatUint(career.ExperiencePoints, 10)))
	lines = append(lines, core.Text("Money: $"+strconv.FormatUint(career.Money, 10)))
	lines = append(lines, core.Text("Level: "+strconv.FormatUint(career.Level(), 10)))
	lines = append(lines, core.Text("Title: "+career.Title()))

	if len(career.UnlockedItems) > 0 {
		lines = append(lines, core.Text("Available Items:"))
		unlockedItems := career.UnlockedItems.ToSlice()
		sort.SliceStable(unlockedItems, utils.NewStringComparer())
		for _, name := range unlockedItems {
			lines = append(lines, core.Text("  "+name))
		}
	}

	if len(career.UnlockedClothes) > 0 {
		lines = append(lines, core.Text("Available Clothes:"))
		unlockedClothes := toSlice(career.UnlockedClothes)
		sort.SliceStable(unlockedClothes, utils.NewStringComparer())
		for _, name := range unlockedClothes {
			lines = append(lines, core.Text("  "+name))
		}
	}

	if len(career.MapStatistics) > 0 {
		lines = append(lines, core.Text("Missions:"))
	}
	for _, mapStat := range career.MapStatistics {
		totalPlayTime += mapStat.TotalDuration
		overview, totalRewards, claimedRewards := g.getChallengeOverview(mapStat.FileHash, mapStat.FileName)
		locationCompletion := float64(claimedRewards) / float64(totalRewards)
		lines = append(lines, core.Text("  "+mapStat.FileName))
		lines = append(lines, core.Text("  "+mapStat.FileHash))
		lines = append(lines, core.Text("    Location Mastery: "+strconv.FormatFloat(locationCompletion*100, 'f', 2, 64)+"%"))
		lines = append(lines, core.Text("    Finish Count: "+strconv.Itoa(mapStat.FinishCount)))
		lines = append(lines, core.Text("    Fastest Time: "+mapStat.FastestDuration.Round(time.Millisecond).String()))
		if mapStat.UnlockedLocations.Cardinality() > 0 {
			lines = append(lines, core.Text("    Unlocked Starting Locations:"))
			unlockedLocations := mapStat.UnlockedLocations.ToSlice()
			sort.SliceStable(unlockedLocations, utils.NewStringComparer())
			for _, location := range unlockedLocations {
				lines = append(lines, core.Text("      "+location))
			}
		}
		/*
			lines = append(lines, core.Text("    Completed Challenges:"))
			for _, challenge := range mapStat.CompletedChallenges {
				lines = append(lines, core.Text("      "+challenge.Signature()))
			}*/
		lines = append(lines, overview...)
		lines = append(lines, core.Text("    Kills: "+strconv.Itoa(mapStat.TotalBodyCount)))
		totalBodyCount += mapStat.TotalBodyCount
	}

	lines = append(lines, core.Text("Total Play Time: "+totalPlayTime.Round(time.Second).String()))
	lines = append(lines, core.Text("Total Body Count: "+strconv.Itoa(totalBodyCount)))
	return lines
}

func toSlice(clothes map[string]*core.Clothing) []string {
	result := make([]string, 0)
	for _, item := range clothes {
		result = append(result, item.Name)
	}
	return result
}

func ChallengeToString(challenge services.Challenge) string {
	return fmt.Sprintf("%s - %d", challenge.Name(), challenge.Reward())
}
func ChallengeToStringWithTime(challenge services.Challenge) string {
	return fmt.Sprintf("%s - %d (%s)", challenge.Name(), challenge.Reward(), challenge.CompletionTime().Round(time.Millisecond).String())
}
func (g *GameStateCareerViewer) getChallengeOverview(mapHash, mapFolder string) ([]core.StyledText, int, int) {
	result := make([]core.StyledText, 0)
	career := g.engine.GetCareer()
	totalRewards := 0
	claimedRewards := 0
	redStyle := common.Style{Foreground: common.Red, Background: common.Black}
	greenStyle := common.Style{Foreground: common.Green, Background: common.Black}

	//files := g.engine.GetFiles()
	//config := g.engine.GetGame().GetConfig()
	//campaignFolderName := config.CampaignDirectory
	mapStats := career.MapStatistics[mapHash]
	completedIDs := mapset.NewSet[int]()
	for _, challenge := range mapStats.CompletedChallenges {
		completedIDs.Add(challenge.ID())
	}

	classicChallenges := services.LoadClassicChallenges(g.engine.GetGame(), core.MissionStats{})
	mapChallenges := career.ParseMapChallenges(mapFolder, g.engine, core.MissionStats{})
	result = append(result, core.Text("    Classic Challenges"))
	for _, challenge := range classicChallenges {
		var lineOfText core.StyledText
		if completedIDs.Contains(challenge.ID()) {
			challenge = mapStats.CompletedChallenges[challenge.ID()]
			lineOfText = core.NewStyledText("      "+ChallengeToStringWithTime(challenge), greenStyle)
			claimedRewards += challenge.Reward()
		} else {
			lineOfText = core.NewStyledText("      "+ChallengeToString(challenge), redStyle)
		}
		result = append(result, lineOfText)
		totalRewards += challenge.Reward()
	}
	result = append(result, core.Text("    Mission Challenges"))
	for _, challenge := range mapChallenges {
		var lineOfText core.StyledText
		if completedIDs.Contains(challenge.ID()) {
			challenge = mapStats.CompletedChallenges[challenge.ID()]
			lineOfText = core.NewStyledText("      "+ChallengeToStringWithTime(challenge), greenStyle)
			claimedRewards += challenge.Reward()
		} else {
			lineOfText = core.NewStyledText("      "+ChallengeToString(challenge), redStyle)
		}
		result = append(result, lineOfText)
		totalRewards += challenge.Reward()
	}
	return result, totalRewards, claimedRewards
}
