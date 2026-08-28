package traderapp

import "ergo.services/ergo/gen"

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
		Mode:        gen.ApplicationModeTransient,
		Group: []gen.ApplicationMemberSpec{
			{
				Name:    "mysup",
				Factory: factoryTraderSup,
				Args:    []any{app.options},
			},
		},
	}, nil
}

// Start invoked once the application started
func (app *TraderApp) Start(mode gen.ApplicationMode) {}

// Terminate invoked once the application stopped
func (app *TraderApp) Terminate(reason error) {}
