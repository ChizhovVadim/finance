package strategy

import (
	"finance/model"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type Strategy struct {
	act.Actor
	signalName      string
	security        model.Security
	portfolio       model.Portfolio
	sizePolicy      SizePolicy
	amountAvailable float64
	// планируемая позиция в портфеле (после исполнения заявки)
	plannedPosition int
	basePrice       model.Signal
}

func FactoryStrategy() gen.ProcessBehavior {
	return &Strategy{}
}

func (s *Strategy) Init(args ...any) error {
	spec := args[0].(StrategySpec)

	s.signalName = spec.SignalName
	s.security = spec.Security
	s.portfolio = spec.Portfolio
	s.sizePolicy = spec.SizePolicy

	if err := s.initAmount(); err != nil {
		return err
	}
	if err := s.initPosition(); err != nil {
		return err
	}
	if err := s.initSignal(); err != nil {
		return err
	}

	s.Log().Info("started signal: %v client: %v account: %v security: %v amountAvailable: %v position: %v",
		s.signalName,
		s.portfolio.Client,
		s.portfolio.Portfolio,
		s.security.Name,
		s.amountAvailable,
		s.plannedPosition,
	)
	return nil
}

func (s *Strategy) initAmount() error {
	resp, err := s.Call(model.MultyBroker, model.GetPortfolioLimitsRequest{
		Portfolio: s.portfolio,
	})
	if err != nil {
		return err
	}
	if respErr, ok := resp.(error); ok {
		return respErr
	}
	limits := resp.(model.PortfolioLimits)

	s.amountAvailable = limits.StartLimitOpenPos
	if s.sizePolicy.MaxAmount != 0 {
		s.amountAvailable = min(s.amountAvailable, s.sizePolicy.MaxAmount)
	}
	return nil
}

func (s *Strategy) initPosition() error {
	resp, err := s.Call(model.MultyBroker, model.GetPositionRequest{
		Portfolio: s.portfolio,
		Security:  s.security,
	})
	if err != nil {
		return err
	}
	if respErr, ok := resp.(error); ok {
		return respErr
	}
	s.plannedPosition = resp.(int)
	return nil
}

func (s *Strategy) initSignal() error {
	// хотим проверить запуск стратегии без исполнения сигналов
	if s.signalName == "" {
		return nil
	}

	events, err := s.MonitorEvent(gen.Event{
		Name: gen.Atom(s.signalName),
	})
	if err != nil {
		return err
	}

	// возможно безопаснее вначале убедиться, что лимиты загрузились верные и не исполнять сразу текущий сигнал
	_ = events

	// if len(events) > 0 {
	// 	s.HandleEvent(events[len(events)-1])
	// }
	return nil
}

func (s *Strategy) HandleMessage(from gen.PID, message any) error {
	return nil
}

func (s *Strategy) HandleCall(from gen.PID, ref gen.Ref, req any) (any, error) {
	switch req.(type) {
	case model.GetStatusRequest:
		return model.PlannedPosition{
			Portfolio: s.portfolio,
			Security:  s.security,
			Planned:   s.plannedPosition,
		}, nil
	}
	return gen.ErrUnsupported, nil
}

func (s *Strategy) HandleEvent(event gen.MessageEvent) error {
	switch message := event.Message.(type) {
	case model.Signal:
		s.onSignal(message)
		return nil
	}
	return nil
}

func (s *Strategy) onSignal(signal model.Signal) {
	// считаем, что сигнал слишком старый
	if signal.Deadline.Before(time.Now()) {
		return
	}
	if s.basePrice.Deadline.IsZero() {
		s.basePrice = signal
		s.Log().Debug("Init base price %v", s.basePrice)
	}
	var idealPos = calcIdealPos(s.amountAvailable, signal.Value, s.sizePolicy, s.security, s.basePrice.Price)
	var volume = int(idealPos - float64(s.plannedPosition))
	// изменение позиции не требуется
	if volume == 0 {
		return
	}

	var expectedBrokerPos = s.plannedPosition
	s.plannedPosition += volume
	s.Send(s.Parent(), model.MonitoringRequest{})
	if !s.checkBrokerPos(expectedBrokerPos) {
		// Ничего не делаем. Зовем старшего.
		s.Log().Warning("check position failed")
		return
	}
	s.Call(model.MultyBroker, model.RegisterOrderRequest{
		Order: model.Order{
			Portfolio: s.portfolio,
			Security:  s.security,
			Volume:    volume,
			Price:     priceWithSlippage(signal.Price, volume),
		},
	})
}

func (s *Strategy) checkBrokerPos(expected int) bool {
	resp, err := s.Call(model.MultyBroker, model.GetPositionRequest{
		Portfolio: s.portfolio,
		Security:  s.security,
	})
	if err != nil {
		return false
	}
	if respError, ok := resp.(error); ok {
		_ = respError
		return false
	}
	var brokerPos = resp.(int)
	return brokerPos == expected
}

func (s *Strategy) Terminate(reason error) {
	s.Log().Info("terminated with reason: %s", reason)
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
