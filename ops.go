package fuse

import (
	"errors"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"bytelog.org/fuse/proto"
)

var (
	ErrClosedWrite   = errors.New("fuse: invalid write on closed Context")
	ErrUnsupportedOp = errors.New("fuse: unsupported op")
)

const (
	headerInSize  = uint32(unsafe.Sizeof(proto.InHeader{}))
	headerOutSize = uint32(unsafe.Sizeof(proto.OutHeader{}))
)

type operation func(*Context)

var ops = [...]operation{
	proto.LOOKUP: handleLookup,
	// proto.FORGET:          handleForget,
	// proto.GETATTR:         handleGetattr,
	// proto.SETATTR:         handleSetattr,
	// proto.READLINK:        handleReadlink,
	// proto.SYMLINK:         handleSymlink,
	// proto.MKNOD:           handleMknod,
	// proto.MKDIR:           handleMkdir,
	// proto.UNLINK:          handleUnlink,
	// proto.RMDIR:           handleRmdir,
	// proto.RENAME:          handleRename,
	// proto.LINK:            handleLink,
	// proto.OPEN:            handleOpen,
	// proto.READ:            handleRead,
	// proto.WRITE:           handleWrite,
	// proto.STATFS:          handleStates,
	// proto.RELEASE:         handleRelease,
	// proto.FSYNC:           handleFsync,
	// proto.SETXATTR:        handleSetxattr,
	// proto.GETXATTR:        handleGetxattr,
	// proto.LISTXATTR:       handleListxattr,
	// proto.REMOVEXATTR:     handleRemovexattr,
	// proto.FLUSH:           handleFlush,
	proto.INIT:    handleInit,
	proto.OPENDIR: handleOpendir,
	proto.READDIR: handleReaddir,
	// proto.RELEASEDIR:      handleReleasedir,
	// proto.FSYNCDIR:        handleFsyncdir,
	// proto.GETLK:           handleGetlk,
	// proto.SETLK:           handleSetlk,
	// proto.SETLKW:          handleSetlk,
	proto.ACCESS: handleAccess,
	// proto.CREATE:          handleCreate,
	// proto.INTERRUPT:       handleInterrupt,
	// proto.BMAP:            handleBmap,
	// proto.DESTROY: handleDestroy,
	// proto.IOCTL:           handleIoctl,
	// proto.POLL:            handlePoll,
	// proto.NOTIFY_REPLY:    handleNotifyReply,
	// proto.BATCH_FORGET:    handleBatchForget,
	// proto.FALLOCATE:       handleFallocate,
	// proto.READDIRPLUS:     handleReaddirplus,
	// proto.RENAME2:         handleRename2,
	// proto.LSEEK:           handleLseek,
	// proto.COPY_FILE_RANGE: handleCopyFileRange,
}

var _ = [1]byte{unsafe.Sizeof(Header{}) - unsafe.Sizeof(proto.InHeader{}): 0}

type Header struct {
	len  uint32
	code proto.OpCode
	ID   uint64
	Node uint64
	UID  uint32
	GID  uint32
	PID  uint32
	_    uint32
}

type Context struct {
	*Header

	conn      *conn
	buf       []byte
	replySize uint32
	closed    bool
}

func (ctx *Context) reply(err unix.Errno) error {
	if ctx.closed {
		return ErrClosedWrite
	}

	header := ctx.outHeader()
	*header = proto.OutHeader{
		Len:    headerOutSize,
		Unique: ctx.Header.ID,
	}

	if err == 0 {
		header.Len += ctx.replySize
	} else {
		header.Error = -int32(err)
	}

	if ctx.conn.sess.opts.WriteTimeout > 0 {
		deadline := time.Now().Add(ctx.conn.sess.opts.WriteTimeout)
		if err := ctx.conn.dev.SetWriteDeadline(deadline); err != nil {
			panic(err)
		}
	}

	p := ctx.buf[ctx.Header.len : ctx.Header.len+header.Len]
	if _, err := ctx.conn.dev.Write(p); err != nil {
		return err
	}
	ctx.closed = true
	return nil
}

func (ctx *Context) data() unsafe.Pointer {
	return unsafe.Pointer(&ctx.buf[headerInSize])
}

func (ctx *Context) bytes() []byte {
	return ctx.buf[headerInSize:ctx.Header.len]
}

func (ctx *Context) outHeader() *proto.OutHeader {
	return (*proto.OutHeader)(unsafe.Pointer(&ctx.buf[ctx.Header.len]))
}

func (ctx *Context) outData() unsafe.Pointer {
	return unsafe.Pointer(&ctx.buf[ctx.Header.len+headerOutSize])
}

