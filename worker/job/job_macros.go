package job

import (
	"github.com/darvik80/monitor/log"
	"github.com/darvik80/monitor/worker"
	"reflect"
)

type jobMacros struct {
	jobs []worker.IJob
}

func NewMacros(jobs ...worker.IJob) worker.IJob {
	return jobMacros{
		jobs: append([]worker.IJob{}, jobs...),
	}
}

func (this jobMacros) Execute() error {
	for _, job := range this.jobs {
		log.Debugf("Execute: %v", reflect.TypeOf(job))
		if err := job.Execute(); err != nil {
			return err
		}
	}

	return nil
}
