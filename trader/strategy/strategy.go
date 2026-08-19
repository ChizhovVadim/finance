package strategy

import (
	"finance/trader/model"
	"fmt"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type SizePolicy struct {
	LongLever  float64
	ShortLever float64
	MaxLever   float64
	Weight     float64
	MaxAmount  float64
}

type Strategy struct {
	act.Actor
	signalName      string
	security        model.Security
	portfolio       model.Portfolio
	sizePolicy      SizePolicy
	broker          gen.Atom
	amountAvailable float64
	plannedPosition int
	basePrice       model.Signal
}

func NewStrategy(
	signalName string,
	security model.Security,
	portfolio model.Portfolio,
	sizePolicy SizePolicy,
	broker gen.Atom,
) gen.ProcessBehavior {
	return &Strategy{
		signalName: signalName,
		security:   security,
		portfolio:  portfolio,
		sizePolicy: sizePolicy,
		broker:     broker,
	}
}

func (s *Strategy) Init(args ...any) error {
	// init amountAvailable
	resp, err := s.Call(s.broker, model.GetPortfolioLimitsRequest{
		Portfolio: s.portfolio,
	})
	if err != nil {
		return err
	}
	if errResponse, ok := resp.(error); ok {
		return errResponse
	}
	limits := resp.(model.PortfolioLimits)
	s.amountAvailable = limits.StartLimitOpenPos
	if s.sizePolicy.MaxAmount != 0 {
		s.amountAvailable = min(s.amountAvailable, s.sizePolicy.MaxAmount)
	}

	// init broker position
	resp, err = s.Call(gen.Atom("broker"), model.GetPositionRequest{
		Portfolio: s.portfolio,
		Security:  s.security,
	})
	if err != nil {
		return err
	}
	if errResponse, ok := resp.(error); ok {
		return errResponse
	}
	s.plannedPosition = int(resp.(float64))

	// subscribe
	_, err = s.MonitorEvent(gen.Event{
		Name: gen.Atom("signalUpdated"),
		Node: s.Node().Name(),
	})
	if err != nil {
		return err
	}

	s.Log().Info("started. amount: %v amountAvailable: %v position: %v",
		limits.StartLimitOpenPos, s.amountAvailable, s.plannedPosition)
	return nil
}

func (s *Strategy) HandleEvent(event gen.MessageEvent) error {
	switch message := event.Message.(type) {
	case model.SignalUpdated:
		s.onSignal(message.Signal)
		return nil
	}
	return nil
}

func (s *Strategy) onSignal(signal model.Signal) {
	// стратегия следит только за своими сигналами
	if !(signal.Name == s.signalName) {
		return
	}
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
	s.Send(gen.Atom("monitoring"), model.CheckStatusMessage{})
	_ = expectedBrokerPos
	s.Call(s.broker, model.RegisterOrderRequest{
		Order: model.Order{
			Portfolio: s.portfolio,
			Security:  s.security,
			Volume:    volume,
			Price:     priceWithSlippage(signal.Price, volume),
		},
	})
}

func (s *Strategy) HandleInspect(from gen.PID, item ...string) map[string]string {
	return map[string]string{
		"client":          s.portfolio.Client,
		"portfolio":       s.portfolio.Portfolio,
		"security":        s.security.Name,
		"plannedPosition": fmt.Sprintf("%v", s.plannedPosition),
	}
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
