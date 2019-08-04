package fuse

type Request interface{}

type Response interface{}

type Handler interface {
	Init(*Context, *InitIn, *InitOut) error
	Access(*Context, *AccessIn, *AccessOut) error
	Getattr(*Context, *GetattrIn, *GetattrOut) error
	Destroy(*Context, *DestroyIn, *DestroyOut) error
	Lookup(*Context, *LookupIn, *LookupOut) error
	/*Destroy(*Context, *DestroyIn, *DestroyOut) error
	Access(*Context, *AccessIn, *AccessOut) error
	Lookup(*Context, *LookupIn, *LookupOut) error
	Opendir(*Context, *OpendirIn, *OpendirOut) error
	Readdir(*Context, *ReaddirIn, *ReaddirOut) error*/

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

type HandlerFunc func(*Context, Request, Response) error

func (f HandlerFunc) Lookup(ctx *Context, req *LookupIn, resp *LookupOut) error {
	return f(ctx, req, resp)
}

func (f HandlerFunc) Init(ctx *Context, req *InitIn, resp *InitOut) error {
	return f(ctx, req, resp)
}

func (f HandlerFunc) Access(ctx *Context, req *AccessIn, resp *AccessOut) error {
	return f(ctx, req, resp)
}

func (f HandlerFunc) Getattr(ctx *Context, req *GetattrIn, resp *GetattrOut) error {
	return f(ctx, req, resp)
}

func (f HandlerFunc) Destroy(ctx *Context, req *DestroyIn, resp *DestroyOut) error {
	return f(ctx, req, resp)
}

var DefaultFilesystem = HandlerFunc(func(ctx *Context, req Request, resp Response) error {
	switch req.(type) {
	case *InitIn:
		return nil
	default:
		return ENOSYS
	}
})
