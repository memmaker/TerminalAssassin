package services

import (
	"encoding/gob"
	"math"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/mapset"
)

type CareerData struct {
	PlayerName            string
	CurrentCampaignFolder string
	ExperiencePoints      uint64
	MapStatistics         map[string]*MapStatistics
	UnlockedItems         mapset.MapSet[string]
	UnlockedClothes       map[string]*core.Clothing
	Money                 uint64
}

func (c *CareerData) Level() uint64 {
	return uint64(math.Ceil((0.0000567567568 * float64(c.ExperiencePoints)) + 0.4864864864865))
}
func (c *CareerData) Title() string {
	switch {
	case c.Level() >= 10 && c.Level() < 20:
		return "Novice Infiltrator"
	case c.Level() >= 20 && c.Level() < 30:
		return "Skilled Operative"
	case c.Level() >= 30 && c.Level() < 40:
		return "Silent killer"
	case c.Level() >= 40 && c.Level() < 50:
		return "Master Assassin"
	case c.Level() >= 50 && c.Level() < 60:
		return "Ghost"
	case c.Level() >= 60 && c.Level() < 70:
		return "Black ops"
	case c.Level() >= 70 && c.Level() < 80:
		return "Shadow warrior"
	case c.Level() >= 80 && c.Level() < 90:
		return "Elite Killer"
	case c.Level() >= 90 && c.Level() < 100:
		return "Night stalker"
	case c.Level() >= 100 && c.Level() < 500:
		return "Specter of vengeance"
	case c.Level() >= 500 && c.Level() < 1000:
		return "Malus Necessarium"
	case c.Level() >= 1000:
		return "Grim Reaper"
	default:
		return "Untrusted Agent"
	}
}
func (c *CareerData) SaveToFile() {
	fileWriter, fileErr := os.Create("career.tmp")
	if fileErr != nil {
		os.Remove("career.tmp")
		println("Error saving career data: " + fileErr.Error())
		return
	}
	encoder := gob.NewEncoder(fileWriter)
	err := encoder.Encode(c)
	if err != nil {
		os.Remove("career.tmp")
		println("Error encoding career data: " + err.Error())
		return
	}
	closeErr := fileWriter.Close()
	if closeErr != nil {
		os.Remove("career.tmp")
		println("Error closing career file: " + closeErr.Error())
		return
	}
	renameErr := os.Rename("career.tmp", "career.gob")
	if renameErr != nil {
		println("Error renaming career file: " + renameErr.Error())
		return
	}
	println("Saved career data to file.")
}

func (c *CareerData) registerChallengePredicates(parser *ChallengeParser, engine Engine, stats core.MissionStats) {
	currentMap := engine.GetGame().GetMap()
	var targetKill core.KillStatistics
	for _, killStat := range stats.Kills {
		if killStat.VictimType == core.ActorTypeTarget {
			targetKill = killStat
		}
	}
	parser.RegisterPredicate("TargetKilledInZone", func(args ...any) bool {
		zoneName := args[0].(string)
		deathZone := currentMap.ZoneAt(targetKill.AtLocation)
		return deathZone.Name == zoneName
	})

	parser.RegisterPredicate("DisguiseWorn", func(args ...any) bool {
		nameOfDisguise := args[0].(string)
		return stats.DisguisesWorn.Contains(nameOfDisguise)
	})

	parser.RegisterPredicate("TargetKilledBeforeTime", func(args ...any) bool {
		time, _ := strconv.ParseFloat(args[0].(string), 64)
		return targetKill.AtSecond < time
	})
	parser.RegisterPredicate("TargetKilledByActorWithName", func(args ...any) bool {
		actorName := args[0].(string)
		if targetKill.CauseOfDeath.Source.Actor == nil {
			return false
		}
		return targetKill.CauseOfDeath.Source.Actor.Name == actorName
	})

	parser.RegisterPredicate("KillDetails", func(args ...any) bool {
		nameOfVictim := args[0].(string)
		typeOfItem := args[1].(string)
		nameOfClothing := args[2].(string)
		for _, kill := range stats.Kills {
			if kill.CauseOfDeath.Source.Item == nil {
				continue
			}
			isMatch := kill.VictimName == nameOfVictim &&
				kill.CauseOfDeath.Source.Item.Type.ToString() == typeOfItem &&
				kill.KilerClothingDuringKill == nameOfClothing
			if isMatch {
				return true
			}
		}
		return false
	})
}
func (c *CareerData) ParseMapChallenges(mapFolder string, engine Engine, stats core.MissionStats) []Challenge {
	var mapChallenges []Challenge
	files := engine.GetFiles()
	currentMap := engine.GetGame().GetMap()
	challengeFilename := path.Join(mapFolder, "challenges.txt")
	if !files.FileExists(challengeFilename) {
		return mapChallenges
	}
	parser := NewChallengeParser(currentMap.Player)
	c.registerChallengePredicates(parser, engine, stats)

	challengeFile, err := files.Open(challengeFilename)
	if err != nil {
		println("Error opening challenge file: " + challengeFilename)
		return mapChallenges
	}
	mapChallenges, err = parser.ChallengesFromFile(challengeFile)
	if err != nil {
		println("Error parsing challenge file: " + challengeFilename)
		return mapChallenges
	}
	return mapChallenges
}

