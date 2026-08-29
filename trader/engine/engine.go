package engine

import (
	"finance/model"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type shouldCheckStatus struct{}

type Engine struct {
	act.Actor
	start                    time.Time
	signals                  []gen.PID
	strategies               []StrategyService
	waitingShouldCheckStatus bool
}

func FactoryEngine() gen.ProcessBehavior {
	return &Engine{}
}

func (eng *Engine) Init(args ...any) error {
	eng.start = time.Now()

	spec := args[0].(EngineSpec)

	for _, signalSpec := range spec.Signals {
		pid, err := eng.Spawn(signalSpec.FactorySignal, gen.ProcessOptions{}, signalSpec.Args...)
		if err != nil {
			return err
		}
		eng.signals = append(eng.signals, pid)
	}

	for _, strategySpec := range spec.Strategies {
		var strategyService = StrategyService{
			signalName: strategySpec.SignalName,
			security:   strategySpec.Security,
			portfolio:  strategySpec.Portfolio,
			sizePolicy: strategySpec.SizePolicy,
		}
		err := eng.strategyInitAmount(&strategyService)
		if err != nil {
			return err
		}
		err = eng.strategyInitPosition(&strategyService)
		if err != nil {
			return err
		}
		eng.strategies = append(eng.strategies, strategyService)
	}

	eng.SendAfter(eng.PID(), shouldCheckStatus{}, 3*time.Second)
	eng.waitingShouldCheckStatus = true

	eng.Log().Info("started")
	return nil
}

func (eng *Engine) HandleMessage(from gen.PID, message any) error {
	switch message := message.(type) {
	case shouldCheckStatus:
		eng.waitingShouldCheckStatus = false
		eng.checkStatus()
	case model.Signal:
		eng.Log().Info("New signal %v", message)
		for i := range eng.strategies {
			var strategy = &eng.strategies[i]
			eng.onSignal(strategy, message)
		}
	}
	return nil
}

func (eng *Engine) Terminate(reason error) {
	eng.Log().Info("terminated with reason: %s", reason)
}
