package traderapp

import (
	"finance/model"
	"finance/trader/brokermulty"
	"finance/trader/engine"

	"ergo.services/ergo/gen"
)

func CreateTraderApp(options Options) gen.ApplicationBehavior {
	return &TraderApp{
		options: options,
	}
}

type TraderApp struct {
	options Options
}

// Load invoked on loading application using method ApplicationLoad of gen.Node interface.
func (app *TraderApp) Load(node gen.Node, args ...any) (gen.ApplicationSpec, error) {
	return gen.ApplicationSpec{
		Name:        "myapp",
		Description: "description of this application",
		Mode:        gen.ApplicationModeTemporary,
		Group: []gen.ApplicationMemberSpec{
			{
				Name:    model.MultyBroker,
				Factory: brokermulty.FactoryMultyBroker,
				Args:    []any{app.options.MultyBrokerSpec},
			},
			{
				Name:    gen.Atom("TradingEngine"),
				Factory: engine.FactoryEngine,
				Args:    []any{app.options.EngineSpec},
			},
		},
	}, nil
}

// Start invoked once the application started
func (app *TraderApp) Start(mode gen.ApplicationMode) {}

// Terminate invoked once the application stopped
func (app *TraderApp) Terminate(reason error) {}
