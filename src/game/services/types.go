package services

import (
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/mapset"
	"math"
	"time"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/gridmap"
)

type FixedChallenge struct {
	id          int
	name        string
	reward      int
	isCompleted func() bool
	timeNeeded  time.Duration
}

func (f FixedChallenge) IsCustom() bool {
	return f.id >= 100
}

type ItemPickedUpEvent struct {
	Item  *core.Item
	Actor *core.Actor
}
type ActorEnteredZoneEvent struct {
	Actor       *core.Actor
	OldPosition geometry.Point
	NewPosition geometry.Point
	OldZone     *gridmap.ZoneInfo
	NewZone     *gridmap.ZoneInfo
}
type TriggerEvent struct {
	Key string
}

func (f FixedChallenge) WithTime(completionTime time.Duration) Challenge {
	f.timeNeeded = completionTime
	return f
}

func (f FixedChallenge) CompletionTime() time.Duration {
	return f.timeNeeded
}

func (f FixedChallenge) ID() int {
	return f.id
}

func (f FixedChallenge) Name() string {
	return f.name
}

func (f FixedChallenge) Reward() int {
	return f.reward
}

func (f FixedChallenge) IsCompleted() bool {
	return f.isCompleted()
}

type MenuItem struct {
	Label               string
	Handler             func()
	Condition           func() bool
	Icon                rune
	Highlight           func() bool
	LeftHandler         func()
	RightHandler        func()
	DynamicLabel        func() string
	QuickKey            core.Key
	IconForegroundColor common.Color
	ShiftHandler        func()
}

type MapStatistics struct {
	FileName            string
	FileHash            string
	CompletedChallenges map[int]DiskChallenge
	FinishCount         int
	TotalBodyCount      int
	TotalDuration       time.Duration
	FastestDuration     time.Duration
	UnlockedLocations   mapset.Set[string]
}

func NewMapStats(mission *gridmap.GridMap[*core.Actor, *core.Item, Object]) *MapStatistics {
	return &MapStatistics{
		FileName:            mission.MapFileName(),
		FileHash:            mission.MapHash(),
		FastestDuration:     math.MaxInt64,
		FinishCount:         0,
		TotalBodyCount:      0,
		CompletedChallenges: make(map[int]DiskChallenge, 0),
		UnlockedLocations:   mapset.NewSet[string](),
	}
}
