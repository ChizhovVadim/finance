package model

import "ergo.services/ergo/gen"

type Portfolio struct {
	Client    string
	Broker    gen.Atom
	Firm      string
	Portfolio string
}

type Security struct {
	// Название инструмента
	Name string
	// Код инструмента
	Code string
	// Код класса
	ClassCode string
	// точность (кол-во знаков после запятой). Если шаг цены может быть не круглым (0.05), то этого будет недостаточно.
	PricePrecision int
	// шаг цены
	PriceStep float64
	// Стоимость шага цены
	PriceStepCost float64
	// Плечо. Для фьючерсов = PriceStepCost/PriceStep.
	Lever float64
}

type Order struct {
	Portfolio Portfolio
	Security  Security
	Volume    int
	Price     float64
}

type PortfolioLimits struct {
	// Лимит открытых позиций на начало дня
	StartLimitOpenPos float64
	// Текущие чистые позиции
	UsedLimOpenPos float64
	// Вариац. маржа
	VarMargin float64
	// Накопленная вариационная маржа
	AccVarMargin float64
}

type BrokerMessageInfoRequest struct {
	Message string
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

type GetLastCandlesRequest struct {
	Ssecurity Security
	Timeframe string
}

type SubscribeCandlesRequest struct {
	Ssecurity Security
	Timeframe string
}

type CandleFinished struct {
	Candle Candle
}
