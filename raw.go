package fuse

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"bytelog.org/fuse/proto"
)

var (
	ErrBadInit       = errors.New("fuse: protocol negotiation failed")
	ErrUnsupportedOp = errors.New("fuse: unsupported op")
)

const (
	headerInSize  = unsafe.Sizeof(proto.InHeader{})
	headerOutSize = unsafe.Sizeof(proto.OutHeader{})
)

// experimental option ideas
type opts struct {
	// control whether or not file descriptor cloning is enabled
	// defaults to true
	CloneFD bool

	// how many polling threads to allow
	MaxWorkers int

	// fuse timeout. sets how long to wait for kernel reads.
	// after a timeout, a cloned descriptor is considered idle and may be
	// reclaimed
	ReadTimeout time.Duration

	// how long a Context has to respond
	WriteTimeout time.Duration

	maxReadahead uint32
	flags        uint32
	maxWrite     uint32
	timeGran     uint32
	maxPages     uint16
}

var defaultOpts = opts{
	CloneFD:      true,
	ReadTimeout:  15 * time.Second,
	WriteTimeout: time.Second,
}

type session struct {
	*logger

	fs   Filesystem
	opts opts
	errc chan error

	dev   *os.File
	minor uint32
	sem   semaphore
	ready bool

	connsMu sync.Mutex
	conns   *list.List

	starved chan struct{}
	done    chan struct{}
}

func (s *session) start(dev *os.File) error {
	c := &conn{
		session: s,
		dev:     dev,
	}

	// allow up to three attempts for protocol negotiation
	for i := 0; i < 3 && !s.ready; i++ {
		s.debugf("protocol negotiation, attempt %d", i+1)
		if err := c.accept(); err != nil {
			return err
		}
	}
	if !s.ready {
		return ErrBadInit
	}
	s.debugf("FUSE 7.%d accepted", s.minor)
	go s.control(dev)
	return nil
}

func (s *session) control(dev *os.File) {
	const (
		min = 1
		max = 1
	)

	count := 0

	for {
		if count < max {
			s.debugf("connections starved, cloning")

			s.sem.release(1)
			count++

			// clone and start connection
			f, err := clone(dev)
			if err != nil {
				panic(err)
			}

			c := &conn{
				session: s,
				dev:     f,
			}
			go c.poll()
			// todo: add to connection list
		}

		select {
		case <-s.starved:
		case <-s.done:
			return
		}
	}
}

func (s *session) close(ctx context.Context) error {
	close(s.done)
	// - close(done)
	// - if ctx has expired, close connection's file from under it
	// - close device
	return s.dev.Close()
}

type conn struct {
	*session
	dev *os.File
}

// poll is a read loop. it waits for requests from the kernel and performs some
// basic sanity checks before sending off to a handler. It is expected that the
// session closes the connection's reader to terminate poll gracefully.
func (c *conn) poll() {
	for {
		if err := c.accept(); err != nil {
			// todo: on deadline error, determine whether or not to
			// close connection
			c.logf("accept error: %v", err)
			return
		}
		select {
		case <-c.done:
			return
		default:
		}
	}
}

var ctxPool sync.Pool

func (c *conn) accept() (err error) {
	defer closeOnErr(c.dev, &err)

	if c.opts.ReadTimeout > 0 {
		deadline := time.Now().Add(c.opts.ReadTimeout)
		if err := c.dev.SetReadDeadline(deadline); err != nil {
			panic(err)
		}
	}

	if !c.sem.tryAcquire(1) {
		c.starved <- struct{}{}
	}

	ctx := c.acquireCtx()
	n, err := c.dev.Read(ctx.buf)
	if err != nil {
		return fmt.Errorf("failed read from fuse device: %w", err)
	}

	if n < int(headerInSize) || n < int(ctx.len) {
		return fmt.Errorf("unexpected request size: %d", n)
	}

	c.debugf("recv %s {ID:%d NodeID:%d UID:%d GID:%d PID:%d Len:%d}",
		ctx, ctx.Header.ID, ctx.Header.NodeID, ctx.Header.UID, ctx.Header.GID,
		ctx.Header.PID, ctx.Header.len)

	// todo: goroutine
	ctx.off = int(ctx.len)
	if err := c.handle(ctx); err != nil {
		return fmt.Errorf("%s: %w", ctx, err)
	}

	c.sem.release(1)
	c.releaseCtx(ctx)
	return nil
}

