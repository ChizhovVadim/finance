package strategy

import "finance/model"

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
