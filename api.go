package fuse

type RequestHandler func()

type ResponseHandler func()

type Timespec struct{}

type Request struct {
	// The Inode ID tracked by the fuse client.
	NodeID uint64

	// UID of the requesting process.
	UID uint32

	// GID of the requesting process.
	GID uint32

	// PID of the requesting process.
	PID uint32
}

type EntryOut struct{}

type AttrOut struct{}

type SetAttrIn struct{}

type Link struct {
	NodeID uint64
	Name   string
}

// entry should have generation, nodeid, attr timeout, entry timeout
// attr should have attr timeout

// todo: mknod, mkdir and create have a umask argument. This can be abstracted
// away by applying the umask immediately, but is there a reason to make the raw
// mode or the raw umask available? umasks aren't even available in bsd-likes,
// and certainly not windows.

// todo: type conversions - use golang int sizes where idiomatic and convertable

type Filesystem interface {
	Init(Request)
	Destroy(Request)

	/*
		Lookup(r Request, name string) (EntryOut, error)
		Forget(r Request, lookups uint64)

		GetAttr(r Request, flags uint32, fh uint64) (AttrOut, error)
		SetAttr(r Request, in SetAttrIn) (AttrOut, error)

		Readlink(r Request) ([]byte, error)
		Symlink(r Request, name, target string) (EntryOut, error)

		Mknod(r Request, name string, mode os.FileMode, dev uint32) (EntryOut, error)
		Mkdir(r Request, name string, mode os.FileMode) (EntryOut, error)

		Unlink(r Request, name string) error
		Rmdir(r Request, name string) error

		// todo: make it clear that parent is the target's new inode.
		Rename(r Request, name string, parent uint64, target string) (EntryOut, error)

		Link(r Request, parent uint64, target string) (EntryOut, error)
		// Open(r Request, flags uint32) (fh uint64, flags uint32, err error)

		Read(r Request, fh uint64, offset int64, size uint32) ([]byte, error)
		Write(r Request, fh uint64, offset int64, data []byte, flags uint32) (uint32, error)
	*/
}

type DefaultFilesystem struct{}

func (fs *DefaultFilesystem) Init(Request) {}

func (fs *DefaultFilesystem) Destroy(Request) {}
