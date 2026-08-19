package signalmock

import (
	"finance/trader/model"
	"fmt"
	"math/rand/v2"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

const SignalUpdatedEventName = gen.Atom("signalUpdated")

type messageTick struct{}

const Interval = 5 * time.Second

type signalMock struct {
	act.Actor
	signalUpdatedEventToken gen.Ref
	currentSignal           model.Signal
}

func FactorySignalMock() gen.ProcessBehavior {
	return &signalMock{}
}

func (a *signalMock) Init(args ...any) error {
	signalUpdatedEventToken, err := a.RegisterEvent(SignalUpdatedEventName, gen.EventOptions{Buffer: 1})
	if err != nil {
		return err
	}
	a.signalUpdatedEventToken = signalUpdatedEventToken

	a.SendAfter(a.PID(), messageTick{}, Interval)
	a.Log().Info("started")
	return nil
}

func (a *signalMock) HandleMessage(from gen.PID, message any) error {
	switch message.(type) {
	case messageTick:
		a.currentSignal = model.Signal{
			Name:     "mock",
			Deadline: time.Now().Add(5 * time.Minute),
			Price:    100_000,
			Value:    rand.Float64()*2 - 1,
		}
		a.Log().Info("New signal %v", a.currentSignal)
		a.SendEvent(SignalUpdatedEventName, a.signalUpdatedEventToken, model.SignalUpdated{
			Signal: a.currentSignal,
		})
		a.SendAfter(a.PID(), messageTick{}, Interval)
	}
	return nil
}

func (a *signalMock) HandleInspect(from gen.PID, item ...string) map[string]string {
	return map[string]string{
		"Name":     a.currentSignal.Name,
		"Deadline": a.currentSignal.Deadline.Format("2006-01-02 15:04"),
		"Price":    fmt.Sprintf("%v", a.currentSignal.Price),
		"Value":    fmt.Sprintf("%v", a.currentSignal.Value),
	}
}

func (a *signalMock) Terminate(reason error) {
	a.Log().Info("terminated with reason: %s", reason)
}
