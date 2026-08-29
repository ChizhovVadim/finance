package engine

import (
	"finance/model"
	"fmt"
)

func (eng *Engine) checkStatus() {
	var r = Report{}
	for _, signalPID := range eng.signals {
		resp, err := eng.Call(signalPID, model.GetStatusRequest{})
		if err != nil {
			return
		}
		if _, ok := resp.(error); ok {
			return
		}
		r.Signals = append(r.Signals, resp.(model.Signal))
	}
	for i := range eng.strategies {
		var strategy = &eng.strategies[i]
		resp, err := eng.Call(model.MultyBroker, model.GetPositionRequest{
			Portfolio: strategy.portfolio,
			Security:  strategy.security,
		})
		if err != nil {
			return
		}
		if _, ok := resp.(error); ok {
			return
		}
		var brokerPos = resp.(int)
		r.Positions = append(r.Positions, PoitionInfo{
			Client:    strategy.portfolio.Client,
			Portfolio: strategy.portfolio.Portfolio,
			Security:  strategy.security.Name,
			Planned:   strategy.plannedPosition,
			Actual:    brokerPos,
			Ok:        brokerPos == strategy.plannedPosition,
		})
	}
	var visitedPortfolios = make(map[string]struct{})
	for i := range eng.strategies {
		var strategy = &eng.strategies[i]
		var portfolio = strategy.portfolio
		if _, found := visitedPortfolios[portfolio.Portfolio]; found {
			continue
		}
		visitedPortfolios[portfolio.Portfolio] = struct{}{}

		resp, err := eng.Call(model.MultyBroker, model.GetPortfolioLimitsRequest{
			Portfolio: portfolio,
		})
		if err != nil {
			return
		}
		if _, ok := resp.(error); ok {
			return
		}
		limits := resp.(model.PortfolioLimits)
		var varMargin = limits.AccVarMargin + limits.VarMargin
		r.Portfolios = append(r.Portfolios, PortfolioInfo{
			Client:         portfolio.Client,
			Portfolio:      portfolio.Portfolio,
			Amount:         limits.StartLimitOpenPos,
			VarMargin:      varMargin,
			VarMarginRatio: varMargin / limits.StartLimitOpenPos,
			UsedRatio:      limits.UsedLimOpenPos / limits.StartLimitOpenPos,
		})
	}
	fmt.Print(formatReport(r))
}
