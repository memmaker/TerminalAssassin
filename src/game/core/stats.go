package core

import (
    "fmt"
    "github.com/memmaker/terminal-assassin/common"
    rec_files "github.com/memmaker/terminal-assassin/rec-files"
    "time"

    "github.com/memmaker/terminal-assassin/geometry"
)

type KillStatistics struct {
    VictimType               ActorType
    VictimName               string
    CauseOfDeath             CauseOfDeath
    AtLocation               geometry.Point
    AtSecond                 float64
}
type MissionStats struct {
    SecondsNeeded float64
    Kills         []KillStatistics
    BodiesFound   bool
    BeenSpotted   bool
}

func NewMissionStats() *MissionStats {
    return &MissionStats{
        Kills: []KillStatistics{},
    }
}

type StartLocation struct {
    Name     string
    Location geometry.Point
}

func NewStartLocationFromRecord(record []rec_files.Field, lookupLocations func(string) geometry.Point) StartLocation {
    newLocation := StartLocation{}
    for _, field := range record {
        if field.Name == "Named_Location" {
            newLocation.Location = lookupLocations(field.Value)
        } else if field.Name == "Name" {
            newLocation.Name = field.Value
        }
    }
    return newLocation
}
func (s StartLocation) ToStyledText() StyledText {
    return Text(fmt.Sprintf("@l%s@N", s.Name)).WithMarkups(
        map[rune]common.Style{
            'l': common.DefaultStyle.WithFg(common.Green),
        })
}
func (s StartLocation) ToString() string {
    return s.Name
}

type MissionPlan struct {
    chosenWeapon         *Item
    chosenGearOne        *Item
    chosenGearTwo        *Item
    defaultStartLocation geometry.Point
    chosenStartLocation  geometry.Point
    startLocationName    string
}

func (g *MissionPlan) Weapon() *Item {
    return g.chosenWeapon
}
func (g *MissionPlan) GearOne() *Item {
    return g.chosenGearOne
}
func (g *MissionPlan) GearTwo() *Item {
    return g.chosenGearTwo
}
func (g *MissionPlan) SetWeapon(weapon *Item) {
    g.chosenWeapon = weapon
}
func (g *MissionPlan) SetSlotOne(tool *Item) {
    g.chosenGearOne = tool
}
func (g *MissionPlan) SetSlotTwo(tool *Item) {
    g.chosenGearTwo = tool
}

func (g *MissionPlan) SetSpecialStartLocation(name string, location geometry.Point) {
    g.chosenStartLocation = location
    g.startLocationName = name
}

func (g *MissionPlan) SetDefaultStartLocation(spawn geometry.Point) {
    g.defaultStartLocation = spawn
    g.chosenStartLocation = spawn
}

func (g *MissionPlan) Location() (geometry.Point, string) {
    return g.chosenStartLocation, g.startLocationName
}
func (m *MissionStats) OnlyKilledTargets() bool {
    for _, kill := range m.Kills {
        if kill.VictimType != ActorTypeTarget {
            return false
        }
    }
    return true
}
func (m *MissionStats) PlayerKilledTargetsWithSniperOnly() bool {
    for _, kill := range m.Kills {
        if !kill.CauseOfDeath.IsPlayer() {
            continue
        }
        if kill.VictimType != ActorTypeTarget || kill.CauseOfDeath.Description != CoDSnipered {
            return false
        }
    }
    return true
}

func (m *MissionStats) MissionDuration() time.Duration {
    return time.Duration(m.SecondsNeeded*1000) * time.Millisecond
}

func (m *MissionStats) AddKill(victim *Actor, death CauseOfDeath, location geometry.Point, timeInSeconds float64) {
    kill := KillStatistics{
        VictimName:   victim.Name,
        VictimType:   victim.Type,
        CauseOfDeath: death,
        AtLocation:   location,
        AtSecond:     timeInSeconds,
    }
    m.Kills = append(m.Kills, kill)
    println(fmt.Sprintf("Killed %s by %s at %s", kill.VictimName, kill.CauseOfDeath.Description, kill.AtLocation))
}

func (m *MissionStats) StartMission() {
    m.SecondsNeeded = 0
    m.BodiesFound = false
    m.BeenSpotted = false
    m.Kills = []KillStatistics{}
}
