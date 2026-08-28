package signalrandom

import (
	"finance/model"
	"math/rand/v2"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type messageTick struct{}

type signalRandom struct {
	act.Actor
	name          string
	interval      time.Duration
	currentSignal model.Signal
}

func FactorySignalRandom() gen.ProcessBehavior {
	return &signalRandom{}
}

func (a *signalRandom) Init(args ...any) error {
	a.name = args[0].(string)
	a.interval = 5 * time.Second

	a.SendAfter(a.PID(), messageTick{}, a.interval)
	a.Log().Info("started")
	return nil
}

func (a *signalRandom) HandleMessage(from gen.PID, message any) error {
	switch message.(type) {
	case messageTick:
		a.currentSignal = model.Signal{
			Name:     a.name,
			Deadline: time.Now().Add(5 * time.Minute),
			Price:    100_000,
			Value:    rand.Float64()*2 - 1,
		}
		a.Send(a.Parent(), a.currentSignal)
		a.SendAfter(a.PID(), messageTick{}, a.interval)
	}
	return nil
}

func (a *signalRandom) HandleCall(from gen.PID, ref gen.Ref, req any) (any, error) {
	switch req.(type) {
	case model.GetStatusRequest:
		return a.currentSignal, nil
	}
	return gen.ErrUnsupported, nil
}
