package game

import (
	"encoding/gob"
	"os"

	"github.com/memmaker/terminal-assassin/game/services"
)

func NewCareerFromFile() *services.CareerData {
	fileReader, fileErr := os.Open("career.gob")
	defer fileReader.Close()
	if fileErr != nil {
		println("Error opening career.gob file: ", fileErr.Error())
		return NewEmptyCareer()
	}
	decoder := gob.NewDecoder(fileReader)
	careerData := &services.CareerData{}
	err := decoder.Decode(careerData)
	if err != nil {
		println("Error decoding career.gob file: ", err.Error())
		return NewEmptyCareer()
	}
	return careerData
}

func NewEmptyCareer() *services.CareerData {
	return &services.CareerData{
		ExperiencePoints: 0,
		MapStatistics:    make(map[string]*services.MapStatistics),
		UnlockedSkills:   services.PlayerSkills{DoubleAssassination: true},
	}
}
