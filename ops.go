package fuse

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"bytelog.org/fuse/proto"
)

var (
	ErrClosedWrite   = errors.New("fuse: invalid write on closed Context")
	ErrUnsupportedOp = errors.New("fuse: unsupported op")
	ErrParam         = errors.New("fuse: bad parameter")
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

	ctx.sess.handler.Lookup(
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
	// todo: determine support at runtime for cansplice, vmsplice
	cansplice := true
	vmsplice := true

	in, out := (*proto.InitIn)(ctx.req.Data), (*proto.InitOut)(ctx.resp.Data)
	size = uint32(unsafe.Sizeof(out))

	if in.Major < 7 {
		return 0, EPROTO
	}

	if in.Major > 7 {
		// allow kernel to downgrade in followup INIT
		*out = proto.InitOut{
			Major: proto.KERNEL_VERSION,
			Minor: proto.KERNEL_MINOR_VERSION,
		}
		return size, nil
	}

	// mask out any unsupported flags
	in.Flags &= proto.ASYNC_READ | proto.POSIX_LOCKS | proto.FILE_OPS |
		proto.ATOMIC_O_TRUNC | proto.EXPORT_SUPPORT | proto.BIG_WRITES |
		proto.DONT_MASK | proto.SPLICE_WRITE | proto.SPLICE_MOVE |
		proto.SPLICE_READ | proto.FLOCK_LOCKS | proto.HAS_IOCTL_DIR |
		proto.AUTO_INVAL_DATA | proto.DO_READDIRPLUS | proto.READDIRPLUS_AUTO |
		proto.ASYNC_DIO | proto.WRITEBACK_CACHE | proto.NO_OPEN_SUPPORT |
		proto.PARALLEL_DIROPS | proto.HANDLE_KILLPRIV | proto.POSIX_ACL |
		proto.ABORT_ERROR | proto.MAX_PAGES | proto.CACHE_SYMLINKS |
		proto.NO_OPENDIR_SUPPORT | proto.EXPLICIT_INVAL_DATA

	if !cansplice {
		in.Flags &^= proto.SPLICE_READ
	}

	if !vmsplice {
		in.Flags &^= proto.SPLICE_WRITE | proto.SPLICE_MOVE
	}

	*out = proto.InitOut{
		Major:               proto.KERNEL_VERSION,
		Minor:               proto.KERNEL_MINOR_VERSION,
		MaxReadahead:        in.MaxReadahead,
		Flags:               in.Flags,
		MaxBackground:       16,
		CongestionThreshold: 12,
		MaxWrite:            32 * uint32(os.Getpagesize()),
		TimeGran:            1,
		MaxPages:            32,
	}

	if out.Flags&proto.MAX_PAGES == 0 {
		out.MaxPages = 0
	}

	if err = ctx.sess.handler.Init(ctx,
		(*InitIn)(ctx.in()),
		(*InitOut)(ctx.out()),
	); err != nil {
		return 0, err
	}

	if extra := out.Flags &^ in.Flags; extra != 0 {
		const format = "%w: flags (%X) not supported by kernel"
		return 0, fmt.Errorf(format, EPROTO, extra)
	}

	if in.MaxReadahead < out.MaxReadahead {
		const format = "%w: MaxReadahead size (%d) too large"
		return 0, fmt.Errorf(format, EPROTO, out.MaxReadahead)
	}

	if out.CongestionThreshold > out.MaxBackground {
		const format = "%w: CongestionThreshold exceeds MaxBackground"
		return 0, fmt.Errorf(format, EPROTO)
	}

	if out.MaxWrite < proto.BUFFER_HEADER_SIZE {
		const format = "%w: MaxWrite (%d) must be at least %d"
		return 0, fmt.Errorf(format, EPROTO, out.MaxWrite, proto.BUFFER_HEADER_SIZE)
	}

	if out.TimeGran < 1 || out.TimeGran > proto.MAX_TIME_GRAN {
		const format = "%w: TimeGran (%d) must be between 1ns and 1s"
		return 0, fmt.Errorf(format, EPROTO, out.TimeGran)
	}

	if out.MaxPages > proto.MAX_MAX_PAGES {
		const format = "%w: MaxPages (%d) cannot exceed %d"
		return 0, fmt.Errorf(format, EPROTO, out.MaxPages, proto.MAX_MAX_PAGES)
	}

	// user data has been accepted, apply it to our session
	ctx.sess.opts.maxReadahead = out.MaxReadahead
	ctx.sess.opts.flags = out.Flags
	ctx.sess.opts.maxWrite = out.MaxWrite
	ctx.sess.opts.timeGran = out.TimeGran
	ctx.sess.opts.maxPages = out.MaxPages

	if in.Minor < 5 {
		return proto.COMPAT_INIT_OUT_SIZE, nil
	}
	if in.Minor < 23 {
		return proto.COMPAT_22_INIT_OUT_SIZE, nil
	}
	return size, nil
}

/*
func handleOpendir(ctx *Context) {
	out := (*proto.OpenOut)(ctx.outData())
	*out = proto.OpenOut{}

	ctx.replySize = uint32(unsafe.Sizeof(proto.OpenOut{}))
	ctx.sess.handler.Opendir(
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
	ctx.sess.handler.Readdir(
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
	ctx.sess.handler.Access(
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
	ctx.sess.handler.Destroy(
		&DestroyRequest{Request: ctx.request()},
		&DestroyResponse{response: ctx.response()},
	)
}
*/
