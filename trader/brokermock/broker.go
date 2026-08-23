package brokermock

import (
	"finance/trader/model"
	"fmt"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type MockBroker struct {
	act.Actor
	positions            map[string]int
	candleFinishedEvents map[gen.Atom]gen.Ref
}

func FactoryMockBroker() gen.ProcessBehavior {
	return &MockBroker{
		positions:            make(map[string]int),
		candleFinishedEvents: make(map[gen.Atom]gen.Ref),
	}
}

func (b *MockBroker) Init(args ...any) error {
	b.Log().Info("started")
	return nil
}

func (b *MockBroker) HandleMessage(from gen.PID, message any) error {
	switch message := message.(type) {
	case model.BrokerMessageInfoRequest:
		fmt.Println(message.Message)
	}
	return nil
}

func (b *MockBroker) HandleCall(from gen.PID, ref gen.Ref, req any) (any, error) {
	b.Log().Debug("received call from %s: %v", from, req)
	switch req := req.(type) {
	case model.GetPortfolioLimitsRequest:
		return model.PortfolioLimits{
			StartLimitOpenPos: 1_000_000,
		}, nil
	case model.GetPositionRequest:
		var pos = b.positions[positionKey(req.Portfolio, req.Security)]
		return float64(pos), nil
	case model.RegisterOrderRequest:
		var order = req.Order
		b.Log().Info("RegisterOrder client: %v portfolio: %v security: %v quantity: %v price: %v",
			order.Portfolio.Client, order.Portfolio.Portfolio, order.Security.Name, order.Volume, order.Price)
		b.positions[positionKey(order.Portfolio, order.Security)] += order.Volume
		return true, nil
	case model.GetLastCandlesRequest:
		return []model.Candle{}, nil
	case model.GetCandleFinishedEvent:
		var candleFinishedEventName = b.getCandleFinishedEventName(req.Timeframe, req.Ssecurity.Code)
		if _, ok := b.candleFinishedEvents[candleFinishedEventName]; !ok {
			candleFinishedEventToken, err := b.RegisterEvent(candleFinishedEventName, gen.EventOptions{})
			if err != nil {
				return err, nil
			}
			b.candleFinishedEvents[candleFinishedEventName] = candleFinishedEventToken
		}
		return candleFinishedEventName, nil
	}
	return gen.ErrUnsupported, nil
}

func (b *MockBroker) Terminate(reason error) {
	b.Log().Info("terminated with reason: %s", reason)
}

func (b *MockBroker) getCandleFinishedEventName(interval, secCode string) gen.Atom {
	return gen.Atom(fmt.Sprintf("candleFinished_%v_%v_%v", b.Name(), interval, secCode))
}

func positionKey(portfolio model.Portfolio, security model.Security) string {
	return portfolio.Portfolio + security.Code
}
