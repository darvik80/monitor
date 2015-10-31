package job

import (
	"github.com/darvik80/monitor/worker"
	"os/exec"
)

type jobCopy struct {
	src string
	dst string
}

func NewCopy(src, dst string) worker.IJob {
	return &jobCopy{
		src: src,
		dst: dst,
	}
}

func (this jobCopy) Execute() error {
	return exec.Command("cp", "-rf", this.src, this.dst).Run()
}
