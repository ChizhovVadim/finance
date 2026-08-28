package traderapp

import (
	"finance/trader/engine"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type traderSup struct {
	act.Supervisor
}

func factoryTraderSup() gen.ProcessBehavior {
	return &traderSup{}
}

func (s *traderSup) Init(args ...any) (act.SupervisorSpec, error) {

	var options = args[0].(Options)

	var spec = act.SupervisorSpec{
		Type: act.SupervisorTypeOneForOne,
		Restart: act.SupervisorRestart{
			Strategy: act.SupervisorStrategyPermanent,
		},
	}

	// add children
	spec.Children = []act.SupervisorChildSpec{
		{
			Name:    gen.Atom("TradingEngine"),
			Factory: engine.FactoryEngine,
			Args:    []any{options.EngineSpec},
		},
	}

	return spec, nil
}
