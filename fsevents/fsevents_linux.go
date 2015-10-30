package fsevents

import (
	"syscall"
	"mime"
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
	mask   uint32 // Mask of events
	Name   string // File name (optional)
	create bool   // set by fsnotify package if found new file
}

// IsCreate reports whether the FileEvent was triggered by a creation
func (e *fileEvent) IsCreate() bool {
	return (e.mask & IN_CREATE) == IN_CREATE
}

// IsDelete reports whether the FileEvent was triggered by a delete
func (e *fileEvent) IsDelete() bool {
	return (e.mask & IN_DELETE) == IN_DELETE || (e.mask & IN_DELETE_SELF) == IN_DELETE_SELF
}

// IsModify reports whether the FileEvent was triggered by a file modification
func (e *FileEvent) IsModify() bool {
	return (e.mask & IN_MODIFY) == IN_MODIFY || (e.mask & IN_MODIFY_SELF) ==  IN_MODIFY_SELF
}

// IsRename reports whether the FileEvent was triggered by a change name
func (e *fileEvent) IsRename() bool {
	return (e.mask & IN_MOVED_FROM) == IN_MOVED_FROM || (e.mask & IN_MOVED_TO) == IN_MOVED_TO
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
	fd       int
	flags    uint32
	path string
}

type watcher struct {
	fd            int
	root string
	wd int
	done          chan bool
}

func NewWatcher() (IFSEventsWatcher, error) {
	if fd, err := syscall.InotifyInit() {
		return nil, os.NewSyscallError("inotify_init", err)
	} else {
		w := &watcher{
			fd:            fd,
			watchedByFD:   make(map[int]*watchInfo),
		}

		return w, nil
	}
}

func (this *watcher) Watch(path string, flags uint32, done <-chan bool) (<-chan IFSEvent, error) {
	if err := this.doAdd(path, syscall.IN_ALL_EVENTS); err != nil {
		return nil, err
	}

	events := make(chan IFSEvent)
	go func() {
		defer close(events)
		devNull := make(chan error)
		go func() {
			for err := range devNull {
				log.Printf("/dev/null/errors: %s\n", err.Error())
			}
		}()

		this.doReadEvents(flags, events, devNull, done)

		this.doDel(path)

		errno := syscall.Close(this.fd)
		if errno != nil {
			devNull <- os.NewSyscallError("close", errno)
		}
	}()

	return events, nil
}

func (this *watcher) doAdd(path string, flags uint32) error {
	//Check that file exists
	fileInfo, err := os.Lstat(path)
	if err != nil {
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

func (this *watcher) doDel(path string) error {
	if errno := syscall.InotifyRmWatch(this.fd, this.wd); errno < 0 {
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

		if len, errno := syscall.Read(w.fd, buf[:]); len < 0 {
			devNull <- os.NewSyscallError("read", errno)
			continue
		} else if len < syscall.SizeofInotifyEvent {
			devNull <- errors.New("inotify: short read in readEvents()")
			continue
		}

		var offset uint32 = 0
		// We don't know how many events we just read into the buffer
		// While the offset points to at least one whole event...
		for offset <= uint32(n-syscall.SizeofInotifyEvent) {
			// Point "raw" to the event in the buffer
			raw := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))

			if this.wd == int(raw.Wd) {
				nameLen := uint32(raw.Len)
				event := fileEvent{
					Name: this.root,
					mask: raw.Mask,
				}

				if nameLen > 0 {
					// Point "bytes" at the first byte of the filename
					bytes := (*[syscall.PathMax]byte)(unsafe.Pointer(&buf[offset + syscall.SizeofInotifyEvent]))
					// The filename is padded with NUL bytes. TrimRight() gets rid of those.
					event.Name += "/" + strings.TrimRight(string(bytes[0:nameLen]), "\000")
				}

				events <- event
			}

			offset += syscall.SizeofInotifyEvent + nameLen
		}
	}
}
