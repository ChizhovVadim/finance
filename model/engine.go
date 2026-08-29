package model

import (
	"time"

	"ergo.services/ergo/gen"
)

type Signal struct {
	Name     string
	Deadline time.Time
	Price    float64
	Value    float64
}

type GetStatusRequest struct{}

type GetCandleFinishedRequest struct {
	Ssecurity Security
	Timeframe string
}

type GetCandleFinishedResponse struct {
	EventName gen.Atom
}
