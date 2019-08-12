package fuse

import (
	"fmt"
	"syscall"
	"unsafe"

	"bytelog.org/fuse/proto"
)

var (
	ENOENT = syscall.ENOENT
	ENOSYS = syscall.ENOSYS
	EPROTO = syscall.EPROTO
)

type Context struct {
	noCopy noCopy

	buf  []byte
	off  int
	sess *session

	// Request buffer begins here. NO ADDITIONAL FIELDS BELOW THIS LINE.
	Header
}

func (ctx *Context) Interrupt() <-chan struct{} {
	return nil
}

func (ctx *Context) String() string {
	return "OP_" + ctx.Op.String()
}

// pointer to the request data
func (ctx *Context) in() unsafe.Pointer {
	return unsafe.Pointer(&ctx.buf[headerInSize])
}

func (ctx *Context) bytes(off uintptr) []byte {
	return ctx.buf[headerInSize+off : ctx.off]
}

func (ctx *Context) string() string {
	buf := ctx.bytes(0)
	return string(buf[:strlen(buf)])
}

func (ctx *Context) strings(n int) []string {
	buf := ctx.bytes(0)
	s := make([]string, n)

	for i := range s {
		n := strlen(buf)
		s[i] = string(buf[:n])
		if len(buf) == n {
			return s
		}
		buf = buf[n+1:]
	}
	return s
}

// pointer to the response data
func (ctx *Context) out() unsafe.Pointer {
	return unsafe.Pointer(&ctx.buf[ctx.off+int(headerOutSize)])
}

// pointer to the response buffer, with size bytes zero initialized
func (ctx *Context) outzero(size uintptr) unsafe.Pointer {
	start := ctx.off + int(headerOutSize)
	if size > 0 {
		buf := ctx.buf[start : start+int(size)]
		for i := range buf {
			buf[i] = 0
		}
	}
	return unsafe.Pointer(&ctx.buf[start])
}

func (ctx *Context) outBuf() []byte {
	return ctx.buf[ctx.off:]
}

func (ctx *Context) outData() []byte {
	return ctx.buf[ctx.off+int(headerOutSize):]
}

func (ctx *Context) outHeader() *proto.OutHeader {
	return (*proto.OutHeader)(unsafe.Pointer(&ctx.buf[ctx.off]))
}

// bump the input buffer size and memclr the affected bytes
func (ctx *Context) shift(n int) {
	buf := ctx.buf[ctx.off : ctx.off+n]
	for i := range buf {
		buf[i] = 0
	}
	ctx.off += n
}

func (ctx *Context) writeString(s string) (n int) {
	// todo: bounds check
	buf := ctx.outData()
	n = copy(buf, s)
	buf[n] = 0
	return n + 1
}

type Header struct {
	len    uint32
	Op     proto.OpCode
	ID     uint64
	NodeID uint64
	UID    uint32
	GID    uint32
	PID    uint32
	_      uint32
}

func (h Header) Debug() string {
	return fmt.Sprintf("{ID:%d NodeID:%d UID:%d GID:%d PID:%d}",
		h.ID, h.NodeID, h.UID, h.GID, h.PID)
}

type InitIn struct {
	Major        uint32
	Minor        uint32
	MaxReadahead uint32
	Flags        uint32
}

type InitOut struct {
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
	Mask uint32
	_    uint32
}

type GetattrIn struct {
	// normally used to tell if Fh is set. We don't expose the flag bits since
	// we don't consider 0 a valid file handle.
	flags uint32
	_     uint32
	Fh    uint64
}

type GetattrOut struct {
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

type LookupIn struct {
	Name string
}

type LookupOut struct {
	EntryOut
}

type ForgetIn struct {
	NLookup uint64
}

type SetattrValid uint32

func (v SetattrValid) Mode() bool      { return v&proto.FATTR_MODE != 0 }
func (v SetattrValid) UID() bool       { return v&proto.FATTR_UID != 0 }
func (v SetattrValid) GID() bool       { return v&proto.FATTR_GID != 0 }
func (v SetattrValid) Size() bool      { return v&proto.FATTR_SIZE != 0 }
func (v SetattrValid) Atime() bool     { return v&proto.FATTR_ATIME != 0 }
func (v SetattrValid) Mtime() bool     { return v&proto.FATTR_MTIME != 0 }
func (v SetattrValid) Fh() bool        { return v&proto.FATTR_FH != 0 }
func (v SetattrValid) AtimeNow() bool  { return v&proto.FATTR_ATIME_NOW != 0 }
func (v SetattrValid) MtimeNow() bool  { return v&proto.FATTR_MTIME_NOW != 0 }
func (v SetattrValid) LockOwner() bool { return v&proto.FATTR_LOCKOWNER != 0 }
func (v SetattrValid) Ctime() bool     { return v&proto.FATTR_CTIME != 0 }

type SetattrIn struct {
	Valid     SetattrValid
	_         uint32
	Fh        uint64
	Size      uint64
	LockOwner uint64
	Atime     uint64
	Mtime     uint64
	Ctime     uint64
	Atimensec uint32
	Mtimensec uint32
	Ctimensec uint32
	Mode      uint32
	_         uint32
	Uid       uint32
	Gid       uint32
	_         uint32
}

type SetattrOut struct {
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

type ReadlinkOut struct {
	Name string
}

type SymlinkIn struct {
	Name     string
	Linkname string
}

type SymlinkOut struct {
	EntryOut
}

// nocast
type MknodIn struct {
	Name  string
	Mode  uint32
	Rdev  uint32
	Umask uint32
	_     uint32
}

type MknodOut struct {
	EntryOut
}

type MkdirIn struct {
	Name  string
	Mode  uint32
	Umask uint32
}

type MkdirOut struct {
	EntryOut
}

type UnlinkIn struct {
	Name string
}

type RmdirIn struct {
	Name string
}

type RenameIn struct {
	Name    string
	Newname string
	Newdir  uint64
	Flags   uint32
}

type LinkIn struct {
	Oldnodeid uint64
}

type LinkOut struct {
	EntryOut
}

type OpenIn struct {
	Flags uint32
	_     uint32
}

type OpenOut struct {
	EntryOut
}

type ReadIn struct {
	Fh        uint64
	Offset    uint64
	Size      uint32
	ReadFlags uint32
	LockOwner uint64
	Flags     uint32
	_         uint32
}

type ReadOut struct {
	Data []byte
}

type LseekIn struct {
	Fh     uint64
	Offset uint64
	Whence uint32
	_      uint32
}

type LseekOut struct {
	Offset uint64
}

type CopyFileRangeIn struct {
	FhIn      uint64
	OffIn     uint64
	NodeidOut uint64
	FhOut     uint64
	OffOut    uint64
	Len       uint64
	Flags     uint64
}

type ReleaseIn struct {
	Fh           uint64
	Flags        uint32
	ReleaseFlags uint32
	LockOwner    uint64
}

// nocast
type GetxattrIn struct {
	Name string
}

// nocast
type GetxattrOut struct {
	Value []byte
}

func strlen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}
