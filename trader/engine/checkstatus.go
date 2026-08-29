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
	var plannedPositions []model.PlannedPosition
	for _, strategyPID := range eng.strategies {
		resp, err := eng.Call(strategyPID, model.GetStatusRequest{})
		if err != nil {
			return
		}
		if _, ok := resp.(error); ok {
			return
		}
		plannedPositions = append(plannedPositions, resp.(model.PlannedPosition))
	}
	for _, pos := range plannedPositions {
		resp, err := eng.Call(model.MultyBroker, model.GetPositionRequest{
			Portfolio: pos.Portfolio,
			Security:  pos.Security,
		})
		if err != nil {
			return
		}
		if _, ok := resp.(error); ok {
			return
		}
		var brokerPos = resp.(int)
		r.Positions = append(r.Positions, PoitionInfo{
			Client:    pos.Portfolio.Client,
			Portfolio: pos.Portfolio.Portfolio,
			Security:  pos.Security.Name,
			Planned:   pos.Planned,
			Actual:    brokerPos,
			Ok:        brokerPos == pos.Planned,
		})
	}
	var visitedPortfolios = make(map[string]struct{})
	for _, pos := range plannedPositions {
		var portfolio = pos.Portfolio
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
