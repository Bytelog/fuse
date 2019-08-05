package fuse

import "bytelog.org/fuse/proto"

type Request interface{}

type Response interface{}

type Filesystem interface {
	Init(*Context, *InitIn, *InitOut) error
	Access(*Context, *AccessIn) error
	Getattr(*Context, *GetattrIn, *GetattrOut) error
	Destroy(*Context) error
	Lookup(*Context, *LookupIn, *LookupOut) error
	Forget(*Context, *ForgetIn)
	Setattr(*Context, *SetattrIn, *SetattrOut) error
	Readlink(*Context, *ReadlinkOut) error
	Symlink(*Context, *SymlinkIn, *SymlinkOut) error
	Mknod(*Context, *MknodIn, *MknodOut) error
	Mkdir(*Context, *MkdirIn, *MkdirOut) error
	// todo: what about *EntryOut? Less types?
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

var _ Filesystem = HandlerFunc(nil)

type HandlerFunc func(*Context, Request, Response) error

func (f HandlerFunc) Lookup(ctx *Context, in *LookupIn, out *LookupOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Forget(ctx *Context, in *ForgetIn) {
	f(ctx, in, nil)
}

func (f HandlerFunc) Init(ctx *Context, in *InitIn, out *InitOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Access(ctx *Context, in *AccessIn) error {
	return f(ctx, in, nil)
}

func (f HandlerFunc) Getattr(ctx *Context, in *GetattrIn, out *GetattrOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Destroy(ctx *Context) error {
	return f(ctx, nil, nil)
}

func (f HandlerFunc) Setattr(ctx *Context, in *SetattrIn, out *SetattrOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Readlink(ctx *Context, out *ReadlinkOut) error {
	return f(ctx, nil, out)
}

func (f HandlerFunc) Symlink(ctx *Context, in *SymlinkIn, out *SymlinkOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Mknod(ctx *Context, in *MknodIn, out *MknodOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Mkdir(ctx *Context, in *MkdirIn, out *MkdirOut) error {
	return f(ctx, in, out)
}

var DefaultFilesystem = HandlerFunc(func(ctx *Context, req Request, resp Response) error {
	switch ctx.Op {
	case proto.INIT:
		return nil
	default:
		return ENOSYS
	}
})
