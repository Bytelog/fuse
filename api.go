package fuse

import (
	"golang.org/x/sys/unix"
)

type Requester interface {
	Headers() *Header
	Interrupt() <-chan struct{}
	String() string
}

type Responder interface {
	Reply(unix.Errno) error
	String() string
}

type Handler interface {
	Init(*InitRequest, *InitResponse)
	Destroy(*DestroyRequest, *DestroyResponse)
	Access(*AccessRequest, *AccessResponse)
	Lookup(*LookupRequest, *LookupResponse)
	Opendir(*OpendirRequest, *OpendirResponse)

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

var _ Handler = HandlerFunc(nil)

type HandlerFunc func(Requester, Responder)

func (f HandlerFunc) Init(req *InitRequest, resp *InitResponse)          { f(req, resp) }
func (f HandlerFunc) Access(req *AccessRequest, resp *AccessResponse)    { f(req, resp) }
func (f HandlerFunc) Lookup(req *LookupRequest, resp *LookupResponse)    { f(req, resp) }
func (f HandlerFunc) Destroy(req *DestroyRequest, resp *DestroyResponse) { f(req, resp) }
func (f HandlerFunc) Opendir(req *OpendirRequest, resp *OpendirResponse) { f(req, resp) }

var DefaultFilesystem = HandlerFunc(func(_ Requester, resp Responder) {
	if err := resp.Reply(unix.ENOSYS); err != nil {
		panic(err)
	}
})



