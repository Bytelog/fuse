package fuse

import (
	"errors"
	"fmt"
)

var (
	ErrReplyUnsupported = errors.New("fuse: operation does not support reply")
)

type Request interface {
	Header() *Header
	Interrupt() <-chan struct{}
	String() string
}

type Response interface {
	ReplyErr(err error) error
}

type Handler interface {
	Init(*InitRequest)
	Destroy(*DestroyRequest)

	/*
		Destroy(Request)
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

type HandlerFunc func(Request, Response)

type noResponse struct {
	req Request
}

func (e noResponse) ReplyErr(_ error) error {
	return fmt.Errorf("%w: %s", ErrReplyUnsupported, e.req.String())
}

func (f HandlerFunc) Init(req *InitRequest)       { f(req, req) }
func (f HandlerFunc) Destroy(req *DestroyRequest) { f(req, req) }
func (f HandlerFunc) Forget(req *ForgetRequest)   { f(req, noResponse{req}) }
