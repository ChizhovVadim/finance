package engine

import (
	"finance/model"
	"time"
)

type StrategyService struct {
	signalName      string
	security        model.Security
	portfolio       model.Portfolio
	sizePolicy      SizePolicy
	amountAvailable float64
	// планируемая позиция в портфеле (после исполнения заявки)
	plannedPosition int
	basePrice       model.Signal
}

func (eng *Engine) strategyInitAmount(strategy *StrategyService) error {
	resp, err := eng.Call(model.MultyBroker, model.GetPortfolioLimitsRequest{
		Portfolio: strategy.portfolio,
	})
	if err != nil {
		return err
	}
	if respErr, ok := resp.(error); ok {
		return respErr
	}
	limits := resp.(model.PortfolioLimits)

	strategy.amountAvailable = limits.StartLimitOpenPos
	if strategy.sizePolicy.MaxAmount != 0 {
		strategy.amountAvailable = min(strategy.amountAvailable, strategy.sizePolicy.MaxAmount)
	}

	eng.Log().Info("Init amount client: %v account: %v startLimit: %v available: %v",
		strategy.portfolio.Client,
		strategy.portfolio.Portfolio,
		limits.StartLimitOpenPos,
		strategy.amountAvailable)
	return nil
}

func (eng *Engine) strategyInitPosition(strategy *StrategyService) error {
	resp, err := eng.Call(model.MultyBroker, model.GetPositionRequest{
		Portfolio: strategy.portfolio,
		Security:  strategy.security,
	})
	if err != nil {
		return err
	}
	if respErr, ok := resp.(error); ok {
		return respErr
	}
	brokerPos := resp.(int)

	strategy.plannedPosition = brokerPos
	eng.Log().Info("Init position client: %v account: %v security: %v position: %v",
		strategy.portfolio.Client,
		strategy.portfolio.Portfolio,
		strategy.security.Name,
		brokerPos)
	return nil
}

func (eng *Engine) onSignal(strategy *StrategyService, signal model.Signal) {
	// стратегия следит только за своими сигналами
	if !(signal.Name == strategy.signalName) {
		return
	}
	// считаем, что сигнал слишком старый
	if signal.Deadline.Before(time.Now()) {
		return
	}
	if strategy.basePrice.Deadline.IsZero() {
		strategy.basePrice = signal
		eng.Log().Debug("Init base price %v", strategy.basePrice)
	}
	var idealPos = calcIdealPos(strategy.amountAvailable, signal.Value, strategy.sizePolicy, strategy.security, strategy.basePrice.Price)
	var volume = int(idealPos - float64(strategy.plannedPosition))
	// изменение позиции не требуется
	if volume == 0 {
		return
	}
	var expectedBrokerPos = strategy.plannedPosition
	strategy.plannedPosition += volume

	// Сообщение может прийти от многих стратегий, поэтому throttling.
	if !eng.waitingShouldCheckStatus {
		eng.waitingShouldCheckStatus = true
		eng.SendAfter(eng.PID(), shouldCheckStatus{}, 10*time.Second)
	}
	_ = expectedBrokerPos
	eng.Call(model.MultyBroker, model.RegisterOrderRequest{
		Order: model.Order{
			Portfolio: strategy.portfolio,
			Security:  strategy.security,
			Volume:    volume,
			Price:     priceWithSlippage(signal.Price, volume),
		},
	})
}

func calcIdealPos(
	amount float64,
	prediction float64,
	sizePolicy SizePolicy,
	security model.Security,
	price float64,
) float64 {
	var pos = prediction
	if pos > 0 {
		pos *= sizePolicy.LongLever
	} else {
		pos *= sizePolicy.ShortLever
	}
	pos = max(-sizePolicy.MaxLever, min(sizePolicy.MaxLever, pos))
	return amount * sizePolicy.Weight * pos / (price * security.Lever)
}

func priceWithSlippage(price float64, volume int) float64 {
	const Slippage = 0.001
	if volume > 0 {
		return price * (1 + Slippage)
	} else {
		return price * (1 - Slippage)
	}
}
