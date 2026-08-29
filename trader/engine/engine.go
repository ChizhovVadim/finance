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
	signals                  []gen.PID
	strategies               []gen.PID
	waitingShouldCheckStatus bool
}

func FactoryEngine() gen.ProcessBehavior {
	return &Engine{}
}

func (eng *Engine) Init(args ...any) error {
	spec := args[0].(EngineSpec)

	for _, signalSpec := range spec.Signals {
		pid, err := eng.Spawn(signalSpec.FactorySignal, gen.ProcessOptions{}, signalSpec.Args...)
		if err != nil {
			return err
		}
		eng.signals = append(eng.signals, pid)
	}

	for _, strategySpec := range spec.Strategies {
		pid, err := eng.Spawn(strategySpec.FactoryStrategy, gen.ProcessOptions{}, strategySpec.Args...)
		if err != nil {
			return err
		}
		eng.strategies = append(eng.strategies, pid)
	}

	eng.SendAfter(eng.PID(), shouldCheckStatus{}, 3*time.Second)
	eng.waitingShouldCheckStatus = true

	eng.Log().Info("started")
	return nil
}

func (eng *Engine) HandleMessage(from gen.PID, message any) error {
	switch message.(type) {
	case model.MonitoringRequest:
		// Сообщение может прийти от многих стратегий, поэтому throttling.
		if !eng.waitingShouldCheckStatus {
			eng.waitingShouldCheckStatus = true
			eng.SendAfter(eng.PID(), shouldCheckStatus{}, 10*time.Second)
		}
	case shouldCheckStatus:
		eng.waitingShouldCheckStatus = false
		eng.checkStatus()
	}
	return nil
}

func (eng *Engine) Terminate(reason error) {
	eng.Log().Info("terminated with reason: %s", reason)
}
