package fsevents

import (
	"github.com/dropbox/godropbox/errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/darvik80/fsevents/log"
	"io/ioutil"
)

// +build darwin

const (
	Note_DELETE = 0x00000001 /* vnode was removed */
	Note_WRITE  = 0x00000002 /* data contents changed */
	Note_EXTEND = 0x00000004 /* size increased */
	Note_ATTRIB = 0x00000008 /* attributes changed */
	Note_LINK   = 0x00000010 /* link count changed */
	Note_RENAME = 0x00000020 /* vnode was renamed */
	Note_REVOKE = 0x00000040 /* vnode access was revoked */
	Note_NONE   = 0x00000080 /* No specific vnode event: to test for EVFILT_READ activation*/

	// Watch all events
	Note_ALLEVENTS = Note_DELETE | Note_WRITE | Note_ATTRIB | Note_RENAME

	open_FLAGS = syscall.O_EVTONLY

	// Block for 100 ms on each call to kevent
	keventWaitTime = 100e6
	sysCallFailed  = -1
)

type FileEvent struct {
	mask   uint32 // Mask of events
	Name   string // File name (optional)
	create bool   // set by fsnotify package if found new file
}

// IsCreate reports whether the FileEvent was triggered by a creation
func (e *FileEvent) IsCreate() bool { return e.create }

// IsDelete reports whether the FileEvent was triggered by a delete
func (e *FileEvent) IsDelete() bool { return (e.mask & Note_DELETE) == Note_DELETE }

// IsModify reports whether the FileEvent was triggered by a file modification
func (e *FileEvent) IsModify() bool {
	return ((e.mask&Note_WRITE) == Note_WRITE || (e.mask&Note_ATTRIB) == Note_ATTRIB)
}

// IsRename reports whether the FileEvent was triggered by a change name
func (e *FileEvent) IsRename() bool { return (e.mask & Note_RENAME) == Note_RENAME }

// IsAttrib reports whether the FileEvent was triggered by a change in the file metadata.
func (e *FileEvent) IsAttrib() bool {
	return (e.mask & Note_ATTRIB) == Note_ATTRIB
}

func (e *FileEvent) Path() string {
	return e.Name
}

func (e *FileEvent) Flags() uint32 {
	return e.mask
}

type watchInfo struct {
	fd       int
	flags    uint32
	realPath string
	fInto    os.FileInfo
	childes  []string
	external bool
}

type watcher struct {
	kq            int
	watchedByFD   map[int]*watchInfo
	watchedByPath map[string]*watchInfo
	done          chan bool
}

func NewWatcher() (IFSEventsWatcher, error) {
	if fd, err := syscall.Kqueue(); fd == sysCallFailed {
		return nil, os.NewSyscallError("kqueue", err)
	} else {
		w := &watcher{
			kq:            fd,
			watchedByFD:   make(map[int]*watchInfo),
			watchedByPath: make(map[string]*watchInfo),
		}

		return w, nil
	}
}

