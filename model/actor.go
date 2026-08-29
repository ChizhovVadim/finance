package model

import "ergo.services/ergo/gen"

const (
	MultyBroker = gen.Atom("MultyBroker")
)

type BrokerMessageInfoRequest struct {
	Message string
}

type GetLastCandlesRequest struct {
	Security  Security
	Timeframe string
}

type SubscribeCandlesRequest struct {
	Ssecurity Security
	Timeframe string
}

type GetPortfolioLimitsRequest struct {
	Portfolio Portfolio
}

type GetPositionRequest struct {
	Portfolio Portfolio
	Security  Security
}

type RegisterOrderRequest struct {
	Order Order
}

type BrokerCallbackMessage struct {
	Message any
}
