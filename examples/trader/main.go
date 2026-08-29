package main

import (
	"finance/model"
	"finance/trader/brokermulty"
	"finance/trader/brokerpaper"
	"finance/trader/engine"
	"finance/trader/signalrandom"
	"finance/trader/traderapp"
	"fmt"
	"time"

	"ergo.services/ergo"
	"ergo.services/ergo/gen"
)

func main() {
	var securitySi = model.Security{Name: "Si-9.26", Code: "SiU6", Lever: 1}

	var traderOptions = traderapp.Options{
		MultyBrokerSpec: brokermulty.MultyBrokerSpec{
			Clients: []brokermulty.ClientSpec{
				{Name: "client_fake", FactoryBroker: brokerpaper.FactoryPaperBroker},
				//{Name: "client_myquik", FactoryBroker: brokerquik.FactoryQuikBroker, Args: []any{34132}}, //port
			},
			//MarketData: "client_myquik",
		},
		EngineSpec: engine.EngineSpec{
			Signals: []engine.SignalSpec{
				{FactorySignal: signalrandom.FactorySignalRandom, Args: []any{"signal_random"}},
			},
			Strategies: []engine.StrategySpec{
				// Случайный сигнал можно протестировать на бумажном брокере
				{
					SignalName: "signal_random",
					Security:   securitySi,
					Portfolio:  model.Portfolio{Client: "client_fake", Firm: "SPBFUT", Portfolio: "account_fake"},
					SizePolicy: engine.SizePolicy{LongLever: 9, ShortLever: 9, MaxLever: 9, Weight: 1},
				},
			},
		},
	}

	var options gen.NodeOptions
	options.Log.Level = gen.LogLevelDebug
	options.Log.DefaultLogger.TimeFormat = time.DateTime
	options.Log.DefaultLogger.IncludeName = true
	options.Log.DefaultLogger.IncludeBehavior = true
	options.Applications = []gen.ApplicationBehavior{
		traderapp.CreateTraderApp(traderOptions),
	}
	// starting node
	const NodeName = "trader@localhost"
	node, err := ergo.StartNode(gen.Atom(NodeName), options)
	if err != nil {
		fmt.Printf("Unable to start node '%s': %s\n", NodeName, err)
		return
	}
	node.Log().Info("Node started %v", NodeName)
	node.Wait()
}
