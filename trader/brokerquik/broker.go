package brokerquik

import (
	"encoding/json"
	"finance/trader/model"
	"fmt"
	"strconv"
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
	queriesInProgress        map[int64]query
}

type query struct {
	from gen.PID
	ref  gen.Ref
}

func NewQuikBroker(
	port int,
) gen.ProcessBehavior {
	return &QuikBroker{
		port:              port,
		nextRequestId:     1,
		nextTransactionId: calculateStartTransId(),
		queriesInProgress: make(map[int64]query),
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
		_ = b.makeRequest(from, gen.Ref{}, "message", message.Message)
	case CallbackJson:
		b.handleCallback(message)
	case messageTCP:
		err := b.handleResponse(message)
		if err != nil {
			b.Log().Warning("handleResponse %v", err)
		}
	}
	return nil
}

func (b *QuikBroker) handleResponse(resp messageTCP) error {
	var respJson ResponseJson2
	var err = json.Unmarshal(resp.Data, &respJson)
	if err != nil {
		return err
	}
	var query, ok = b.queriesInProgress[respJson.Id]
	if !ok {
		return fmt.Errorf("query not found %v", respJson.Id)
	}
	_ = query
	// TODO b.SendResponse()
	delete(b.queriesInProgress, respJson.Id)
	return nil
}

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
		err := b.makeRequest(from, ref, "getPortfolioInfoEx",
			fmt.Sprintf("%v|%v|%v",
				req.Portfolio.Firm, req.Portfolio.Portfolio, 0))
		if err != nil {
			return err, nil
		}
		// async request
		return nil, nil
	case model.GetPositionRequest:
		if req.Security.ClassCode == model.FuturesClassCode {
			err := b.makeRequest(from, ref, "getFuturesHolding",
				fmt.Sprintf("%v|%v|%v|%v",
					req.Portfolio.Firm, req.Portfolio.Portfolio, req.Security.Code, 0))
			if err != nil {
				return err, nil
			}
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
		var transId = b.nextRequestId
		b.nextTransactionId += 1
		var trans = map[string]string{
			"TRANS_ID":    fmt.Sprintf("%v", transId),
			"ACTION":      "NEW_ORDER",
			"SECCODE":     order.Security.Code,
			"CLASSCODE":   order.Security.ClassCode,
			"ACCOUNT":     order.Portfolio.Portfolio,
			"PRICE":       sPrice,
			"CLIENT_CODE": fmt.Sprintf("%v", transId),
		}
		if order.Volume > 0 {
			trans["OPERATION"] = "B"
			trans["QUANTITY"] = strconv.Itoa(order.Volume)
		} else {
			trans["OPERATION"] = "S"
			trans["QUANTITY"] = strconv.Itoa(-order.Volume)
		}
		err := b.makeRequest(from, gen.Ref{}, "sendTransaction", trans)
		if err != nil {
			return err, nil
		}
		// Чтобы не заблокироваться, не ждем ответа от брокера, а сразу продолжаем работу.
		return true, nil
	case model.GetLastCandlesRequest:
		var candleInterval, ok = timeframeCodes[req.Timeframe]
		if !ok {
			return fmt.Errorf("timeframe not supported %v", req.Timeframe), nil
		}
		err := b.makeRequest(from, ref, "get_candles_from_data_source",
			fmt.Sprintf("%v|%v|%v|%v", req.Ssecurity.ClassCode, req.Ssecurity.Code, candleInterval, 5_000))
		if err != nil {
			return err, nil
		}
		// async request
		return nil, nil
	case model.SubscribeCandlesRequest:
		var candleInterval, ok = timeframeCodes[req.Timeframe]
		if !ok {
			return fmt.Errorf("timeframe not supported %v", req.Timeframe), nil
		}
		err := b.makeRequest(from, gen.Ref{}, "subscribe_to_candles",
			fmt.Sprintf("%v|%v|%v",
				req.Ssecurity.ClassCode, req.Ssecurity.Code, candleInterval))
		if err != nil {
			return err, nil
		}
		// Чтобы не заблокироваться, не ждем ответа от брокера, а сразу продолжаем работу.
		return true, nil
	}
	return gen.ErrUnsupported, nil
}

func (b *QuikBroker) Terminate(reason error) {
	b.Log().Info("terminated with reason: %s", reason)
}

func (b *QuikBroker) makeRequest(from gen.PID, ref gen.Ref, cmd string, data any) error {
	var r = RequestJson{
		Id:          b.nextRequestId,
		Command:     cmd,
		Data:        data,
		CreatedTime: timeToQuikTime(time.Now()),
	}
	b.nextRequestId += 1

	bytes, err := json.Marshal(r)
	if err != nil {
		return err
	}
	err = b.Send(b.mainConnId, messageTCP{Data: bytes})
	if err != nil {
		return err
	}
	b.queriesInProgress[r.Id] = query{
		from: from,
		ref:  ref,
	}
	return nil
}
