package fsevents

type IFSEvent interface {
	// IsCreate reports whether the FileEvent was triggered by a creation
	IsCreate() bool
	// IsDelete reports whether the FileEvent was triggered by a delete
	IsDelete() bool
	// IsModify reports whether the FileEvent was triggered by a file modification
	IsModify() bool
	// IsRename reports whether the FileEvent was triggered by a change name
	IsRename() bool
	// IsAttrib reports whether the FileEvent was triggered by a change in the file metadata.
	IsAttrib() bool

	Path() string
}

type IFSEventsWatcher interface {
	Watch(path string, done <-chan bool) (<-chan IFSEvent, error)
}
