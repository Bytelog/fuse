package fuse

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"bytelog.org/fuse/proto"
)

var (
	ErrUnsupportedOp = errors.New("fuse: unsupported op")
)

const (
	headerInSize  = unsafe.Sizeof(proto.InHeader{})
	headerOutSize = unsafe.Sizeof(proto.OutHeader{})
)

type RawRequest struct {
	Header *proto.InHeader
	Data   []byte
}

type RawResponse struct {
	Header *proto.OutHeader
	Data   []byte
}

func (ctx *Context) handleInit(in *InitIn, out *InitOut) error {
	// todo: determine support at runtime for cansplice, vmsplice
	cansplice := true
	vmsplice := true

	if in.Major < 7 {
		return EPROTO
	}

	if in.Major > 7 {
		// allow kernel to downgrade in followup INIT
		*out = InitOut{
			major: proto.KERNEL_VERSION,
			minor: proto.KERNEL_MINOR_VERSION,
		}
		return nil
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

	*out = InitOut{
		major:               proto.KERNEL_VERSION,
		minor:               proto.KERNEL_MINOR_VERSION,
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

	if err := ctx.sess.fs.Init(ctx, in, out); err != nil {
		return err
	}

	if extra := out.Flags &^ in.Flags; extra != 0 {
		const format = "%w: flags (%X) not supported by kernel"
		return fmt.Errorf(format, EPROTO, extra)
	}

	if in.MaxReadahead < out.MaxReadahead {
		const format = "%w: MaxReadahead size (%d) too large"
		return fmt.Errorf(format, EPROTO, out.MaxReadahead)
	}

	if out.CongestionThreshold > out.MaxBackground {
		const format = "%w: CongestionThreshold exceeds MaxBackground"
		return fmt.Errorf(format, EPROTO)
	}

	if out.MaxWrite < proto.BUFFER_HEADER_SIZE {
		const format = "%w: MaxWrite (%d) must be at least %d"
		return fmt.Errorf(format, EPROTO, out.MaxWrite, proto.BUFFER_HEADER_SIZE)
	}

	if out.TimeGran < 1 || out.TimeGran > proto.MAX_TIME_GRAN {
		const format = "%w: TimeGran (%d) must be between 1ns and 1s"
		return fmt.Errorf(format, EPROTO, out.TimeGran)
	}

	if out.MaxPages > proto.MAX_MAX_PAGES {
		const format = "%w: MaxPages (%d) cannot exceed %d"
		return fmt.Errorf(format, EPROTO, out.MaxPages, proto.MAX_MAX_PAGES)
	}

	// user data has been accepted, apply it to our session
	ctx.sess.minor = in.Minor
	ctx.sess.opts.maxReadahead = out.MaxReadahead
	ctx.sess.opts.flags = out.Flags
	ctx.sess.opts.maxWrite = out.MaxWrite
	ctx.sess.opts.timeGran = out.TimeGran
	ctx.sess.opts.maxPages = out.MaxPages
	return nil
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
