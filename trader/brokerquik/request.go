package brokerquik

import (
	"fmt"
	"strconv"
)

type RequestQuik struct {
	Command string
	Data    any
}

func RequestMessage(msg string) RequestQuik {
	return RequestQuik{
		Command: "message",
		Data:    msg,
	}
}

func RequestGetPortfolioInfoEx(
	firmId string,
	clientCode string,
	limitKind int,
) RequestQuik {
	return RequestQuik{
		Command: "getPortfolioInfoEx",
		Data:    fmt.Sprintf("%v|%v|%v", firmId, clientCode, limitKind),
	}
}

func RequestGetFuturesHolding(
	firmId string,
	accId string,
	secCode string,
	posType int,
) RequestQuik {
	return RequestQuik{
		Command: "getFuturesHolding",
		Data:    fmt.Sprintf("%v|%v|%v|%v", firmId, accId, secCode, posType),
	}
}

func RequestSendTransaction(
	transId int64,
	secCode string,
	classCode string,
	account string,
	price string,
	clientCode string,
	quantity int,
) RequestQuik {
	type Transaction struct {
		TRANS_ID    string
		ACTION      string
		ACCOUNT     string
		CLASSCODE   string
		SECCODE     string
		QUANTITY    string
		OPERATION   string
		PRICE       string
		CLIENT_CODE string
	}
	var data = Transaction{
		TRANS_ID:    fmt.Sprintf("%v", transId),
		ACTION:      "NEW_ORDER",
		ACCOUNT:     account,
		CLASSCODE:   classCode,
		SECCODE:     secCode,
		PRICE:       price,
		CLIENT_CODE: clientCode,
	}
	if quantity > 0 {
		data.OPERATION = "B"
		data.QUANTITY = strconv.Itoa(quantity)
	} else {
		data.OPERATION = "S"
		data.QUANTITY = strconv.Itoa(-quantity)
	}
	return RequestQuik{
		Command: "sendTransaction",
		Data:    data,
	}
}

func RequestGetCandlesFromDataSource(
	classCode string,
	securityCode string,
	interval int,
	count int,
) RequestQuik {
	return RequestQuik{
		Command: "get_candles_from_data_source",
		Data:    fmt.Sprintf("%v|%v|%v|%v", classCode, securityCode, interval, count),
	}
}

func RequestSubscribeToCandles(
	classCode string,
	securityCode string,
	interval int,
) RequestQuik {
	return RequestQuik{
		Command: "subscribe_to_candles",
		Data:    fmt.Sprintf("%v|%v|%v", classCode, securityCode, interval),
	}
}
