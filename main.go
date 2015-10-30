package main

import (
	"github.com/darvik80/monitor/fsevents"
	"github.com/darvik80/monitor/log"
	"time"
)

func main() {
	log.Info("Start watch")

	watcher, err := fsevents.NewWatcher()
	if err != nil {
		return
	}
	done := make(chan bool)
	if events, err := watcher.Watch("/tmp", done); err != nil {
		return
	} else {
		var opened = true
		for opened {
			select {
			case event, more := <-events:
				if more {
					if event.IsCreate() {
						log.Debugf("Created: %s", event.Path())
					} else if event.IsDelete() {
						log.Debugf("Deleted: %s", event.Path())
					} else if event.IsRename() {
						log.Debugf("Renamed: %s", event.Path())
					}
					//else if (event.IsModify()) {
					//	log.Printf("Modified: %s", event.Path())
					//}
				} else {
					opened = false
				}
			case <-time.After(time.Minute * 30):
				done <- true
			}
		}
	}

	log.Info("Stop watch")
}
