package traderapp

import (
	"finance/trader/brokermulty"
	"finance/trader/engine"
)

type Options struct {
	MultyBrokerSpec brokermulty.MultyBrokerSpec
	EngineSpec      engine.EngineSpec
}
