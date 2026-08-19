package monitoring

import (
	"finance/trader/model"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

type shouldCheckStatus struct{}

type Monitoring struct {
	act.Actor
	waitingShouldCheckStatus bool
}

func NewMonitoring() gen.ProcessBehavior {
	return &Monitoring{}
}

func (a *Monitoring) Init(args ...any) error {
	a.Log().Info("started.")
	return nil
}

func (a *Monitoring) HandleMessage(from gen.PID, message any) error {
	switch message.(type) {
	case model.CheckStatusMessage:
		// Сообщение может прийти от многих стратегий, поэтому throttling.
		if a.waitingShouldCheckStatus {
			return nil
		}
		a.waitingShouldCheckStatus = true
		a.SendAfter(a.PID(), shouldCheckStatus{}, 10*time.Second)
		return nil
	case shouldCheckStatus:
		a.waitingShouldCheckStatus = false
		a.checkStatus()
	}
	return nil
}

func (a *Monitoring) checkStatus() {
	a.Log().Info("TODO checkStatus")
}
