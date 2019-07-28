package fuse

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"bytelog.org/fuse/proto"
)

var (
	ErrWriteAfterClose = errors.New("fuse: invalid write on closed request")
	ErrUnsupportedOp   = errors.New("fuse: unsupported op")
)

const (
	headerInSize  = uint32(unsafe.Sizeof(proto.InHeader{}))
	headerOutSize = uint32(unsafe.Sizeof(proto.OutHeader{}))
)

var _ = [1]byte{unsafe.Sizeof(Header{}) - unsafe.Sizeof(proto.InHeader{}): 0}

type Header struct {
	len  uint32
	op   proto.OpCode
	ID   uint64
	Node uint64
	UID  uint32
	GID  uint32
	PID  uint32
	_    uint32
}

type request struct {
	header *proto.InHeader

	w      io.Writer
	conn   *conn
	buf    []byte
	closed uint32
}

func (r *request) Header() *Header {
	return (*Header)(unsafe.Pointer(r.header))
}

func (r *request) String() string {
	return r.header.OpCode.String()
}

func (r *request) Interrupt() <-chan struct{} {
	return nil
}

func (r *request) data() unsafe.Pointer {
	return unsafe.Pointer(&r.buf[headerInSize])
}

func (r *request) outHeader() *proto.OutHeader {
	return (*proto.OutHeader)(unsafe.Pointer(&r.buf[r.header.Len]))
}

func (r *request) outData() unsafe.Pointer {
	return unsafe.Pointer(&r.buf[r.header.Len+headerOutSize])
}

func (r *request) reply(size uint32, err error) error {
	// prevent accidental double writes
	if !atomic.CompareAndSwapUint32(&r.closed, 0, 1) {
		return ErrWriteAfterClose
	}

	defer pool.Put(r.buf)

	header := r.outHeader()
	*header = proto.OutHeader{
		Len:    headerOutSize,
		Unique: r.header.Unique,
	}

	if err == nil {
		header.Len += size
	} else {
		// todo: map error to errno
		header.Error = -int32(unix.ENOSYS)
	}

	if r.conn.sess.opts.WriteTimeout > 0 {
		deadline := time.Now().Add(r.conn.sess.opts.WriteTimeout)
		if err := r.conn.dev.SetWriteDeadline(deadline); err != nil {
			return err
		}
	}

	_, err = r.conn.dev.Write(r.buf[r.header.Len : r.header.Len+header.Len])
	return err
}

func (r *request) responder(size uintptr) responder {
	return responder{request: r, size: uint32(size)}
}

type responder struct {
	*request
	size uint32
}

func (r *responder) Reply() error {
	return r.reply(r.size, nil)
}

func (r *responder) ReplyErr(err error) error {
	return r.reply(0, err)
}

type InitRequest struct {
	Major        uint32
	Minor        uint32
	MaxReadahead uint32
	Flags        uint32
	responder
}

type DestroyRequest struct {
	responder
}

type ForgetRequest struct {
	*request
}

func handle(req *request) error {
	code := req.header.OpCode
	if int(code) < len(ops) && ops[code] != nil {
		ops[code](req)
		return nil
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedOp, code)
}

// rules imposed on operation handler:
// - must not retain references
type operation func(*request)

var ops = [...]operation{
	proto.INIT: handleInit,
}

// / todo: consider concurrent access?
func handleInit(req *request) {
	in, out := (*proto.InitIn)(req.data()), (*proto.InitOut)(req.outData())

	*out = proto.InitOut{
		Major: proto.KERNEL_VERSION,
		Minor: proto.KERNEL_MINOR_VERSION,
	}

	if in.Major < 7 {
		// error: unsupported proto version
		return
	}

	if in.Major > 7 {
		// allow kernel to downgrade proto version
		return
	}

	if in.Minor >= 6 {
		// todo: only allow downgrading of max_readahead once set
		// todo: determine bufsize from max_pages?

		out.MaxReadahead = in.MaxReadahead
		req.conn.sess.flags |= in.Flags
	}

	if in.Minor >= 14 {
		// todo: determine if splice is supported, vmsplice?
		const canSplice = true
		const vmSplice = true

		// wip

	}

	out.TimeGran = 1000000000
	out.CongestionThreshold = 10
	out.MaxBackground = 10
	out.MaxPages = 10
	out.MaxReadahead = 65536
	out.MaxWrite = 65536

	req.conn.sess.handler.Init(&InitRequest{
		Major:        in.Major,
		Minor:        in.Minor,
		MaxReadahead: in.MaxReadahead,
		Flags:        in.Flags,
		responder:    req.responder(unsafe.Sizeof(proto.InitOut{})),
	})
}
