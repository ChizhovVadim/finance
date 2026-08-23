package main

import (
	"finance/trader/brokermock"
	"finance/trader/model"
	"finance/trader/monitoring"
	"finance/trader/signalmock"
	"finance/trader/strategy"
	"fmt"
	"time"

	"ergo.services/ergo"
	"ergo.services/ergo/gen"
)

// Генерация случайных сигналов и их исполнение "на бумаге"
func main() {
	var err = run()
	fmt.Println("App finished", "err", err)
}

func run() error {
	var options gen.NodeOptions
	//options.Log.Level = gen.LogLevelDebug
	options.Log.DefaultLogger.TimeFormat = time.DateTime
	options.Log.DefaultLogger.IncludeName = true
	options.Log.DefaultLogger.IncludeBehavior = true

	node, err := ergo.StartNode(gen.Atom("example@localhost"), options)
	if err != nil {
		return fmt.Errorf("Unable to start node %w", err)
	}

	const (
		signalEventName = gen.Atom("signalEvent_rnd")
		brokerName      = gen.Atom("broker")
	)

	node.SpawnRegister(gen.Atom("monitoring"), monitoring.FactoryMonitoring, gen.ProcessOptions{})
	node.SpawnRegister(brokerName, brokermock.FactoryMockBroker, gen.ProcessOptions{})
	node.SpawnRegister(gen.Atom("signal"), signalmock.FactorySignalMock(signalEventName), gen.ProcessOptions{})
	node.SpawnRegister(gen.Atom("strategy"),
		strategy.FactoryStrategy(
			signalEventName,
			model.Security{Name: "Si-9.26", Code: "SiU6", Lever: 1},
			model.Portfolio{Broker: brokerName, Firm: "SPBFUT", Portfolio: "test"},
			strategy.SizePolicy{LongLever: 9, ShortLever: 9, MaxLever: 6, Weight: 1},
		),
		gen.ProcessOptions{})

	node.Wait()
	return nil
}
