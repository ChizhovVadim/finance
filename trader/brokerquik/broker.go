package brokerquik

import (
	"encoding/json"
	"finance/trader/model"
	"fmt"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

const CandleFinishedEventName = gen.Atom("candleFinished")

type messageTCP struct {
	Data []byte
}

type QuikBroker struct {
	act.Actor
	port                     int
	nextRequestId            int64
	nextTransactionId        int64
	candleFinishedEventToken gen.Ref
	mainConnId               gen.Alias
}

func NewQuikBroker(
	port int,
) gen.ProcessBehavior {
	return &QuikBroker{
		port:              port,
		nextRequestId:     1,
		nextTransactionId: calculateStartTransId(),
	}
}

func (b *QuikBroker) Init(args ...any) error {
	candleFinishedEventToken, err := b.RegisterEvent(CandleFinishedEventName, gen.EventOptions{})
	if err != nil {
		return err
	}
	b.candleFinishedEventToken = candleFinishedEventToken

	mainConn, err := createMainConnection(b.port)
	if err != nil {
		return err
	}
	mainConnId, err := b.SpawnMeta(mainConn, gen.MetaOptions{})
	if err != nil {
		mainConn.Terminate(err)
		return err
	}
	b.mainConnId = mainConnId

	callbackConn, err := createCallbackConnection(b.port + 1)
	if err != nil {
		return err
	}
	callbackConnId, err := b.SpawnMeta(callbackConn, gen.MetaOptions{})
	if err != nil {
		callbackConn.Terminate(err)
		return err
	}

	b.Log().Info("started (meta-process: %s %s)",
		mainConnId, callbackConnId)
	return nil
}

func (b *QuikBroker) HandleMessage(from gen.PID, message any) error {
	switch message := message.(type) {
	case model.BrokerMessageInfoRequest:
		b.makeRequest("message", message.Message)
	case CallbackJson:
		b.handleCallback(message)
	case messageTCP:
		b.handleResponse(message)
	}
	return nil
}

func (b *QuikBroker) handleResponse(resp messageTCP) {}

func (b *QuikBroker) handleCallback(cj CallbackJson) {
	if cj.Command == "NewCandle" {
		if cj.Data != nil {
			var newCandle Candle
			var err = json.Unmarshal(*cj.Data, &newCandle)
			if err != nil {
				return //err
			}
			// TODO можно фильтровать слишком ранние бары
			b.SendEvent(CandleFinishedEventName, b.candleFinishedEventToken, model.CandleFinished{
				Candle: convertToCandle(newCandle),
			})
		}
		return
	}
}

func (b *QuikBroker) HandleCall(from gen.PID, ref gen.Ref, req any) (any, error) {
	b.Log().Debug("received call from %s: %v", from, req)
	switch req := req.(type) {
	case model.GetPortfolioLimitsRequest:
		b.makeRequest("getPortfolioInfoEx",
			fmt.Sprintf("%v|%v|%v",
				req.Portfolio.Firm, req.Portfolio.Portfolio, 0))
		// async request
		return nil, nil
	case model.GetPositionRequest:
		if req.Security.ClassCode == model.FuturesClassCode {
			b.makeRequest("getFuturesHolding",
				fmt.Sprintf("%v|%v|%v|%v",
					req.Portfolio.Firm, req.Portfolio.Portfolio, req.Security.Code, 0))
			// async request
			return nil, nil
		} else {
			return fmt.Errorf("not supported classcode %v", req.Security.ClassCode), nil
		}
	case model.RegisterOrderRequest:
		var order = req.Order
		var sPrice = formatPrice(order.Security.PriceStep, order.Security.PricePrecision, order.Price)
		b.Log().Info("RegisterOrder client: %v portfolio: %v security: %v quantity: %v price: %v",
			order.Portfolio.Client, order.Portfolio.Portfolio, order.Security.Name, order.Volume, sPrice)
		// Чтобы не заблокироваться, не ждем ответа от брокера, а сразу продолжаем работу.
		return true, nil
	case model.GetLastCandlesRequest:
		return fmt.Errorf("not implemented"), nil
	case model.SubscribeCandlesRequest:
		var candleInterval, ok = timeframeCodes[req.Timeframe]
		if !ok {
			return fmt.Errorf("timeframe not supported %v", req.Timeframe), nil
		}
		b.makeRequest("subscribe_to_candles",
			fmt.Sprintf("%v|%v|%v",
				req.Ssecurity.ClassCode, req.Ssecurity.Code, candleInterval))
		// Чтобы не заблокироваться, не ждем ответа от брокера, а сразу продолжаем работу.
		return true, nil
	}
	return gen.ErrUnsupported, nil
}

func (b *QuikBroker) Terminate(reason error) {
	b.Log().Info("terminated with reason: %s", reason)
}

func (b *QuikBroker) makeRequest(cmd string, data any) {
	var r = RequestJson{
		Id:          b.nextRequestId,
		Command:     cmd,
		Data:        data,
		CreatedTime: timeToQuikTime(time.Now()),
	}
	b.nextRequestId += 1

	bytes, err := json.Marshal(r)
	if err != nil {
		return
	}
	b.Send(b.mainConnId, messageTCP{Data: bytes})
}
