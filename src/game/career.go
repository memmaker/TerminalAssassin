package game

import (
	"encoding/gob"
	"os"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/mapset"
)

func NewCareerFromFile(data *services.ExternalData) *services.CareerData {
	fileReader, fileErr := os.Open("career.gob")
	defer fileReader.Close()
	if fileErr != nil {
		println("Error opening career.gob file: ", fileErr.Error())
		return NewEmptyCareer(data)
	}
	decoder := gob.NewDecoder(fileReader)
	careerData := &services.CareerData{}
	err := decoder.Decode(careerData)
	if err != nil {
		println("Error decoding career.gob file: ", err.Error())
		return NewEmptyCareer(data)
	}
	return careerData
}

func NewEmptyCareer(data *services.ExternalData) *services.CareerData {
	defaultClothes := data.DefaultPlayerClothing()
	defaultWeapon := *data.DefaultWeapon()
	defaultItem := *data.DefaultItem()
	unlockedItems := mapset.NewSet[string]()
	unlockedItems.Add(defaultWeapon.Name)
	unlockedItems.Add(defaultItem.Name)
	return &services.CareerData{
		ExperiencePoints: 0,
		MapStatistics:    make(map[string]*services.MapStatistics),
		UnlockedClothes:  map[string]*core.Clothing{defaultClothes.Name: &defaultClothes},
		UnlockedItems:    *unlockedItems,
	}
}
