package services

import "github.com/memmaker/terminal-assassin/game/core"

func LoadClassicChallenges(game GameInterface, stats core.MissionStats) []Challenge {
	return []Challenge{
		FixedChallenge{id: 0, name: "Piano Man", isCompleted: TargetCauseOfDeath(stats, core.CoDStrangledWithWire), reward: 1000},   // kill target with piano wire
		FixedChallenge{id: 1, name: "Tasteless, Traceless", isCompleted: TargetCauseOfDeath(stats, core.CoDPoisoned), reward: 1000}, // kill target with poison
		FixedChallenge{id: 2, name: "Hold My Hair", isCompleted: TargetCauseOfDeath(stats, core.CoDDrownedInToilet), reward: 1000},  // kill target by drowning
		FixedChallenge{id: 3, name: "Straight Shot", isCompleted: TargetCauseOfDeath(stats, core.CoDOnePistolRound), reward: 1000},  // kill target with a headshot
		FixedChallenge{id: 4, name: "Someone Could Hurt Themselves", isCompleted: TargetDiedOfAccident(stats), reward: 1000},        // kill target with an environmental hazard/accident
		FixedChallenge{id: 5, name: "Silent Assassin", isCompleted: SilentAssassin(stats), reward: 5000},                            // only kill targets, no bodies found, not spotted
		FixedChallenge{id: 6, name: "Sniper Assassin", isCompleted: SniperAssassin(stats), reward: 4000},                            // only kill targets with sniper shots, not spotted
		FixedChallenge{id: 7, name: "Suit Only", isCompleted: NeverChangedClothes(stats), reward: 5000},                             // do not change clothes
		FixedChallenge{id: 8, name: "TKEP", isCompleted: AllActorsDead(game), reward: 3000},                                         // kill all actors in the mission
	}
}

func SniperAssassin(statistics core.MissionStats) func() bool {
	return func() bool {
		return statistics.PlayerKilledTargetsWithSniperOnly() && !statistics.BeenSpotted
	}
}

func SilentAssassin(statistics core.MissionStats) func() bool {
	return func() bool {
		return statistics.OnlyKilledTargets() && !statistics.BodiesFound && !statistics.BeenSpotted
	}
}

func NeverChangedClothes(statistics core.MissionStats) func() bool {
	return func() bool {
		return statistics.DisguisesWorn.Cardinality() == 0
	}
}

func AllActorsDead(game GameInterface) func() bool {
	return func() bool {
		missionMap := game.GetMap()
		for _, actor := range missionMap.Actors() {
			if actor == missionMap.Player {
				continue
			}
			if actor.IsAlive() {
				return false
			}
		}
		for _, downedActor := range missionMap.DownedActors() {
			if downedActor == missionMap.Player {
				continue
			}
			if downedActor.IsAlive() {
				return false
			}
		}
		return true
	}
}
func TargetCauseOfDeath(statistics core.MissionStats, cod core.CoDDescription) func() bool {
	return func() bool {
		for _, kill := range statistics.Kills {
			if kill.CauseOfDeath.Description == cod && kill.VictimType == core.ActorTypeTarget {
				return true
			}
		}
		return false
	}
}

func TargetDiedOfAccident(statistics core.MissionStats) func() bool {
	return func() bool {
		for _, kill := range statistics.Kills {
			if kill.VictimType == core.ActorTypeTarget && (kill.CauseOfDeath.Description == core.CoDFalling ||
				kill.CauseOfDeath.Description == core.CoDBurned ||
				kill.CauseOfDeath.Description == core.CoDDrowned ||
				kill.CauseOfDeath.Description == core.CoDElectrocuted) {
				return true
			}
		}
		return false
	}
}
