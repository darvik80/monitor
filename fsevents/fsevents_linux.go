package fsevents

import (
	"errors"
	"github.com/darvik80/monitor/log"
	"os"
	"strings"
	"syscall"
	"unsafe"
	"sync"
)

// +build linux

const (
	// Events
	IN_ACCESS        uint32 = syscall.IN_ACCESS
	IN_ALL_EVENTS    uint32 = syscall.IN_ALL_EVENTS
	IN_ATTRIB        uint32 = syscall.IN_ATTRIB
	IN_CLOSE         uint32 = syscall.IN_CLOSE
	IN_CLOSE_NOWRITE uint32 = syscall.IN_CLOSE_NOWRITE
	IN_CLOSE_WRITE   uint32 = syscall.IN_CLOSE_WRITE
	IN_CREATE        uint32 = syscall.IN_CREATE
	IN_DELETE        uint32 = syscall.IN_DELETE
	IN_DELETE_SELF   uint32 = syscall.IN_DELETE_SELF
	IN_MODIFY        uint32 = syscall.IN_MODIFY
	IN_MOVE          uint32 = syscall.IN_MOVE
	IN_MOVED_FROM    uint32 = syscall.IN_MOVED_FROM
	IN_MOVED_TO      uint32 = syscall.IN_MOVED_TO
	IN_MOVE_SELF     uint32 = syscall.IN_MOVE_SELF
	IN_OPEN          uint32 = syscall.IN_OPEN
)

type fileEvent struct {
	mask uint32 // Mask of events
	Name string // File name (optional)
}

// IsCreate reports whether the FileEvent was triggered by a creation
func (e *fileEvent) IsCreate() bool {
	return (e.mask & IN_CREATE) == IN_CREATE
}

// IsDelete reports whether the FileEvent was triggered by a delete
func (e *fileEvent) IsDelete() bool {
	return (e.mask&IN_DELETE) == IN_DELETE || (e.mask&IN_DELETE_SELF) == IN_DELETE_SELF
}

// IsModify reports whether the FileEvent was triggered by a file modification
func (e *fileEvent) IsModify() bool {
	return (e.mask & IN_MODIFY) == IN_MODIFY
}

// IsRename reports whether the FileEvent was triggered by a change name
func (e *fileEvent) IsRename() bool {
	return (e.mask&IN_MOVED_FROM) == IN_MOVED_FROM || (e.mask&IN_MOVED_TO) == IN_MOVED_TO || (e.mask&IN_MOVE) == IN_MOVE || (e.mask&IN_MOVE_SELF) == IN_MOVE_SELF
}

// IsAttrib reports whether the FileEvent was triggered by a change in the file metadata.
func (e *fileEvent) IsAttrib() bool {
	return (e.mask & IN_ATTRIB) == IN_ATTRIB
}

func (e *fileEvent) Path() string {
	return e.Name
}

func (e *fileEvent) Flags() uint32 {
	return e.mask
}

type watchInfo struct {
	fd    int
	flags uint32
	path  string
}

type watcher struct {
	fd   int
	root string
	wd   int
	done chan bool
}

func NewWatcher() (IFSEventsWatcher, error) {
	if fd, errno := syscall.InotifyInit(); errno != nil {
		return nil, os.NewSyscallError("inotify_init", errno)
	} else {
		w := &watcher{
			fd: fd,
		}

		return w, nil
	}
}

func (this *watcher) Watch(path string, done <-chan bool) (<-chan IFSEvent, error) {
	if err := this.doAdd(path, IN_ALL_EVENTS); err != nil {
		return nil, err
	}

	events := make(chan IFSEvent)
	go func() {
		defer close(events)
		devNull := make(chan error)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for err := range devNull {
				log.Printf("/dev/null/errors: %s\n", err.Error())
			}
		}()

		wg.Add(1)
		go func() {
			defer func() {
				close(devNull)
				wg.Done()
			}()

			this.doReadEvents(IN_ALL_EVENTS, events, devNull, done)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			<- done
			this.doDel()

			errno := syscall.Close(this.fd)
			if errno != nil {
				devNull <- os.NewSyscallError("close", errno)
			}
		}()
		wg.Wait()
	}()

	return events, nil
}

func (this *watcher) doAdd(path string, flags uint32) error {
	//Check that file exists
	if _, err := os.Lstat(path); err != nil {
		return err
	}

	if wd, errno := syscall.InotifyAddWatch(this.fd, path, flags); wd < 0 {
		return os.NewSyscallError("inotify_add_watch", errno)
	} else {
		this.wd = wd
		this.root = path
	}

	return nil
}

func (this *watcher) doDel() error {
	if res, errno := syscall.InotifyRmWatch(this.fd, uint32(this.wd)); res < 0 {
		return os.NewSyscallError("inotify_add_watch", errno)
	}

	return nil
}

// readEvents reads from the kqueue file descriptor, converts the
// received events into Event objects and sends them via the Event channel
func (this *watcher) doReadEvents(flags uint32, events chan<- IFSEvent, devNull chan<- error, done <-chan bool) {
	var buf [syscall.SizeofInotifyEvent * 4096]byte

	for {
		// If "done" message is received
		select {
		case <-done:
			return
		default:

		}

		if len, errno := syscall.Read(this.fd, buf[:]); len < 0 {
			devNull <- os.NewSyscallError("read", errno)
			continue
		} else if len < syscall.SizeofInotifyEvent {
			devNull <- errors.New("inotify: short read in readEvents()")
			continue
		} else {

			var offset uint32 = 0
			// We don't know how many events we just read into the buffer
			// While the offset points to at least one whole event...
			for offset <= uint32(len-syscall.SizeofInotifyEvent) {
				// Point "raw" to the event in the buffer
				raw := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))

				nameLen := uint32(raw.Len)
				if this.wd == int(raw.Wd) {
					event := &fileEvent{
						Name: this.root,
						mask: raw.Mask,
					}

					if nameLen > 0 {
						// Point "bytes" at the first byte of the filename
						bytes := (*[syscall.PathMax]byte)(unsafe.Pointer(&buf[offset+syscall.SizeofInotifyEvent]))
						// The filename is padded with NUL bytes. TrimRight() gets rid of those.
						event.Name += "/" + strings.TrimRight(string(bytes[0:nameLen]), "\000")
					}

					events <- event
				}

				offset += syscall.SizeofInotifyEvent + nameLen
			}
		}
	}
}
