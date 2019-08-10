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
	Unlink(*Context, *UnlinkIn) error
	Rmdir(*Context, *RmdirIn) error
	Rename(*Context, *RenameIn) error
	Link(*Context, *LinkIn, *LinkOut) error
	Open(*Context, *OpenIn, *OpenOut) error
	Read(*Context, *ReadIn, *ReadOut) error
	Lseek(*Context, *LseekIn, *LseekOut) error
	CopyFileRange(*Context, *CopyFileRangeIn) error
	Release(*Context, *ReleaseIn) error

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
	_ = f(ctx, in, nil)
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

func (f HandlerFunc) Unlink(ctx *Context, in *UnlinkIn) error {
	return f(ctx, in, nil)
}

func (f HandlerFunc) Rmdir(ctx *Context, in *RmdirIn) error {
	return f(ctx, in, nil)
}

func (f HandlerFunc) Rename(ctx *Context, in *RenameIn) error {
	return f(ctx, in, nil)
}

func (f HandlerFunc) Link(ctx *Context, in *LinkIn, out *LinkOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Open(ctx *Context, in *OpenIn, out *OpenOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Read(ctx *Context, in *ReadIn, out *ReadOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) Lseek(ctx *Context, in *LseekIn, out *LseekOut) error {
	return f(ctx, in, out)
}

func (f HandlerFunc) CopyFileRange(ctx *Context, in *CopyFileRangeIn) error {
	return f(ctx, in, nil)
}

func (f HandlerFunc) Release(ctx *Context, in *ReleaseIn) error {
	return f(ctx, in, nil)
}

var DefaultFilesystem = HandlerFunc(func(ctx *Context, req Request, resp Response) error {
	switch ctx.Op {
	case proto.INIT:
		return nil
	default:
		return ENOSYS
	}
})