func (ctx *Context) request() *Request {
	return (*Request)(unsafe.Pointer(ctx))
}

func (ctx *Context) response() *response {
	return (*response)(unsafe.Pointer(ctx))
}

type Request struct {
	Context
}

func (req *Request) Headers() *Header {
	return (*Header)(unsafe.Pointer(req.Header))
}

func (req *Request) String() string {
	return "REQUEST_" + req.Header.code.String()
}

func (req *Request) Interrupt() <-chan struct{} {
	return nil
}

type response struct {
	Context
}

func (resp *response) String() string {
	return "RESPONSE_" + resp.Header.code.String()
}

func (resp *response) Reply(err unix.Errno) error {
	return resp.reply(err)
}

type LookupRequest struct {
	*Request
	Name string
}

type LookupResponse struct {
	*response
	*proto.EntryOut
}

func handleLookup(ctx *Context) {
	in, out := ctx.bytes(), (*proto.EntryOut)(ctx.outData())
	if len(in) == 0 || in[len(in)-1] != 0 {
		// todo: error handling at this context
		panic("bad string")
	}

	// zero out the entry data
	*out = proto.EntryOut{}

	ctx.conn.sess.handler.Lookup(
		&LookupRequest{
			Request: ctx.request(),
			Name:    string(in[:len(in)-1]),
		},
		&LookupResponse{
			response: ctx.response(),
			EntryOut: out,
		},
	)
}

type InitRequest struct {
	*Request
	Major        uint32
	Minor        uint32
	MaxReadahead uint32
	Flags        uint32
}

type InitResponse struct {
	*response
}

func handleInit(ctx *Context) {
	in, out := (*proto.InitIn)(ctx.data()), (*proto.InitOut)(ctx.outData())

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
		ctx.conn.sess.flags |= in.Flags
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

	ctx.replySize = uint32(unsafe.Sizeof(proto.InitOut{}))
	ctx.conn.sess.handler.Init(
		&InitRequest{
			Request:      ctx.request(),
			Major:        in.Major,
			Minor:        in.Minor,
			MaxReadahead: in.MaxReadahead,
			Flags:        in.Flags,
		},
		&InitResponse{
			response: ctx.response(),
		},
	)
}

type OpendirRequest struct {
	*Request
	Flags uint32
}

type OpendirResponse struct {
	*response
	*proto.OpenOut
}

func handleOpendir(ctx *Context) {
	in, out := (*proto.OpenIn)(ctx.data()), (*proto.OpenOut)(ctx.outData())

	// zero out the open response data
	*out = proto.OpenOut{}

	ctx.conn.sess.handler.Opendir(
		&OpendirRequest{
			Request: ctx.request(),
			Flags:   in.Flags,
		},
		&OpendirResponse{
			response: ctx.response(),
			OpenOut:  out,
		},
	)
}

type ReaddirRequest struct {
	*Request
	Fh        uint64
	Offset    uint64
	Size      uint32
	ReadFlags uint32
	LockOwner uint64
	Flags     uint32
}

type ReaddirResponse struct {
	*response
}

// Intended Behavior from fuse(4): "The requested action is to read up to
//   size bytes of the file or directory, starting at offset. The bytes
//   should be returned directly following the usual reply header."
func handleReaddir(ctx *Context) {
	in := (*proto.ReadIn)(ctx.data())

	// todo: this doesn't actually output anything yet. add output handling.
	ctx.conn.sess.handler.Readdir(
		&ReaddirRequest{
			Request:   ctx.request(),
			Fh:        in.Fh,
			Offset:    in.Offset,
			Size:      in.Size,
			ReadFlags: in.ReadFlags,
			LockOwner: in.LockOwner,
			Flags:     in.Flags,
		},
		&ReaddirResponse{
			response: ctx.response(),
		},
	)
}

type AccessRequest struct {
	*Request
	Mask uint32
}

type AccessResponse struct {
	*response
}

func handleAccess(ctx *Context) {
	in := (*proto.AccessIn)(ctx.data())
	ctx.conn.sess.handler.Access(
		&AccessRequest{
			Request: ctx.request(),
			Mask:    in.Mask,
		},
		&AccessResponse{response: ctx.response()},
	)
}

type DestroyRequest struct {
	*Request
}

type DestroyResponse struct {
	*response
}

func handleDestroy(ctx *Context) {
	ctx.conn.sess.handler.Destroy(
		&DestroyRequest{Request: ctx.request()},
		&DestroyResponse{response: ctx.response()},
	)
}
