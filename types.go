package fuse

import (
	"syscall"
	"unsafe"

	"bytelog.org/fuse/proto"
)

type Context struct {
	req  RawRequest
	resp RawResponse

	sess *session
}

func (ctx *Context) Interrupt() <-chan struct{} {
	return nil
}

func (ctx *Context) String() string {
	return "OP_" + ctx.req.Header.OpCode.String()
}

// pointer to the request buffer, reserved up to size bytes.
func (ctx *Context) in(size uintptr) unsafe.Pointer {
	if len(ctx.req.Data) < int(size) {
		panic("requested size overflows request buffer")
	}
	return unsafe.Pointer(&ctx.req.Data[0])
}

// pointer to the request buffer's data segment, reserved up to size bytes
func (ctx *Context) inData(size uintptr) unsafe.Pointer {
	if len(ctx.req.Data)-int(headerInSize) < int(size) {
		panic("requested size overflows request buffer")
	}
	return unsafe.Pointer(&ctx.req.Data[headerInSize])
}

// pointer to the response buffer, clearing bytes if requested.
func (ctx *Context) out(clear uintptr) unsafe.Pointer {
	if clear > 0 {
		data := ctx.resp.Data[:clear]
		for i := range data {
			data[i] = 0
		}
	}
	return unsafe.Pointer(&ctx.resp.Data[0])
}

// pointer to the response buffer's data segment, clearing bytes if requested.
func (ctx *Context) outData(clear uintptr) unsafe.Pointer {
	if clear > 0 {
		data := ctx.resp.Data[headerOutSize : headerOutSize+clear]
		for i := range data {
			data[i] = 0
		}
	}
	return unsafe.Pointer(&ctx.resp.Data[headerOutSize])
}

func (ctx *Context) bytes(off int) []byte {
	return ctx.req.Data[int(headerInSize)+off:]
}

var (
	ENOENT = syscall.ENOENT
	ENOSYS = syscall.ENOSYS
	EPROTO = syscall.EPROTO
)

type Header struct {
	len    uint32
	code   proto.OpCode
	ID     uint64
	NodeID uint64
	UID    uint32
	GID    uint32
	PID    uint32
	_      uint32
}

type outHeader struct {
	proto.OutHeader
}

type InitIn struct {
	Header
	Major        uint32
	Minor        uint32
	MaxReadahead uint32
	Flags        uint32
}

type InitOut struct {
	outHeader
	major               uint32
	minor               uint32
	MaxReadahead        uint32
	Flags               uint32
	MaxBackground       uint16
	CongestionThreshold uint16
	MaxWrite            uint32
	TimeGran            uint32
	MaxPages            uint16
	_                   uint16
	_                   [8]uint32
}

type AccessIn struct {
	Header
	Mask uint32
	_    uint32
}

type AccessOut struct {
	outHeader
}

type GetattrIn struct {
	Header
	// normally used to tell if Fh is set. We don't expose the flag bits since
	// we don't consider 0 a valid file handle.
	flags uint32
	_     uint32
	Fh    uint64
}

type GetattrOut struct {
	outHeader
	AttrValid     uint64
	AttrValidNsec uint32
	_             uint32
	Attr
}

type Attr struct {
	Ino       uint64
	Size      uint64
	Blocks    uint64
	Atime     uint64
	Mtime     uint64
	Ctime     uint64
	Atimensec uint32
	Mtimensec uint32
	Ctimensec uint32
	Mode      uint32
	Nlink     uint32
	Uid       uint32
	Gid       uint32
	Rdev      uint32
	Blksize   uint32
	_         uint32
}

type DestroyIn struct{ Header }
type DestroyOut struct{ outHeader }

type LookupIn struct {
	Header
}

type LookupOut struct {
	outHeader
	EntryOut
}

type EntryOut struct {
	// Inode ID
	Nodeid uint64

	// Inode generation: Nodeid:gen must be unique for the fs's lifetime
	Generation uint64

	// Cache timeout for the name
	EntryValid uint64

	// Cache timeout for the attributes
	AttrValid uint64

	EntryValidNsec uint32
	AttrValidNsec  uint32
	Attr           Attr
}
