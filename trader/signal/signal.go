package signal

import (
	"finance/trader/model"
	"fmt"
	"iter"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

const SignalUpdatedEventName = gen.Atom("signalUpdated")

type IAdvisor interface {
	Add(dt time.Time, price float64) (prediction float64, ok bool)
}

type ICandleReader interface {
	Candles(candleInterval string, securityCode string) iter.Seq2[model.Candle, error]
}

type Signal struct {
	act.Actor
	name                    string
	security                model.Security
	candleInterval          string
	advisorFactory          func() (IAdvisor, error)
	marketData              gen.Atom
	start                   time.Time
	advisor                 IAdvisor
	signalUpdatedEventToken gen.Ref
	currentSignal           model.Signal
}

func NewSignal(
	name string,
	security model.Security,
	candleInterval string,
	advisorFactory func() (IAdvisor, error),
	marketData gen.Atom,
) *Signal {
	return &Signal{
		name:           name,
		security:       security,
		candleInterval: candleInterval,
		advisorFactory: advisorFactory,
		marketData:     marketData,
	}
}

func (a *Signal) Init(args ...any) error {
	a.start = time.Now()

	advisor, err := a.advisorFactory()
	if err != nil {
		return err
	}
	a.advisor = advisor

	signalUpdatedEventToken, err := a.RegisterEvent(SignalUpdatedEventName, gen.EventOptions{Buffer: 1})
	if err != nil {
		return err
	}
	a.signalUpdatedEventToken = signalUpdatedEventToken

	_, err = a.MonitorEvent(gen.Event{
		Name: gen.Atom("candleFinished"),
		Node: a.Node().Name(),
	})
	if err != nil {
		return err
	}

	err = a.Send(a.marketData, model.SubscribeCandlesRequest{
		Ssecurity: a.security,
		Timeframe: a.candleInterval,
	})
	if err != nil {
		return err
	}

	a.Log().Info("started. init signal: %v", a.currentSignal)
	return nil
}

func (s *Signal) addHistoryCandles(historyCandles iter.Seq2[model.Candle, error]) error {
	var (
		firstCandle model.Candle
		lastCandle  model.Candle
		size        int
	)
	for candle, err := range historyCandles {
		if err != nil {
			return err
		}

		if size == 0 {
			firstCandle = candle
		}
		lastCandle = candle
		size += 1

		var prediction, ok = s.advisor.Add(candle.DateTime, candle.ClosePrice)
		if !ok {
			continue
		}
		s.currentSignal = model.Signal{
			Name:     s.name,
			Deadline: candle.DateTime.Add(9 * time.Minute), //9 минут от открытия бара, 4 минуты от закрытия бара.
			Price:    candle.ClosePrice,
			Value:    prediction,
		}
	}
	if size == 0 {
		s.Log().Warning("History candles empty")
	} else {
		s.Log().Debug("History candles %v %v %v",
			size, firstCandle, lastCandle)
	}
	return nil
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
	// советник следит только за своими барами
	if !( //a.candleInterval == candle.Interval &&
	a.security.Code == candle.SecurityCode) {
		return
	}
	prediction, predictionOk := a.advisor.Add(candle.DateTime, candle.ClosePrice)
	if !predictionOk {
		return
	}
	var signalValueChanged = a.currentSignal.Value != prediction
	a.currentSignal = model.Signal{
		Name:     a.name,
		Deadline: candle.DateTime.Add(9 * time.Minute),
		Price:    candle.ClosePrice,
		Value:    prediction,
	}
	if a.currentSignal.Deadline.Before(a.start) {
		return
	}
	if signalValueChanged {
		a.Log().Info("New signal %v", a.currentSignal)
	}
	a.SendEvent(SignalUpdatedEventName, a.signalUpdatedEventToken, model.SignalUpdated{
		Signal: a.currentSignal,
	})
}

func (a *Signal) HandleInspect(from gen.PID, item ...string) map[string]string {
	return map[string]string{
		"Name":     a.currentSignal.Name,
		"Deadline": a.currentSignal.Deadline.Format("2006-01-02 15:04"),
		"Price":    fmt.Sprintf("%v", a.currentSignal.Price),
		"Value":    fmt.Sprintf("%v", a.currentSignal.Value),
	}
}
