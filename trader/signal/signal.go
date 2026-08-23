package signal

import (
	"finance/trader/model"
	"fmt"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type IAdvisor interface {
	Add(dt time.Time, price float64) (prediction float64, ok bool)
}

type IHistoryMarketData interface {
	Load(security model.Security, interval string) ([]model.Candle, error)
}

type Signal struct {
	act.Actor
	signalUpdatedEventName  gen.Atom
	security                model.Security
	candleInterval          string
	advisorFactory          func() (IAdvisor, error)
	marketData              gen.Atom
	historyMarketData       IHistoryMarketData
	start                   time.Time
	advisor                 IAdvisor
	signalUpdatedEventToken gen.Ref
	currentSignal           model.Signal
}

func FactorySignal(
	signalUpdatedEventName gen.Atom,
	security model.Security,
	candleInterval string,
	advisorFactory func() (IAdvisor, error),
	marketData gen.Atom,
	historyMarketData IHistoryMarketData,
) func() gen.ProcessBehavior {
	return func() gen.ProcessBehavior {
		return &Signal{
			signalUpdatedEventName: signalUpdatedEventName,
			security:               security,
			candleInterval:         candleInterval,
			advisorFactory:         advisorFactory,
			marketData:             marketData,
			historyMarketData:      historyMarketData,
		}
	}
}

func (a *Signal) Init(args ...any) error {
	a.start = time.Now()

	advisor, err := a.advisorFactory()
	if err != nil {
		return err
	}
	a.advisor = advisor

	if a.historyMarketData != nil {
		historyCandles, err := a.historyMarketData.Load(a.security, a.candleInterval)
		if err != nil {
			return err
		}
		a.addHistoryCandles(historyCandles)
	}

	signalUpdatedEventToken, err := a.RegisterEvent(a.signalUpdatedEventName, gen.EventOptions{Buffer: 1})
	if err != nil {
		return err
	}
	a.signalUpdatedEventToken = signalUpdatedEventToken

	resp, err := a.Call(a.marketData, model.GetLastCandlesRequest{
		Ssecurity: a.security,
		Timeframe: a.candleInterval,
	})
	if err != nil {
		return err
	}
	if errResponse, ok := resp.(error); ok {
		return errResponse
	}
	a.addHistoryCandles(resp.([]model.Candle))

	resp, err = a.Call(a.marketData, model.GetCandleFinishedEvent{
		Ssecurity: a.security,
		Timeframe: a.candleInterval,
	})
	if err != nil {
		return err
	}
	if respErr, ok := resp.(error); ok {
		return respErr
	}
	candleFinishedEventName := resp.(gen.Atom)

	_, err = a.MonitorEvent(gen.Event{
		Name: candleFinishedEventName,
		Node: a.Node().Name(),
	})
	if err != nil {
		return err
	}
	// Process any existing events that were returned
	/*if len(events) > 0 {
		a.Log().Info("Processing %d existing events", len(events))
		for _, existingEvent := range events {
			_ = existingEvent
		}
	}*/

	a.Log().Info("started init signal: %v", a.currentSignal)
	return nil
}

func (s *Signal) addHistoryCandles(historyCandles []model.Candle) {
	for _, candle := range historyCandles {
		var prediction, ok = s.advisor.Add(candle.DateTime, candle.ClosePrice)
		if !ok {
			continue
		}
		s.currentSignal = model.Signal{
			Deadline: candle.DateTime.Add(9 * time.Minute), //9 минут от открытия бара, 4 минуты от закрытия бара.
			Price:    candle.ClosePrice,
			Value:    prediction,
		}
	}
	if len(historyCandles) == 0 {
		s.Log().Warning("History candles empty")
	} else {
		s.Log().Debug("History candles %v %v %v",
			len(historyCandles),
			historyCandles[0],
			historyCandles[len(historyCandles)-1])
	}
}

func (a *Signal) HandleEvent(event gen.MessageEvent) error {
	switch message := event.Message.(type) {
	case model.CandleFinished:
		a.onCandle(message.Candle)
		return nil
	}
	return nil
}

func (a *Signal) onCandle(candle model.Candle) {
	if candle.DateTime.Add(5 * time.Minute).After(a.start) {
		a.Log().Debug("New candle %v", candle)
	}
	prediction, predictionOk := a.advisor.Add(candle.DateTime, candle.ClosePrice)
	if !predictionOk {
		return
	}
	var signalValueChanged = a.currentSignal.Value != prediction
	a.currentSignal = model.Signal{
		Deadline: candle.DateTime.Add(9 * time.Minute),
		Price:    candle.ClosePrice,
		Value:    prediction,
	}
	if a.currentSignal.Deadline.Before(a.start) {
		return
	}
	if signalValueChanged {
		a.Log().Info("New signal %v", a.currentSignal)
	} else {
		a.Log().Debug("New signal %v", a.currentSignal)
	}
	a.SendEvent(a.signalUpdatedEventName, a.signalUpdatedEventToken, model.SignalUpdated{
		Signal: a.currentSignal,
	})
}

func (a *Signal) HandleInspect(from gen.PID, item ...string) map[string]string {
	return map[string]string{
		"Deadline": a.currentSignal.Deadline.Format("2006-01-02 15:04"),
		"Price":    fmt.Sprintf("%v", a.currentSignal.Price),
		"Value":    fmt.Sprintf("%v", a.currentSignal.Value),
	}
}
