package engine

import (
	"finance/model"

	"ergo.services/ergo/gen"
)

type EngineSpec struct {
	Clients    []ClientSpec
	MarketData string
	Signals    []SignalSpec
	Strategies []StrategySpec
}

type ClientSpec struct {
	Name          string
	FactoryBroker gen.ProcessFactory
	Args          []any
}

type SignalSpec struct {
	FactorySignal gen.ProcessFactory
	Args          []any
}

type StrategySpec struct {
	SignalName string
	Security   model.Security
	Portfolio  model.Portfolio
	SizePolicy SizePolicy
}

type SizePolicy struct {
	LongLever  float64
	ShortLever float64
	MaxLever   float64
	Weight     float64
	MaxAmount  float64
}
