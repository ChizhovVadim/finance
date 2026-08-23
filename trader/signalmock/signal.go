package signalmock

import (
	"finance/trader/model"
	"fmt"
	"math/rand/v2"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type messageTick struct{}

const Interval = 5 * time.Second

type signalMock struct {
	act.Actor
	eventName     gen.Atom
	eventToken    gen.Ref
	currentSignal model.Signal
}

func FactorySignalMock(
	eventName gen.Atom,
) func() gen.ProcessBehavior {
	return func() gen.ProcessBehavior {
		return &signalMock{
			eventName: eventName,
		}
	}
}

func (a *signalMock) Init(args ...any) error {
	eventToken, err := a.RegisterEvent(a.eventName, gen.EventOptions{Buffer: 1})
	if err != nil {
		return err
	}
	a.eventToken = eventToken

	a.SendAfter(a.PID(), messageTick{}, Interval)
	a.Log().Info("started")
	return nil
}

func (a *signalMock) HandleMessage(from gen.PID, message any) error {
	switch message.(type) {
	case messageTick:
		a.currentSignal = model.Signal{
			Deadline: time.Now().Add(5 * time.Minute),
			Price:    100_000,
			Value:    rand.Float64()*2 - 1,
		}
		a.Log().Info("New signal %v", a.currentSignal)
		a.SendEvent(a.eventName, a.eventToken, model.SignalUpdated{
			Signal: a.currentSignal,
		})
		a.SendAfter(a.PID(), messageTick{}, Interval)
	}
	return nil
}

func (a *signalMock) HandleInspect(from gen.PID, item ...string) map[string]string {
	return map[string]string{
		"Deadline": a.currentSignal.Deadline.Format("2006-01-02 15:04"),
		"Price":    fmt.Sprintf("%v", a.currentSignal.Price),
		"Value":    fmt.Sprintf("%v", a.currentSignal.Value),
	}
}

func (a *signalMock) Terminate(reason error) {
	a.Log().Info("terminated with reason: %s", reason)
}
