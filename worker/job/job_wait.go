package job

import (
	"github.com/darvik80/monitor/worker"
	"time"
)

type jobWait struct {
	delay time.Duration
}

func NewWait(delay time.Duration) worker.IJob {
	return &jobWait{
		delay: delay,
	}
}

func (this jobWait) Execute() error {
	time.Sleep(this.delay)
	return nil
}
