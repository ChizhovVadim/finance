package brokermock

import (
	"finance/trader/model"
	"fmt"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type MockBroker struct {
	act.Actor
	positions map[string]int
}

func FactoryMockBroker() gen.ProcessBehavior {
	return &MockBroker{
		positions: make(map[string]int),
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
	}
	return gen.ErrUnsupported, nil
}

func (b *MockBroker) Terminate(reason error) {
	b.Log().Info("terminated with reason: %s", reason)
}

func positionKey(portfolio model.Portfolio, security model.Security) string {
	return portfolio.Portfolio + security.Code
}