func (this *watcher) Watch(path string, flags uint32, done <-chan bool) (<-chan IFSEvent, error) {
	if err := this.doAdd(path, Note_ALLEVENTS); err != nil {
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

		errno := syscall.Close(this.kq)
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

	mode := fileInfo.Mode()

	// don't watch socket
	if mode&os.ModeSocket == os.ModeSocket {
		return nil
	}

	if mode&os.ModeSymlink == os.ModeSymlink {
		if path, err = filepath.EvalSymlinks(path); err != nil {
			return err
		}
		if fileInfo, err = os.Lstat(path); err != nil {
			return err
		}
	}

	fd, err := syscall.Open(path, open_FLAGS, 0700)
	if fd == sysCallFailed {
		return err
	}

	info := &watchInfo{
		fd:       fd,
		flags:    flags,
		realPath: path,
		fInto:    fileInfo,
	}
	this.watchedByFD[fd] = info
	this.watchedByPath[path] = info

	var kbuf [1]syscall.Kevent_t
	watchEntry := &kbuf[0]
	watchEntry.Fflags = flags
	syscall.SetKevent(watchEntry, info.fd, syscall.EVFILT_VNODE, syscall.EV_ADD|syscall.EV_CLEAR)
	entryFlags := watchEntry.Flags
	if success, err := syscall.Kevent(this.kq, kbuf[:], nil, nil); success == -1 {
		return err
	} else if (entryFlags & syscall.EV_ERROR) == syscall.EV_ERROR {
		return errors.New("kevent add error")
	}

	if (true == info.fInto.IsDir()) && (info.flags&Note_WRITE == Note_WRITE) {
		if infos, err := ioutil.ReadDir(path); err != nil {
			return err
		} else {
			for _, fileInfo := range infos {
				childPath := filepath.Join(path, fileInfo.Name())
				if err := this.doAdd(childPath, flags); err != nil {
					return err
				}
				info.childes = append(info.childes, childPath)
			}
		}
	}

	return nil
}

func (this *watcher) doDel(path string) error {
	info, found := this.watchedByPath[path]
	if !found {
		return errors.Newf("can't remove non-existent kevent watch for: %s", path)
	}

	var kbuf [1]syscall.Kevent_t
	watchEntry := &kbuf[0]
	syscall.SetKevent(watchEntry, info.fd, syscall.EVFILT_VNODE, syscall.EV_DELETE)
	entryFlags := watchEntry.Flags
	success, errno := syscall.Kevent(this.kq, kbuf[:], nil, nil)
	if success == sysCallFailed {
		return os.NewSyscallError("kevent_rm_watch", errno)
	} else if entryFlags&syscall.EV_ERROR == syscall.EV_ERROR {
		return errors.New("kevent rm error")
	}
	syscall.Close(info.fd)

	//Remove childs if it's directory
	for _, child := range info.childes {
		this.doDel(child)
	}

	delete(this.watchedByPath, path)
	delete(this.watchedByFD, info.fd)

	return nil
}

// readEvents reads from the kqueue file descriptor, converts the
// received events into Event objects and sends them via the Event channel
func (this *watcher) doReadEvents(flags uint32, events chan<- IFSEvent, devNull chan<- error, done <-chan bool) {
	var (
		keventbuf [10]syscall.Kevent_t // Event buffer
		kevents   []syscall.Kevent_t   // Received events
		twait     *syscall.Timespec    // Time to block waiting for events
	)
	kevents = keventbuf[0:0]
	twait = new(syscall.Timespec)
	*twait = syscall.NsecToTimespec(keventWaitTime)

	for {
		// If "done" message is received

		select {
		case <-done:
			return
		default:

		}

		// Get new events
		if len(kevents) == 0 {
			n, errno := syscall.Kevent(this.kq, nil, keventbuf[:], twait)

			// EINTR is okay, basically the syscall was interrupted before
			// timeout expired.
			if errno != nil && errno != syscall.EINTR {
				devNull <- os.NewSyscallError("kevent", errno)
				continue
			}

			// Received some events
			if n > 0 {
				kevents = keventbuf[0:n]
			}
		}

		// Flush the events we received to the events channel
		for len(kevents) > 0 {
			fileEvent := new(FileEvent)
			watchEvent := &kevents[0]
			// Move to next event
			kevents = kevents[1:]

			fileEvent.mask = uint32(watchEvent.Fflags)
			info := this.watchedByFD[int(watchEvent.Ident)]
			if info != nil {
				fileEvent.Name = info.realPath
			}
			if info != nil && info.fInto.IsDir() && !fileEvent.IsDelete() {
				// Double check to make sure the directory exist. This can happen when
				// we do a rm -fr on a recursively watched folders and we receive a
				// modification event first but the folder has been deleted and later
				// receive the delete event
				if _, err := os.Lstat(fileEvent.Name); os.IsNotExist(err) {
					// mark is as delete event
					fileEvent.mask |= Note_DELETE
				}
			}

			if info != nil && info.fInto.IsDir() && fileEvent.IsModify() && !fileEvent.IsDelete() {
				this.sendDirectoryChangeEvents(info.realPath, info.flags, devNull, events)
			} else {
				// Send the event on the events channel
				if flags&fileEvent.mask != 0 || fileEvent.create {
					events <- fileEvent
				}
			}

			if fileEvent.IsDelete() {
				if err := this.doDel(fileEvent.Name); err != nil {
					devNull <- err
				}

				if _, err := os.Lstat(info.realPath); !os.IsNotExist(err) {
					this.sendDirectoryChangeEvents(info.realPath, flags, devNull, events)
				}
			} else if fileEvent.IsRename() {
				if err := this.doDel(fileEvent.Name); err != nil {
					devNull <- err
				}
			}

		}
	}
}

// sendDirectoryEvents searches the directory for newly created files
// and sends them over the event channel. This functionality is to have
// the BSD version of fsnotify match linux fsnotify which provides a
// create event for files created in a watched directory.
func (this *watcher) sendDirectoryChangeEvents(dirPath string, flags uint32, devNull chan<- error, events chan<- IFSEvent) {
	// Get all files
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		devNull <- err
	} else {
		// Search for new files
		for _, fileInfo := range files {
			filePath := filepath.Join(dirPath, fileInfo.Name())
			if _, found := this.watchedByPath[filePath]; !found {
				if err := this.doAdd(filePath, Note_ALLEVENTS); err == nil {
					// Send create event
					fileEvent := new(FileEvent)
					fileEvent.Name = filePath
					fileEvent.create = true
					if flags&fileEvent.mask != 0 || fileEvent.create {
						events <- fileEvent
					}
				} else {
					devNull <- err
				}
			}
		}
	}
}
