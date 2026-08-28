package model

type BrokerMessageInfoRequest struct {
	Message string
}

type GetLastCandlesRequest struct {
	Ssecurity Security
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
