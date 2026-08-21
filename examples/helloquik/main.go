package main

import (
	"finance/trader/brokerquik"
	"finance/trader/model"
	"flag"
	"fmt"
	"time"

	"ergo.services/ergo"
	"ergo.services/ergo/gen"
)

// Проверка связи с Quik через lua скрипты.
func main() {
	var err = run()
	fmt.Println("App finished", "err", err)
}

func run() error {
	var port int
	flag.IntVar(&port, "port", 0, "")
	flag.Parse()
	if port == 0 {
		return fmt.Errorf("port required")
	}

	var options gen.NodeOptions
	options.Log.DefaultLogger.TimeFormat = time.DateTime
	options.Log.DefaultLogger.IncludeName = true
	options.Log.DefaultLogger.IncludeBehavior = true

	node, err := ergo.StartNode(gen.Atom("example@localhost"), options)
	if err != nil {
		return fmt.Errorf("Unable to start node %w", err)
	}

	node.SpawnRegister(gen.Atom("quik"),
		func() gen.ProcessBehavior {
			return brokerquik.NewQuikBroker(port)
		},
		gen.ProcessOptions{})

	node.Send(gen.Atom("quik"), model.BrokerMessageInfoRequest{Message: "Где деньги, Лебовски?"})

	node.Wait()
	return nil
}
