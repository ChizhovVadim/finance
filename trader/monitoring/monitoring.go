package monitoring

import (
	"finance/trader/model"
	"fmt"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type shouldCheckStatusMessage struct{}

type Monitoring struct {
	act.Actor
	waitingShouldCheckStatus bool
	strategies               map[gen.PID]struct{}
}

func NewMonitoring() gen.ProcessBehavior {
	return &Monitoring{
		strategies: make(map[gen.PID]struct{}),
	}
}

func (a *Monitoring) Init(args ...any) error {
	a.Log().Info("started.")
	return nil
}

func (a *Monitoring) HandleMessage(from gen.PID, message any) error {
	switch message.(type) {
	case model.MonitoringStrategyMessage:
		a.strategies[from] = struct{}{}
		// Сообщение может прийти от многих стратегий, поэтому throttling.
		if a.waitingShouldCheckStatus {
			return nil
		}
		a.waitingShouldCheckStatus = true
		a.SendAfter(a.PID(), shouldCheckStatusMessage{}, 10*time.Second)
		return nil
	case shouldCheckStatusMessage:
		a.waitingShouldCheckStatus = false
		a.checkStatus()
	}
	return nil
}

func (a *Monitoring) checkStatus() {
	var r Report

	var visitedPortfolios = make(map[string]struct{})
	for strategy := range a.strategies {
		resp, err := a.Call(strategy, model.GetStartegyPlannedPositionRequest{})
		if err != nil {
			continue
		}
		if errResponse, ok := resp.(error); ok {
			_ = errResponse
			continue
		}
		plannedPos := resp.(model.PlannedPosition)

		resp, err = a.Call(plannedPos.Portfolio.Broker, model.GetPositionRequest{
			Portfolio: plannedPos.Portfolio,
			Security:  plannedPos.Security,
		})
		if err != nil {
			continue
		}
		if errResponse, ok := resp.(error); ok {
			_ = errResponse
			continue
		}
		var brokerPos = int(resp.(float64))

		r.Positions = append(r.Positions, PoitionInfo{
			Client:    plannedPos.Portfolio.Client,
			Portfolio: plannedPos.Portfolio.Portfolio,
			Security:  plannedPos.Security.Name,
			Planned:   plannedPos.Planned,
			Actual:    brokerPos,
			Ok:        plannedPos.Planned == brokerPos,
		})

		var portfolio = plannedPos.Portfolio
		if _, found := visitedPortfolios[portfolio.Portfolio]; found {
			continue
		}
		visitedPortfolios[portfolio.Portfolio] = struct{}{}

		resp, err = a.Call(portfolio.Broker, model.GetPortfolioLimitsRequest{
			Portfolio: portfolio,
		})
		if err != nil {
			continue
		}
		if errResponse, ok := resp.(error); ok {
			_ = errResponse
			continue
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
