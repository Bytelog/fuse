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
	buf  []byte
}

func (ctx *Context) Interrupt() <-chan struct{} {
	return nil
}

func (ctx *Context) String() string {
	return "OP_" + ctx.req.Header.OpCode.String()
}

func (ctx *Context) in() unsafe.Pointer {
	return unsafe.Pointer(&ctx.buf[0])
}

func (ctx *Context) out() unsafe.Pointer {
	return unsafe.Pointer(&ctx.buf[ctx.req.Header.Len])
}

var (
	ENOENT = syscall.ENOENT
	ENOSYS = syscall.ENOSYS
	EPROTO = syscall.EPROTO
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
	proto.OutHeader
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
