package utils

import (
	"log"
	"strconv"
	"strings"
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

func MustParseFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		println("Error parsing float: " + s)
		return 0
	}
	return f
}

func MustParseInt(s string) int {
	f, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		println("Error parsing integer: " + s)
		return 0
	}
	return int(f)
}

