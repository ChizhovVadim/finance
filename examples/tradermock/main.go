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

	node.SpawnRegister(gen.Atom("monitoring"), monitoring.NewMonitoring, gen.ProcessOptions{})
	node.SpawnRegister(gen.Atom("broker"), brokermock.FactoryMockBroker, gen.ProcessOptions{})
	node.SpawnRegister(gen.Atom("signal"), signalmock.FactorySignalMock, gen.ProcessOptions{})
	node.SpawnRegister(gen.Atom("strategy"), func() gen.ProcessBehavior {
		return strategy.NewStrategy(
			"mock",
			model.Security{Name: "Si-9.26", Code: "SiU6", Lever: 1},
			model.Portfolio{Broker: gen.Atom("broker"), Firm: "SPBFUT", Portfolio: "test"},
			strategy.SizePolicy{LongLever: 9, ShortLever: 9, MaxLever: 6, Weight: 1},
		)
	}, gen.ProcessOptions{})

	node.Wait()
	return nil
}
