// Package rng provides a shared seeded random-number generator used for all
// gameplay-affecting randomness so that replays can reproduce the same results.
package rng

import "math/rand"

// R is the global gameplay RNG. Replace it via Seed before starting a mission.
var R = rand.New(rand.NewSource(0))

// Seed replaces R with a new source seeded to the given value.
func Seed(seed int64) {
	R = rand.New(rand.NewSource(seed))
}

