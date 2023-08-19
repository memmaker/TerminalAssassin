package stimuli

import (
	"encoding/gob"
	"fmt"
	"strings"
)

type StimulusType string

const (
	StimulusNone            StimulusType = "none"
	StimulusPiercingDamage  StimulusType = "piercing_damage"
	StimulusBluntDamage     StimulusType = "blunt_damage"
	StimulusChokingDamage   StimulusType = "choking_damage"
	StimulusEmeticPoison    StimulusType = "emetic_poison"
	StimulusLethalPoison    StimulusType = "lethal_poison"
	StimulusInducedSleep    StimulusType = "sleep"
	StimulusFire            StimulusType = "fire"
	StimulusExplosionDamage StimulusType = "explosion_damage"
	StimulusWater           StimulusType = "water"
	StimulusBlood           StimulusType = "blood"
	StimulusBurnableLiquid  StimulusType = "make_burnable"
	StimulusHighVoltage     StimulusType = "high_voltage"
)

type Stim struct {
	StimType  StimulusType
	StimForce int
}

func (s Stim) ToString() string {
	return fmt.Sprintf("%s with force %d", s.StimType, s.StimForce)
}

func (s Stim) Type() StimulusType {
	return s.StimType
}

func (s Stim) Force() int {
	return s.StimForce
}

func (s Stim) WithForce(newForceValue int) Stimulus {
	return Stim{StimType: s.StimType, StimForce: newForceValue}
}

type Stimulus interface {
	Type() StimulusType
	Force() int
	ToString() string
	WithForce(newForceValue int) Stimulus
}
type MethodOfDistribution int

func (d MethodOfDistribution) ToString() string {
	switch d {
	case DistributeDirect:
		return "direct"
	case DistributeExplode:
		return "explode"
	case DistributeLiquid:
		return "liquid"
	}
	return "unknown"
}

func NewMethodOfDistributionFromString(method string) MethodOfDistribution {
	switch method {
	case "direct":
		return DistributeDirect
	case "explode":
		return DistributeExplode
	case "liquid":
		return DistributeLiquid
	}
	return DistributeDirect
}

const (
	DistributeDirect MethodOfDistribution = iota
	DistributeExplode
	DistributeLiquid
)

type StimEffect struct {
	Stimuli              []Stimulus
	Distribution         MethodOfDistribution
	Distance             int
	Pressure             int
	DestroyOnApplication bool
}

func (se StimEffect) ToString() string {
	result := make([]string, len(se.Stimuli))
	for index, stim := range se.Stimuli {
		result[index] = stim.ToString()
	}
	return strings.Join(result, ", ")
}

func EffectExplosion(stimForce, blastPressure, blastRadius int) StimEffect {
	return StimEffect{
		Distribution:         DistributeExplode,
		Distance:             blastRadius,
		Pressure:             blastPressure,
		DestroyOnApplication: true,
		Stimuli: []Stimulus{
			Stim{
				StimType:  StimulusExplosionDamage,
				StimForce: stimForce,
			},
			Stim{
				StimType:  StimulusFire,
				StimForce: stimForce,
			},
		},
	}
}

func EffectLeak(stimType StimulusType, stimForce int, leakSize int) StimEffect {
	return StimEffect{
		Distribution: DistributeLiquid,
		Distance:     leakSize,
		Stimuli: []Stimulus{
			Stim{StimType: stimType, StimForce: stimForce},
		}}
}

func init() {
	gob.Register(&Stim{})
}
