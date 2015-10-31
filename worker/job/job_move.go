package job

import (
	"github.com/darvik80/monitor/worker"
	"os"
)

type jobMove struct {
	src string
	dst string
}

func NewMove(src, dst string) worker.IJob {
	return &jobMove{
		src: src,
		dst: dst,
	}
}

func (this jobMove) Execute() error {
	return os.Rename(this.src, this.dst)
}
