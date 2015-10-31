package worker

import (
	"github.com/dropbox/godropbox/errors"
)

const (
	Priority_None = iota
	Priority_Low
	Priority_High
)

var (
	ErrCancel = errors.New("Cancel")
)

type IWorker interface {
	Push(job IJob, priority int)
	Done()
}

type IJob interface {
	Execute() error
}
