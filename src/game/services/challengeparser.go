package services

import (
	"io"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/memmaker/terminal-assassin/game/core"
)

type ChallengeParser struct {
	*core.Logic
}

func (p ChallengeParser) ChallengesFromFile(file fs.File) ([]Challenge, error) {
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return p.parse(string(b), 100), nil
}

// NOTE: Map Challenges start with ID 100

/* EXAMPLE CHALLANGE FILE CONTENTS:

# 3000 - On the spot

## AND-CONDITIONS
TargetKilledInZone(Park)

# 2000 - Gang war

## OR-CONDITIONS
TargetKilledByActorWithName(Blue Lotus Guard 1)
TargetKilledByActorWithName(Blue Lotus Guard 2)
TargetKilledByActorWithName(Blue Lotus Leader)

*/

type ChallengeReadState int

const (
	ChallengeReadStatePreamble ChallengeReadState = iota
	ChallengeReadStateAndConditions
	ChallengeReadStateOrConditions
)

type CustomChallenge struct {
	id                  int
	name                string
	reward              int
	completionCondition core.CombinedPredicate
	completionTime      time.Duration
}

func (c CustomChallenge) IsCustom() bool {
	return c.id >= 100
}

func (c CustomChallenge) CompletionTime() time.Duration {
	return c.completionTime
}

func (c CustomChallenge) WithTime(completionTime time.Duration) Challenge {
	c.completionTime = completionTime
	return c
}

func (c CustomChallenge) ID() int {
	return c.id
}

func (c CustomChallenge) Name() string {
	return c.name
}

func (c CustomChallenge) Reward() int {
	return c.reward
}

func (c CustomChallenge) IsCompleted() bool {
	return c.completionCondition.Evaluate()
}

func (p ChallengeParser) parse(contents string, firstID int) []Challenge {
	results := make([]Challenge, 0)
	challengeNamePattern := regexp.MustCompile(`^# (\d+) - (.+)$`)
	var completedCondition core.CombinedPredicate
	state := ChallengeReadStatePreamble
	lines := strings.Split(contents, "\n")
	value := 0
	name := ""

	for _, line := range lines {
		if len(line) == 0 {
			continue
		} else if line == "## AND-CONDITIONS" {
			state = ChallengeReadStateAndConditions
		} else if line == "## OR-CONDITIONS" {
			state = ChallengeReadStateOrConditions
		} else if challengeNamePattern.MatchString(line) {
			if value != 0 && name != "" {
				results = append(results, &CustomChallenge{
					id:                  firstID,
					name:                name,
					reward:              value,
					completionCondition: completedCondition,
				})
				firstID++
			}
			completedCondition = core.CombinedPredicate{}
			matches := challengeNamePattern.FindStringSubmatch(line)
			value, _ = strconv.Atoi(matches[1])
			name = matches[2]
		} else if core.LooksLikeAFunction(line) {
			if state == ChallengeReadStateAndConditions {
				completedCondition = completedCondition.And(p.LineToPredicate(line))
			} else if state == ChallengeReadStateOrConditions {
				completedCondition = completedCondition.Or(p.LineToPredicate(line))
			}
		}
	}

	if value != 0 && name != "" {
		results = append(results, &CustomChallenge{
			id:                  firstID,
			name:                name,
			reward:              value,
			completionCondition: completedCondition,
		})
	}
	return results
}

func NewChallengeParser(player *core.Actor) *ChallengeParser {
	return &ChallengeParser{Logic: core.NewLogicCore(player)}
}
