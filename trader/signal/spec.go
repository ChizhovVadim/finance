package signal

import (
	"finance/model"
	"time"
)

type SignalSpec struct {
	Name           string
	Security       model.Security
	CandleInterval string
	AdvisorFactory func() (IAdvisor, error)
}

type IAdvisor interface {
	Add(dt time.Time, price float64) (prediction float64, ok bool)
}
