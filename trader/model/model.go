package model

import (
	"time"
)

type Signal struct {
	Deadline time.Time
	Price    float64
	Value    float64
}

type PlannedPosition struct {
	Deadline  time.Time
	Portfolio Portfolio
	Security  Security
	Planned   int
}
