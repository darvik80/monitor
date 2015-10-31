package job

import (
	"github.com/darvik80/monitor/worker"
	"strings"
)

type jobCheckExt struct {
	path      string
	extension string
}

func NewCheckExt(path, extension string) worker.IJob {
	return &jobCheckExt{
		path:      path,
		extension: extension,
	}
}

func (this jobCheckExt) Execute() error {
	if strings.HasSuffix(this.path, this.extension) {
		return nil
	}

	return worker.ErrCancel
}
