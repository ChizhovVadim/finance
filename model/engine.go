package model

import (
	"time"
)

type Signal struct {
	Name     string
	Deadline time.Time
	Price    float64
	Value    float64
}

type GetStatusRequest struct{}
