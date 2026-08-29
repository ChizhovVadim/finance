package brokermulty

import (
	"finance/model"
	"fmt"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type MultyBroker struct {
	act.Actor
	start                time.Time
	historyMarketData    IHistoryMarketData
	brokers              map[string]gen.PID
	marketdata           gen.PID
	candleFinishedEvents map[gen.Atom]gen.Ref
}

func FactoryMultyBroker() gen.ProcessBehavior {
	return &MultyBroker{
		brokers:              make(map[string]gen.PID),
		candleFinishedEvents: make(map[gen.Atom]gen.Ref),
	}
}

func (b *MultyBroker) Init(args ...any) error {
	b.start = time.Now()

	spec := args[0].(MultyBrokerSpec)

	b.historyMarketData = spec.HistoryMarketData

	for _, clientSpec := range spec.Clients {
		pid, err := b.Spawn(clientSpec.FactoryBroker, gen.ProcessOptions{}, clientSpec.Args...)
		if err != nil {
			return err
		}
		b.brokers[clientSpec.Name] = pid
	}
	if spec.MarketData != "" {
		b.marketdata = b.brokers[spec.MarketData]
	}

	b.Log().Info("started")
	return nil
}

func (b *MultyBroker) HandleMessage(from gen.PID, message any) error {
	switch message := message.(type) {
	case model.BrokerCallbackMessage:
		switch callback := message.Message.(type) {
		case model.Candle:
			b.onCandleCallback(callback)
		}
	case model.GetCandleFinishedRequest:
		var candleFinishedEventName = getCandleFinishedEventName(message.Timeframe, message.Security.Code)
		if _, found := b.candleFinishedEvents[candleFinishedEventName]; !found {
			candleFinishedEventToken, err := b.RegisterEvent(candleFinishedEventName,
				gen.EventOptions{
					Buffer: 5_000,
				})
			if err != nil {
				return err
			}
			b.candleFinishedEvents[candleFinishedEventName] = candleFinishedEventToken
			err = b.prepareCandleFinished(message.Security, message.Timeframe, candleFinishedEventName, candleFinishedEventToken)
			if err != nil {
				b.Log().Error("prepareCandleFinished %v %v %v", message.Security.Name, message.Timeframe, err)
				return nil
			}
		}
		b.Send(from, model.GetCandleFinishedResponse{
			EventName: candleFinishedEventName,
		})
	}
	return nil
}

func (b *MultyBroker) onCandleCallback(candle model.Candle) {
	var candleFinishedEventName = getCandleFinishedEventName(candle.Interval, candle.SecurityCode)
	candleFinishedEventToken, found := b.candleFinishedEvents[candleFinishedEventName]
	if !found {
		return
	}
	// TODO верификация баров
	if candle.DateTime.Add(10 * time.Minute).After(b.start) {
		b.Log().Debug("New candle %v", candle)
	}
	b.SendEvent(candleFinishedEventName, candleFinishedEventToken, candle)
}

func (b *MultyBroker) HandleCall(from gen.PID, ref gen.Ref, req any) (any, error) {
	switch req := req.(type) {
	case model.GetPortfolioLimitsRequest:
		return b.Call(b.brokers[req.Portfolio.Client], req)
	case model.GetPositionRequest:
		return b.Call(b.brokers[req.Portfolio.Client], req)
	case model.RegisterOrderRequest:
		return b.Call(b.brokers[req.Order.Portfolio.Client], req)
	}
	return gen.ErrUnsupported, nil
}

func (b *MultyBroker) Terminate(reason error) {
	b.Log().Info("terminated with reason: %s", reason)
}

func (b *MultyBroker) prepareCandleFinished(
	security model.Security,
	timeframe string,
	candleFinishedEventName gen.Atom,
	candleFinishedEventToken gen.Ref,
) error {
	var lastCandleTime time.Time

	if b.historyMarketData != nil {
		var candles, err = b.historyMarketData.Load(security, timeframe)
		if err != nil {
			return err
		}
		for _, candle := range candles {
			err := b.SendEvent(candleFinishedEventName, candleFinishedEventToken, candle)
			if err != nil {
				return err
			}
		}
		if len(candles) > 0 {
			lastCandleTime = candles[len(candles)-1].DateTime
		}
	}

	resp, err := b.Call(b.marketdata, model.GetLastCandlesRequest{
		Security:  security,
		Timeframe: timeframe,
	})
	if err != nil {
		return err
	}
	if respErr, ok := resp.(error); ok {
		return respErr
	}
	var candles = candlesAfter(resp.([]model.Candle), lastCandleTime)
	for _, candle := range candles {
		err := b.SendEvent(candleFinishedEventName, candleFinishedEventToken, candle)
		if err != nil {
			return err
		}
	}

	resp, err = b.Call(b.marketdata, model.SubscribeCandlesRequest{
		Ssecurity: security,
		Timeframe: timeframe,
	})
	if err != nil {
		return err
	}
	if respErr, ok := resp.(error); ok {
		return respErr
	}

	return nil
}

func candlesAfter(source []model.Candle, date time.Time) []model.Candle {
	for i, candle := range source {
		if candle.DateTime.After(date) {
			return source[i:]
		}
	}
	return nil
}

func getCandleFinishedEventName(interval, secCode string) gen.Atom {
	//TODO temporary fix
	if interval == "TODO" {
		interval = model.CandleIntervalMinutes5
	}
	return gen.Atom(fmt.Sprintf("candleFinished_%v_%v",
		interval, secCode))
}
