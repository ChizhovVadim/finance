package signal

import (
	"finance/model"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type signal struct {
	act.Actor
	start            time.Time
	name             string
	advisor          IAdvisor
	signalEventToken gen.Ref
	currentSignal    model.Signal
}

func FactorySignal() gen.ProcessBehavior {
	return &signal{}
}

func (sig *signal) Init(args ...any) error {
	sig.start = time.Now()

	spec := args[0].(SignalSpec)

	sig.name = spec.Name

	advisor, err := spec.AdvisorFactory()
	if err != nil {
		return err
	}
	sig.advisor = advisor

	signalEventToken, err := sig.RegisterEvent(gen.Atom(sig.name),
		gen.EventOptions{
			Buffer: 1,
		})
	if err != nil {
		return err
	}
	sig.signalEventToken = signalEventToken

	err = sig.Send(model.MultyBroker, model.GetCandleFinishedRequest{
		Security:  spec.Security,
		Timeframe: spec.CandleInterval,
	})
	if err != nil {
		return err
	}

	sig.Log().Info("started")
	return nil
}

func (sig *signal) HandleMessage(from gen.PID, message any) error {
	switch message := message.(type) {
	case model.GetCandleFinishedResponse:
		events, err := sig.MonitorEvent(gen.Event{
			Name: message.EventName,
		})
		if err != nil {
			return err
		}

		// Process any existing events that were returned
		if len(events) > 0 {
			sig.Log().Debug("Processing %d existing events", len(events))
			for _, existingEvent := range events {
				sig.HandleEvent(existingEvent)
			}
			sig.Log().Info("init signal: %v", sig.currentSignal)
		}
	}
	return nil
}

func (sig *signal) HandleCall(from gen.PID, ref gen.Ref, req any) (any, error) {
	switch req.(type) {
	case model.GetStatusRequest:
		return sig.currentSignal, nil
	}
	return gen.ErrUnsupported, nil
}

func (sig *signal) HandleEvent(event gen.MessageEvent) error {
	switch message := event.Message.(type) {
	case model.Candle:
		sig.onCandle(message)
		return nil
	}
	return nil
}

func (sig *signal) onCandle(candle model.Candle) {
	prediction, predictionOk := sig.advisor.Add(candle.DateTime, candle.ClosePrice)
	if !predictionOk {
		return
	}
	sig.currentSignal = model.Signal{
		Name:     sig.name,
		Deadline: candle.DateTime.Add(9 * time.Minute), //9 минут от открытия бара, 4 минуты от закрытия бара.
		Price:    candle.ClosePrice,
		Value:    prediction,
	}
	if sig.currentSignal.Deadline.After(sig.start) {
		sig.Log().Info("New signal %v", sig.currentSignal)
		return
	}
	if sig.currentSignal.Deadline.After(time.Now()) {
		sig.SendEvent(gen.Atom(sig.name), sig.signalEventToken, sig.currentSignal)
	}
}