type ChallengeResults struct {
	NewlyCompleted          []Challenge
	FasterCompleted         []Challenge
	NewCompletionPercentage float64
	OldCompletionPercentage float64
}

func (c *CareerData) CheckForChallengeCompletion(engine Engine, stats core.MissionStats) ChallengeResults {
	completionTime := stats.MissionDuration()
	game := engine.GetGame()
	completedChallenges := make([]*DiskChallenge, 0)
	newlyCompletedChallenges := make([]Challenge, 0)
	fasterCompletedChallenges := make([]Challenge, 0)
	missionMap := game.GetMap()
	mapHash := missionMap.MapHash()
	if _, ok := c.MapStatistics[mapHash]; !ok {
		c.MapStatistics[mapHash] = NewMapStats(missionMap)
	}
	challengeCount := 0
	classicChallenges := LoadClassicChallenges(game, stats)
	for _, challenge := range classicChallenges {
		if challenge.IsCompleted() {
			completedChallenges = append(completedChallenges, NewDiskChallenge(challenge, completionTime))
		}
	}
	challengeCount += len(classicChallenges)

	mapChallenges := c.ParseMapChallenges(missionMap.MapFileName(), engine, stats)
	for _, challenge := range mapChallenges {
		if challenge.IsCompleted() {
			completedChallenges = append(completedChallenges, NewDiskChallenge(challenge, completionTime))
		}
	}
	challengeCount += len(mapChallenges)
	oldCompletedCount := len(c.MapStatistics[mapHash].CompletedChallenges)
	for _, challenge := range completedChallenges {
		if existingChallengeCompletion, ok := c.MapStatistics[mapHash].CompletedChallenges[challenge.ID()]; ok {
			if existingChallengeCompletion.CompletionTime() > challenge.CompletionTime() {
				c.MapStatistics[mapHash].CompletedChallenges[challenge.ID()] = *challenge
				fasterCompletedChallenges = append(fasterCompletedChallenges, challenge)
			}
		} else {
			c.MapStatistics[mapHash].CompletedChallenges[challenge.ID()] = *challenge
			newlyCompletedChallenges = append(newlyCompletedChallenges, challenge)
			c.ExperiencePoints += uint64(challenge.Reward())
		}
	}
	//completion percentage..
	newCompletedCount := len(c.MapStatistics[mapHash].CompletedChallenges)
	newCompletionPercentage := float64(newCompletedCount) / float64(challengeCount)
	oldCompletionPercentage := float64(oldCompletedCount) / float64(challengeCount)

	return ChallengeResults{
		NewlyCompleted:          newlyCompletedChallenges,
		FasterCompleted:         fasterCompletedChallenges,
		OldCompletionPercentage: oldCompletionPercentage,
		NewCompletionPercentage: newCompletionPercentage,
	}
}

type DiskChallenge struct {
	Identifier      int
	ChallengeName   string
	ChallengeReward int
	FastestTime     time.Duration
}

func (d DiskChallenge) IsCustom() bool {
	return d.Identifier >= 100
}

func (d DiskChallenge) WithTime(completionTime time.Duration) Challenge {
	if completionTime < d.FastestTime {
		d.FastestTime = completionTime
	}
	return d
}

func (d DiskChallenge) CompletionTime() time.Duration {
	return d.FastestTime
}

func (d DiskChallenge) ID() int {
	return d.Identifier
}

func (d DiskChallenge) Name() string {
	return d.ChallengeName
}

func (d DiskChallenge) Reward() int {
	return d.ChallengeReward
}

func (d DiskChallenge) IsCompleted() bool {
	return true
}

func NewDiskChallenge(challenge Challenge, completionTime time.Duration) *DiskChallenge {
	return &DiskChallenge{
		Identifier:      challenge.ID(),
		ChallengeName:   challenge.Name(),
		ChallengeReward: challenge.Reward(),
		FastestTime:     completionTime,
	}
}

func (c *CareerData) AddMissionStats(missionMap *gridmap.GridMap[*core.Actor, *core.Item, Object], stats core.MissionStats) {
	mapHash := missionMap.MapHash()
	if _, ok := c.MapStatistics[mapHash]; !ok {
		c.MapStatistics[mapHash] = NewMapStats(missionMap)
	}
	c.MapStatistics[mapHash].TotalBodyCount += len(stats.Kills)
	c.MapStatistics[mapHash].FinishCount++
	c.MapStatistics[mapHash].TotalDuration += stats.MissionDuration()
	if c.MapStatistics[mapHash].FastestDuration > stats.MissionDuration() {
		c.MapStatistics[mapHash].FastestDuration = stats.MissionDuration()
	}
	if c.MapStatistics[mapHash].UnlockedLocations == nil {
		c.MapStatistics[mapHash].UnlockedLocations = mapset.NewSet[string]()
	}
}

func (c *CareerData) AddUnlockedLocation(hash string, locationName string) {
	c.MapStatistics[hash].UnlockedLocations.Add(locationName)
}
