package brokerquik

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"

	"finance/internal/moex"
	"finance/model"
)

type messageTCP struct {
	Data []byte
}

type QuikBroker struct {
	act.Actor
	nextRequestId     int64
	nextTransactionId int64
	mainConnId        gen.Alias
	queriesInProgress map[int64]query
}

type query struct {
	from    gen.PID
	ref     gen.Ref
	command string
}

func FactoryQuikBroker() gen.ProcessBehavior {
	return &QuikBroker{}
}

func (b *QuikBroker) Init(args ...any) error {
	port := args[0].(int)

	b.nextRequestId = 1
	b.nextTransactionId = calculateStartTransId()
	b.queriesInProgress = make(map[int64]query)

	mainConn, err := createMainConnection(port)
	if err != nil {
		return err
	}
	mainConnId, err := b.SpawnMeta(mainConn, gen.MetaOptions{})
	if err != nil {
		mainConn.Terminate(err)
		return err
	}
	b.mainConnId = mainConnId

	callbackConn, err := createCallbackConnection(port + 1)
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
		_ = b.makeRequest(from, gen.Ref{}, RequestMessage(message.Message))
	case CallbackJson:
	case messageTCP:
		err := b.handleResponse(message)
		if err != nil {
			b.Log().Warning("handleResponse %v", err)
		}
	}
	return nil
}

func (b *QuikBroker) handleResponse(resp messageTCP) error {
	var respJson ResponseJson
	var err = json.Unmarshal(resp.Data, &respJson)
	if err != nil {
		return err
	}
	var query, ok = b.queriesInProgress[respJson.Id]
	if !ok {
		return fmt.Errorf("query not found %v", respJson.Id)
	}
	delete(b.queriesInProgress, respJson.Id)

	// Уже вернули ответ клиенту в HandleCall
	if query.ref.Node == "" {
		return nil
	}

	var msg any
	if respJson.LuaError != "" {
		msg = fmt.Errorf("lua error: %v", respJson.LuaError)
	} else {
		msg = parseResponse(respJson.Command, respJson.Data)
	}
	return b.SendResponse(query.from, query.ref, msg)
}

func newParseError(child error) error {
	return fmt.Errorf("parse error: %w", child)
}

func parseResponse(command string, data json.RawMessage) any {
	switch command {
	case "getPortfolioInfoEx":
		var m map[string]any
		var err = json.Unmarshal(data, &m)
		if err != nil {
			return newParseError(err)
		}
		if m == nil {
			return errors.New("portfolio not found")
		}
		startLimitOpenPos, err := parseFloat(m["start_limit_open_pos"])
		if err != nil {
			return newParseError(err)
		}
		usedLimOpenPos, _ := parseFloat(m["used_lim_open_pos"])
		varMargin, _ := parseFloat(m["varmargin"])
		accVarMargin, _ := parseFloat(m["fut_accured_int"])
		return model.PortfolioLimits{
			StartLimitOpenPos: startLimitOpenPos,
			UsedLimOpenPos:    usedLimOpenPos,
			VarMargin:         varMargin,
			AccVarMargin:      accVarMargin,
		}
	case "getFuturesHolding":
		var m map[string]any
		var err = json.Unmarshal(data, &m)
		if err != nil {
			return newParseError(err)
		}
		if m == nil {
			return 0
		}
		// TODO use json.Number
		pos, err := parseFloat(m["totalnet"])
		if err != nil {
			return newParseError(err)
		}
		return int(pos)
	case "get_candles_from_data_source":
		var candles []Candle
		var err = json.Unmarshal(data, &candles)
		if err != nil {
			return newParseError(err)
		}
		// последний бар за сегодня может быть не завершен
		if len(candles) > 0 &&
			isToday(candles[len(candles)-1].Datetime.ToTime(moex.TimeZone)) {
			candles = candles[:len(candles)-1]
		}
		var res = make([]model.Candle, 0, len(candles))
		for i := range candles {
			res = append(res, convertToCandle(candles[i]))
		}
		return res
	}
	return errors.New("not implemented")
}

func (b *QuikBroker) HandleCall(from gen.PID, ref gen.Ref, req any) (any, error) {
	//b.Log().Debug("received call from %s: %v", from, req)
	switch req := req.(type) {
	case model.GetPortfolioLimitsRequest:
		err := b.makeRequest(from, ref,
			RequestGetPortfolioInfoEx(req.Portfolio.Firm, req.Portfolio.Portfolio, 0))
		if err != nil {
			return err, nil
		}
		// async request
		return nil, nil
	case model.GetPositionRequest:
		if req.Security.ClassCode == moex.FuturesClassCode {
			err := b.makeRequest(from, ref,
				RequestGetFuturesHolding(req.Portfolio.Firm, req.Portfolio.Portfolio, req.Security.Code, 0))
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
		err := b.makeRequest(from, gen.Ref{},
			RequestSendTransaction(transId, order.Security.Code, order.Security.ClassCode, order.Portfolio.Portfolio, sPrice, fmt.Sprintf("%v", transId), order.Volume))
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
		const candleCount = 5_000 // Если не указывать размер, то может прийти слишком много баров и unmarshal большой json
		err := b.makeRequest(from, ref,
			RequestGetCandlesFromDataSource(req.Security.ClassCode, req.Security.Code, candleInterval, candleCount))
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
		b.makeRequest(from, gen.Ref{}, RequestSubscribeToCandles(req.Ssecurity.ClassCode, req.Ssecurity.Code, candleInterval))
		// Чтобы не заблокироваться, не ждем ответа от брокера, а сразу продолжаем работу.
		return true, nil
	}
	return gen.ErrUnsupported, nil
}

func (b *QuikBroker) Terminate(reason error) {
	b.Log().Info("terminated with reason: %s", reason)
}

func (b *QuikBroker) makeRequest(from gen.PID, ref gen.Ref, requestQuik RequestQuik) error {
	var r = RequestJson{
		Id:          b.nextRequestId,
		Command:     requestQuik.Command,
		Data:        requestQuik.Data,
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
		from:    from,
		ref:     ref,
		command: r.Command,
	}
	return nil
}
