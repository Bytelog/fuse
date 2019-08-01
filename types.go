package fuse

import (
	"syscall"
	"unsafe"

	"bytelog.org/fuse/proto"
)

type Context struct {
	req  RawRequest
	resp RawResponse

	conn *conn
	buf  []byte

	// the size of the data segment in the operation's reply.
	replySize uint32

	// indicates that the request needs no response.
	closed bool
}

func (ctx *Context) Interrupt() <-chan struct{} {
	return nil
}

func (ctx *Context) in() unsafe.Pointer {
	return unsafe.Pointer(&ctx.buf[0])
}

func (ctx *Context) out() unsafe.Pointer {
	return unsafe.Pointer(&ctx.buf[ctx.req.Header.Len])
}

var (
	OK     = error(nil)
	ENOSYS = syscall.ENOSYS
	ENOENT = syscall.ENOENT
)

var _ = [1]byte{unsafe.Sizeof(Header{}) - unsafe.Sizeof(proto.InHeader{}): 0}

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

func (header *Header) Raw() RawRequest {
	raw := RawRequest{
		Header: (*proto.InHeader)(unsafe.Pointer(header)),
	}
	if raw.Header.Len > headerInSize {
		raw.Data = unsafe.Pointer(
			uintptr(unsafe.Pointer(header)) + unsafe.Sizeof(proto.InHeader{}),
		)
	}
	return raw
}

type outHeader struct {
	proto.InHeader
}

func (header *outHeader) Raw() RawResponse {
	raw := RawResponse{
		Header: (*proto.OutHeader)(unsafe.Pointer(header)),
	}
	if raw.Header.Len > headerOutSize {
		raw.Data = unsafe.Pointer(
			uintptr(unsafe.Pointer(header)) + unsafe.Sizeof(proto.OutHeader{}),
		)
	}
	return raw
}

var _ = [1]byte{unsafe.Sizeof(InitIn{}) - unsafe.Sizeof(proto.InHeader{}) - unsafe.Sizeof(proto.InitIn{}): 0}

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
	maxReadahead        uint32
	flags               uint32
	maxBackground       uint16
	congestionThreshold uint16
	maxWrite            uint32
	timeGran            uint32
	maxPages            uint16
	_                   uint16
	_                   [8]uint32
}
