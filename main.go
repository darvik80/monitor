package main

import (
	"github.com/darvik80/monitor/config"
	"github.com/darvik80/monitor/fsevents"
	"github.com/darvik80/monitor/log"
	"github.com/darvik80/monitor/worker"
	"github.com/darvik80/monitor/worker/job"
	"github.com/darvik80/daemon"
	"path/filepath"
	"time"
	"os"
	"os/signal"
	"syscall"
	"sync"
	"strings"
)

const (
	// name of the service
	name        = "torrent-monitor"
	description = "Torrent monitor"
)

//  dependencies that are NOT required by the service, but might be used
var dependencies = []string{}

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage(cfg *config.Config) (string, error) {

	usage := "Usage: myservice install | remove | start | stop | status"

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			if !strings.HasPrefix(command, "logfile=") {
				return usage, nil
			}
		}
	}

	run(cfg)
	return "", nil
}

func run(cfg *config.Config) {
	workerInst := worker.NewWorker()

	done := make(chan bool)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func () {
		defer func() {
			workerInst.Done()
			close(done)
			log.Info("Stop watch")
			wg.Done()
		}()

		killSignal, data := <-interrupt
		if data {
			log.Warningf("Got signal: %s", killSignal)
		} else {
			log.Warning("Kill self")
		}
	}()

	go func() {
		defer func() {
			log.Info("Stop monitoring")
			close(interrupt)
			wg.Done()
		}()

		log.Info("Start watch")
		watcher, err := fsevents.NewWatcher()
		if err != nil {
			log.Errorf("Create watcher failed: %s", err.Error())
			return
		}
		if events, err := watcher.Watch(cfg.WatchDir, done); err != nil {
			log.Errorf("Monitoring failed: %s", err.Error())
		} else {
			log.Info("Start monitoring")
			for event := range events {
				if event.IsCreate() {
					log.Debugf("Created: %s", event.Path())
					job := job.NewMacros(
						job.NewCheckExt(event.Path(), ".torrent"),
						job.NewWait(time.Second),
						job.NewMove(event.Path(), filepath.Join(cfg.TorrentsDir, filepath.Base(event.Path()))))
					workerInst.Push(job, worker.Priority_Low)
				} else if event.IsDelete() {
					log.Debugf("Deleted: %s", event.Path())
				} else if event.IsRename() {
					log.Debugf("Renamed: %s", event.Path())
				}
			}
		}

	}()

	wg.Wait()
}

func main() {
	var filepath *string
	if len(os.Args) > 1 {
		for _, arg := range os.Args {
			if strings.HasPrefix(arg, "logfile=") {
				path := arg[8:]
				filepath = &path
			}
		}
	}
	log.Setup(filepath)

	cfg := config.NewConfig()
	if err := cfg.Read([]string{"/etc/torrent-monitor", "../etc", "./etc"}); err != nil {
		log.Critical(err.Error())
		return
	}

	if cfg.DaemonMode {
		srv, err := daemon.New(name, description, dependencies...)
		if err != nil {
			log.Critical(err.Error())
			os.Exit(1)
		}
		service := &Service{srv}
		status, err := service.Manage(cfg)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		if len(status) > 0 {
			log.Info(status)
		}
	} else {
		run(cfg)
	}
}
