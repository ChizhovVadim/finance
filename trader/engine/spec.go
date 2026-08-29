package engine

import (
	"ergo.services/ergo/gen"
)

type EngineSpec struct {
	Signals    []SignalSpec
	Strategies []StrategySpec
}

type SignalSpec struct {
	FactorySignal gen.ProcessFactory
	Args          []any
}

type StrategySpec struct {
	FactoryStrategy gen.ProcessFactory
	Args            []any
}
