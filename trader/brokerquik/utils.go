package brokerquik

import (
	"finance/trader/model"
	"math"
	"net"
	"strconv"
	"time"
)

var timeframeCodes = map[string]int{
	model.CandleIntervalMinutes5: CandleIntervalM5,
	model.CandleIntervalHourly:   CandleIntervalH1,
	model.CandleIntervalDaily:    CandleIntervalD1,
}

func convertToCandle(item Candle) model.Candle {
	return model.Candle{
		Interval:     "TODO",
		SecurityCode: item.SecCode,
		DateTime:     item.Datetime.ToTime(model.MoexTimeZone),
		OpenPrice:    item.Open,
		HighPrice:    item.High,
		LowPrice:     item.Low,
		ClosePrice:   item.Close,
		Volume:       item.Volume,
	}
}

func dial(port int) (net.Conn, error) {
	//TODO net.JoinHostPort
	return net.Dial("tcp", "localhost:"+strconv.Itoa(port))
}

func timeToQuikTime(time time.Time) int64 {
	return time.UnixNano() / 1000
}

func calculateStartTransId() int64 {
	var hour, min, sec = time.Now().Clock()
	return 60*(60*int64(hour)+int64(min)) + int64(sec)
}

func formatPrice(priceStep float64, pricePrecision int, price float64) string {
	if priceStep != 0 {
		price = math.Round(price/priceStep) * priceStep
	}
	return strconv.FormatFloat(price, 'f', pricePrecision, 64)
}
