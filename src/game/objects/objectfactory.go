package objects

import (
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/game/stimuli"
)

// two use-cases:
// iteration for the editor
// resolve by name for instantiation

type ObjectProducer struct {
	Name   string
	Icon   rune
	Create func(string) services.Object
}

func CreateBoulder(engine services.Engine, description string, symbol rune) *Boulder {
	boulder := NewBoulder(description, symbol)
	engine.SubscribeToEvents(services.NewFilter(func(event services.TriggerEvent) bool {
		if event.Key == boulder.GetKey() {
			boulder.Activate(engine)
		}
		return false
	}))
	return boulder
}

func NewFactory(engine services.Engine) *ObjectFactory {
	return &ObjectFactory{engine: engine}
}

type ObjectFactory struct {
	engine     services.Engine
	factoryMap map[string]services.ObjectCreator
}

func (f ObjectFactory) getFactoryMap() map[string]services.ObjectCreator {
	if f.factoryMap == nil {
		f.factoryMap = make(map[string]services.ObjectCreator)
		for _, v := range f.SimpleObjects() {
			f.factoryMap[v.Name] = v
		}
	}
	return f.factoryMap
}

func (f ObjectFactory) NewObjectFromName(name string) services.Object {
	factoryMap := f.getFactoryMap()
	producer, ok := factoryMap[name]
	if !ok {
		return nil
	}
	return producer.Create(name)
}

func (f ObjectFactory) SimpleObjects() []services.ObjectCreator {
	return []services.ObjectCreator{
		{
			Name: "radio (distractor)",
			Icon: core.GlyphRadio,
			Create: func(name string) services.Object {
				return NewZoneDistractor(name, core.GlyphRadio)
			},
		},
		{
			Name: "power box (distractor)",
			Icon: core.GlyphPowerBox,
			Create: func(name string) services.Object {
				return NewZoneDistractor(name, core.GlyphPowerBox)
			},
		},
		{
			Name: "light window (open)",
			Icon: core.GlyphOpenWindow,
			Create: func(name string) services.Object {
				return NewOpenWindowAt(name, 15)
			},
		},
		{
			Name: "light window (closed)",
			Icon: core.GlyphClosedWindow,
			Create: func(name string) services.Object {
				return NewClosedWindowAt(name, 15)
			},
		},
		{
			Name: "light door (open)",
			Icon: core.GlyphOpenDoor,
			Create: func(name string) services.Object {
				return NewOpenDoorAt(name, 15)
			},
		},
		{
			Name: "light door (closed)",
			Icon: core.GlyphClosedDoor,
			Create: func(name string) services.Object {
				return NewClosedDoorAt(name, 15)
			},
		},
		{
			Name: "light door (locked)",
			Icon: core.GlyphLockedDoor,
			Create: func(name string) services.Object {
				return NewLockedDoorAt(name, "", 15)
			},
		},
		{
			Name: "electronic door (locked)",
			Icon: core.GlyphLockedDoorElectronic,
			Create: func(name string) services.Object {
				return NewLockedElectronicDoorAt(name, "", 15)
			},
		},
		{
			Name: "locker (container)",
			Icon: core.GlyphLocker,
			Create: func(name string) services.Object {
				return NewCorpseContainerAt(name, core.GlyphLocker)
			},
		},
		{
			Name: "cabinet (container)",
			Icon: core.GlyphCabinet,
			Create: func(name string) services.Object {
				return NewCorpseContainerAt(name, core.GlyphCabinet)
			},
		},
		{
			Name: "oil drum (liquid container)",
			Icon: core.GlyphOilDrum,
			Create: func(name string) services.Object {
				return NewLiquidContainer(name, core.GlyphOilDrum, stimuli.StimulusBurnableLiquid)
			},
		},
		{
			Name: "water barrel (liquid container)",
			Icon: core.GlyphOilDrum,
			Create: func(name string) services.Object {
				return NewLiquidContainer(name, core.GlyphOilDrum, stimuli.StimulusWater)
			},
		},
		{
			Name: "a sink (liquid faucet)",
			Icon: core.GlyphSink,
			Create: func(name string) services.Object {
				return NewLiquidFaucet(name, core.GlyphSink, stimuli.StimulusWater)
			},
		},
		{
			Name: "lever (trigger)",
			Icon: 'L',
			Create: func(name string) services.Object {
				return NewTriggerObject(name, 'L')
			},
		},
		{
			Name: "boulder (falling)",
			Icon: 'b',
			Create: func(name string) services.Object {
				return CreateBoulder(f.engine, name, 'b')
			},
		},
	}
}
