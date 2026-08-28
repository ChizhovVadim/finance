package brokerquik

import (
	"errors"
	"finance/internal/moex"
	"finance/model"
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
		DateTime:     item.Datetime.ToTime(moex.TimeZone),
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

func isToday(d time.Time) bool {
	var y1, m1, d1 = d.Date()
	var y2, m2, d2 = time.Now().Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func parseFloat(a any) (float64, error) {
	switch v := a.(type) {
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, errors.New("unknown type")
	}
}
