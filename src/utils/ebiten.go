package utils

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

func SecondsToTicks(seconds float64) int {
	return int(seconds * float64(ebiten.TPS()))
}

func TicksToSeconds(ticks int) float64 {
	return float64(ticks) / float64(ebiten.TPS())
}
func UTicksToSeconds(ticks uint64) float64 {
	return float64(ticks) / float64(ebiten.TPS())
}

func RandomStringFromSlice(slice []string) string {
	return slice[rand.Intn(len(slice))]
}
