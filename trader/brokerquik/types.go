package brokerquik

import (
	"encoding/json"
	"time"
)

type RequestJson struct {
	Id          int64  `json:"id"`
	Command     string `json:"cmd"`
	CreatedTime int64  `json:"t"`
	Data        any    `json:"data"`
}

type ResponseJson struct {
	Id          int64   `json:"id"`
	Command     string  `json:"cmd"`
	CreatedTime float64 `json:"t"`
	Data        any     `json:"data"`
	LuaError    string  `json:"lua_error"`
}

type ResponseJson2 struct {
	Id          int64            `json:"id"`
	Command     string           `json:"cmd"`
	CreatedTime float64          `json:"t"`
	Data        *json.RawMessage `json:"data"`
	LuaError    string           `json:"lua_error"`
}

type CallbackJson struct {
	Command     string           `json:"cmd"`
	CreatedTime float64          `json:"t"`
	Data        *json.RawMessage `json:"data"`
	LuaError    string           `json:"lua_error"`
}

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

const (
	CandleIntervalM1 = 1
	CandleIntervalM5 = 5
	CandleIntervalH1 = 60
	CandleIntervalD1 = 1440
)

type Candle struct {
	Low       float64      `json:"low"`
	Close     float64      `json:"close"`
	High      float64      `json:"high"`
	Open      float64      `json:"open"`
	Volume    float64      `json:"volume"`
	Datetime  QuikDateTime `json:"datetime"`
	SecCode   string       `json:"sec"`
	ClassCode string       `json:"class"`
	Interval  int          `json:"interval"`
}

type QuikDateTime struct {
	Ms    int `json:"ms"`
	Sec   int `json:"sec"`
	Min   int `json:"min"`
	Hour  int `json:"hour"`
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
}

func (t *QuikDateTime) ToTime(loc *time.Location) time.Time {
	return time.Date(t.Year, time.Month(t.Month), t.Day, t.Hour, t.Min, t.Sec, 0, loc)
}