func (c *conn) handle(ctx *Context) error {
	var err error
	var size uintptr

	switch ctx.Op {
	case proto.LOOKUP:
		size = unsafe.Sizeof(LookupOut{})
		err = c.fs.Lookup(ctx, &LookupIn{Name: ctx.string()}, (*LookupOut)(ctx.outzero(size)))
	case proto.FORGET:
		c.fs.Forget(ctx, (*ForgetIn)(ctx.in()))
		return nil
	case proto.GETATTR:
		if c.minor < 9 {
			ctx.shift(int(unsafe.Sizeof(GetattrIn{})))
		}
		size = unsafe.Sizeof(GetattrOut{})
		err = c.fs.Getattr(ctx, (*GetattrIn)(ctx.in()), (*GetattrOut)(ctx.outzero(size)))
	case proto.SETATTR:
		size = unsafe.Sizeof(SetattrOut{})
		err = c.fs.Setattr(ctx, (*SetattrIn)(ctx.in()), (*SetattrOut)(ctx.outzero(size)))
	case proto.READLINK:
		out := ReadlinkOut{}
		err = c.fs.Readlink(ctx, &out)
		size = uintptr(ctx.writeString(out.Name))
	case proto.SYMLINK:
		names := ctx.strings(2)
		size = unsafe.Sizeof(SymlinkOut{})
		err = c.fs.Symlink(ctx, &SymlinkIn{Name: names[0], Linkname: names[1]}, (*SymlinkOut)(ctx.outzero(size)))
	case proto.MKNOD:
		if c.minor < 12 {
			ctx.shift(int(unsafe.Sizeof(proto.MknodIn{})) - proto.COMPAT_MKNOD_IN_SIZE)
		}
		rawIn := (*proto.MknodIn)(ctx.in())
		in := MknodIn{
			Name:  ctx.string(),
			Mode:  rawIn.Mode,
			Rdev:  rawIn.Rdev,
			Umask: rawIn.Umask,
		}
		size = unsafe.Sizeof(MknodOut{})
		err = c.fs.Mknod(ctx, &in, (*MknodOut)(ctx.outzero(size)))
	case proto.MKDIR:
		rawIn := (*proto.MkdirIn)(ctx.in())
		in := &MkdirIn{
			Name:  ctx.string(),
			Mode:  rawIn.Mode,
			Umask: rawIn.Umask,
		}
		size = unsafe.Sizeof(MkdirOut{})
		err = c.fs.Mkdir(ctx, in, (*MkdirOut)(ctx.outzero(size)))
	case proto.UNLINK:
		err = c.fs.Unlink(ctx, &UnlinkIn{Name: ctx.string()})
	case proto.RMDIR:
		err = c.fs.Rmdir(ctx, &RmdirIn{Name: ctx.string()})
	case proto.RENAME:
		names := ctx.strings(2)
		raw := (*proto.RenameIn)(ctx.in())
		err = c.fs.Rename(ctx, &RenameIn{
			Name:    names[0],
			Newname: names[1],
			Newdir:  raw.Newdir,
		})
	case proto.LINK:
		size = unsafe.Sizeof(LinkOut{})
		err = c.fs.Link(ctx, (*LinkIn)(ctx.in()), (*LinkOut)(ctx.outzero(size)))
	case proto.OPEN:
		// todo: pre-set flags for entryout requests?
		size = unsafe.Sizeof(OpenOut{})
		err = c.fs.Open(ctx, (*OpenIn)(ctx.in()), (*OpenOut)(ctx.outzero(size)))
	case proto.READ:
		// todo: compat for version 9, flocking
		// todo: len resizing, bounds checking
		err = c.fs.Read(ctx, (*ReadIn)(ctx.in()), &ReadOut{Data: ctx.outData()})
		size = uintptr(len(ctx.outData()))
	case proto.WRITE:
	case proto.STATFS:
	case proto.RELEASE:
		// todo: lock handling
	case proto.FSYNC:
	case proto.SETXATTR:
	case proto.GETXATTR:
		size = unsafe.Sizeof(GetxattrOut{})
		err = c.fs.Getxattr(ctx, (*GetxattrIn)(ctx.in()), (*GetxattrOut)(ctx.out()))
	case proto.LISTXATTR:
	case proto.REMOVEXATTR:
	case proto.FLUSH:
	case proto.INIT:
		err = ctx.handleInit((*InitIn)(ctx.in()), (*InitOut)(ctx.out()))
		switch {
		case c.minor < 5:
			size += proto.COMPAT_INIT_OUT_SIZE
		case c.minor < 23:
			size += proto.COMPAT_22_INIT_OUT_SIZE
		default:
			size += unsafe.Sizeof(InitOut{})
		}
	case proto.OPENDIR:
	case proto.READDIR:
	case proto.RELEASEDIR:
	case proto.FSYNCDIR:
	case proto.GETLK:
	case proto.SETLK:
	case proto.SETLKW:
	case proto.ACCESS:
		err = c.fs.Access(ctx, (*AccessIn)(ctx.in()))
	case proto.CREATE:
	case proto.INTERRUPT:
	case proto.BMAP:
	case proto.DESTROY:
		// todo: server shutdown
		err = c.fs.Destroy(ctx)
	case proto.IOCTL:
	case proto.POLL:
	case proto.NOTIFY_REPLY:
	case proto.BATCH_FORGET:
	case proto.FALLOCATE:
	case proto.READDIRPLUS:
	case proto.RENAME2:
		names := ctx.strings(2)
		raw := (*proto.Rename2In)(ctx.in())
		err = c.fs.Rename(ctx, &RenameIn{
			Name:    names[0],
			Newname: names[1],
			Newdir:  raw.Newdir,
			Flags:   raw.Flags,
		})
	case proto.LSEEK:
		size = unsafe.Sizeof(LseekOut{})
		err = c.fs.Lseek(ctx, (*LseekIn)(ctx.in()), (*LseekOut)(ctx.outzero(size)))
	case proto.COPY_FILE_RANGE:
		err = c.fs.CopyFileRange(ctx, (*CopyFileRangeIn)(ctx.in()))
		// todo: reply write
	default:
		return fmt.Errorf("%w: (%d)", ErrUnsupportedOp, ctx.Op)
	}

	var errno syscall.Errno
	switch {
	case errors.As(err, &errno) && errno != 0:
		size, err = 0, nil
	case err != nil:
		return fmt.Errorf("handler error in %s: %w", ctx, err)
	}

	header := ctx.outHeader()
	*header = proto.OutHeader{
		Len:    uint32(headerOutSize + size),
		Error:  -int32(errno),
		Unique: ctx.ID,
	}

	if c.opts.WriteTimeout > 0 {
		deadline := time.Now().Add(c.opts.WriteTimeout)
		if err := c.dev.SetWriteDeadline(deadline); err != nil {
			panic(err)
		}
	}

	c.debugf("send %s {ID:%d Error:%d Len:%d}",
		ctx, header.Unique, header.Error, header.Len)
	if _, err = c.dev.Write(ctx.outBuf()[:header.Len]); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	return nil
}

func (c *conn) acquireCtx() (ctx *Context) {
	v := ctxPool.Get()
	if v == nil {
		buf := make([]byte, 64*1024)
		ctx = (*Context)(unsafe.Pointer(&buf[0]))
		ctx.buf = buf[unsafe.Offsetof(ctx.Header):]
	} else {
		ctx = v.(*Context)
	}
	ctx.sess = c.session
	return ctx
}

func (c *conn) releaseCtx(r *Context) {
	ctxPool.Put(r)
}

func closeErr(closer io.Closer, err *error) {
	if err == nil {
		panic("nil error")
	}
	if e := closer.Close(); *err == nil {
		*err = e
	}
}

func closeOnErr(closer io.Closer, err *error) {
	if err == nil {
		panic("nil error")
	}
	if *err != nil {
		_ = closer.Close()
	}
}
