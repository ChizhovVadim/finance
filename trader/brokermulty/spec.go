package brokermulty

import (
	"finance/model"

	"ergo.services/ergo/gen"
)

type MultyBrokerSpec struct {
	HistoryMarketData IHistoryMarketData
	Clients           []ClientSpec
	MarketData        string
}

type IHistoryMarketData interface {
	Load(security model.Security, interval string) ([]model.Candle, error)
}

type ClientSpec struct {
	Name          string
	FactoryBroker gen.ProcessFactory
	Args          []any
}
