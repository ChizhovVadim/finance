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

type PlannedPosition struct {
	Portfolio Portfolio
	Security  Security
	Planned   int
}

type MonitoringRequest struct{}

type GetStatusRequest struct{}

type GetCandleFinishedRequest struct {
	Security  Security
	Timeframe string
}

type GetCandleFinishedResponse struct {
	EventName gen.Atom
}
