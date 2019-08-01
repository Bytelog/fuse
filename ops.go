package fuse

import (
	"errors"
	"unsafe"

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

type operation func(*Context) (size uint32, err error)

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

type RawRequest struct {
	Header *proto.InHeader
	Data   unsafe.Pointer
}

type RawResponse struct {
	Header *proto.OutHeader
	Data   unsafe.Pointer
}

/*
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
}*/

func handleInit(ctx *Context) (size uint32, err error) {
	in, out := (*proto.InitIn)(ctx.req.Data), (*proto.InitOut)(ctx.resp.Data)

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

	err = ctx.conn.sess.handler.Init(ctx, (*InitIn)(ctx.in()), (*InitOut)(ctx.out()))
	return uint32(unsafe.Sizeof(proto.InitOut{})), err
}

/*
func handleOpendir(ctx *Context) {
	out := (*proto.OpenOut)(ctx.outData())
	*out = proto.OpenOut{}

	ctx.replySize = uint32(unsafe.Sizeof(proto.OpenOut{}))
	ctx.conn.sess.handler.Opendir(
		&OpendirRequest{
			Request: ctx.request(),
			OpenIn:  (*proto.OpenIn)(ctx.data()),
		},
		&OpendirResponse{
			response: ctx.response(),
			OpenOut:  out,
		},
	)
}
*/

/*
// Intended Behavior from fuse(4): "The requested action is to read up to
//   size bytes of the file or directory, starting at offset. The bytes
//   should be returned directly following the usual reply header."
func handleReaddir(ctx *Context) {

	// todo: this doesn't actually output anything yet. add output handling.
	ctx.conn.sess.handler.Readdir(
		&ReaddirRequest{
			Request: ctx.request(),
			ReadIn:  (*proto.ReadIn)(ctx.data()),
		},
		&ReaddirResponse{
			response: ctx.response(),
		},
	)
}
*/

/*

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
*/

/*

func handleDestroy(ctx *Context) {
	ctx.conn.sess.handler.Destroy(
		&DestroyRequest{Request: ctx.request()},
		&DestroyResponse{response: ctx.response()},
	)
}
*/
