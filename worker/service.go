package worker

import (
	"github.com/darvik80/monitor/log"
)

type workerInstance struct {
	high chan IJob
	low  chan IJob
	done chan bool
}

func NewWorker() IWorker {
	service := &workerInstance{
		high: make(chan IJob, 10),
		low:  make(chan IJob, 10),
		done: make(chan bool),
	}

	go service.process()

	return service
}

func (this workerInstance) Push(job IJob, priority int) {
	switch priority {
	case Priority_High:
		this.high <- job
	case Priority_Low:
		this.low <- job
	default:
		break
	}
}

func (this workerInstance) Done() {
	this.done <- true
	<-this.done
}

func (this workerInstance) process() {
	defer close(this.done)

	var err error

	for {
		select {
		case job := <-this.high:
			err = job.Execute()
		case <-this.done:
			return
		default:
			select {
			case job := <-this.high:
				err = job.Execute()
			case job := <-this.low:
				err = job.Execute()
			case <-this.done:
				return
			}
		}

		if err != nil && err != ErrCancel {
			log.Warning(err.Error())
		}
	}
}
