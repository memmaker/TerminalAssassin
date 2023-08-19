package utils

import (
	"log"
	"time"
)

func NewStringComparer() func(i int, j int) bool {
	return func(i int, j int) bool {
		return i < j
	}
}
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
