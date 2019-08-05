package fuse

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"bytelog.org/fuse/proto"
)

// from: http://man7.org/linux/man-pages/man8/mount.fuse.8.html
// we may want to not support all of these. Just listing them for now.
type Options struct {
	// our options
	Debug bool

	// mount options
	DefaultPermissions bool
	AllowOther         bool
	RootMode           uint32
	BlockDevice        bool
	BlockSize          int
	MaxRead            int
	FD                 int
	UID                int
	GID                int
	FSName             string
	SubType            string

	// libfuse options
	AllowRoot   bool
	AutoUnmount bool // can we make this default behavior? It's convenient.
}

func (o Options) String() string {
	return ""
}

type Server struct {
	target string
	fs     Filesystem
	conn   *net.UnixConn

	session *session
}

func Serve(fs Filesystem, target string) error {
	return (&Server{fs: fs}).Serve(target)
}

func (s *Server) Serve(target string) error {

	// create the mount directory
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if err := os.Mkdir(target, 0777); err != nil {
			return err
		}
	}

	// attempt to clean up existing mounts
	// todo: abort via fusectl?
	_ = umount(target)

	// register the mount
	dev, err := mount(target)
	if err != nil {
		return err
	}

	defer func() {
		dev.Close()
		umount(target)
	}()

	sess := &session{
		fs:   s.fs,
		opts: defaultOpts,
		errc: make(chan error, 1),
		sem:  semaphore{avail: 1},
	}
	return sess.loop(dev)
}

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
	ReadTimeout:  time.Second,
	WriteTimeout: time.Second,
}

type session struct {
	fs   Filesystem
	opts opts
	errc chan error

	minor uint32
	sem   semaphore
}

func (s *session) loop(dev *os.File) error {
	// todo: dynamic thread scaling. fusectl for pending requests?

	const threads = 4
	// todo: open multiple connections to /dev/fuse to allow for multi-threading
	// IOCTL(FUSE_DEV_IOC_CLONE, &session_fd)

	c := &conn{
		session: s,
		dev:     dev,
	}

	c.poll()
	return nil
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
			panic(err)
		}
	}
}

var ctxPool sync.Pool

func (c *conn) accept() (err error) {
	defer closeOnErr(c.dev, &err)

	if c.opts.ReadTimeout > 0 {
		// todo: each time this gets reset, consider
		// bumping number of workers based on some heuristic
		deadline := time.Now().Add(c.opts.ReadTimeout)
		if err := c.dev.SetReadDeadline(deadline); err != nil {
			panic(err)
		}
	}

	ctx := c.acquireCtx()
	n, err := c.dev.Read(ctx.buf)
	if err != nil {
		return fmt.Errorf("failed read from fuse device: %w", err)
	}

	if n < int(headerInSize) || n < int(ctx.len) {
		return fmt.Errorf("unexpected request size: %d", n)
	}

	go func() {
		if err := c.handle(ctx); err != nil {
			// todo: log error
			fmt.Println(err)
			panic(err)
		}
		c.releaseCtx(ctx)
	}()

	return nil
}

func (c *conn) handle(ctx *Context) error {
	var err error
	var size uintptr

	// todo: clear output first

	switch ctx.Op {
	case proto.LOOKUP:
		size = unsafe.Sizeof(LookupOut{})
		err = c.fs.Lookup(ctx, &LookupIn{Name: ctx.strings(1)[0]}, (*LookupOut)(ctx.outzero(size)))
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
			Name:  ctx.strings(1)[0],
			Mode:  rawIn.Mode,
			Rdev:  rawIn.Rdev,
			Umask: rawIn.Umask,
		}
		size = unsafe.Sizeof(MknodOut{})
		err = c.fs.Mknod(ctx, &in, (*MknodOut)(ctx.outzero(size)))
	case proto.MKDIR:
		rawIn := (*proto.MkdirIn)(ctx.in())
		in := &MkdirIn{
			Name:  ctx.strings(1)[0],
			Mode:  rawIn.Mode,
			Umask: rawIn.Umask,
		}
		size = unsafe.Sizeof(MkdirOut{})
		err = c.fs.Mkdir(ctx, in, (*MkdirOut)(ctx.outzero(size)))
	case proto.UNLINK:
	case proto.RMDIR:
	case proto.RENAME:
	case proto.LINK:
	case proto.OPEN:
	case proto.READ:
	case proto.WRITE:
	case proto.STATFS:
	case proto.RELEASE:
	case proto.FSYNC:
	case proto.SETXATTR:
	case proto.GETXATTR:
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
	case proto.LSEEK:
	case proto.COPY_FILE_RANGE:
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

	*ctx.outHeader() = proto.OutHeader{
		Unique: ctx.ID,
		Error:  -int32(errno),
		Len:    uint32(headerOutSize + size),
	}

	if c.opts.WriteTimeout > 0 {
		deadline := time.Now().Add(c.opts.WriteTimeout)
		if err := c.dev.SetWriteDeadline(deadline); err != nil {
			panic(err)
		}
	}

	if _, err = c.dev.Write(ctx.outBuf()[:size]); err != nil {
		return fmt.Errorf("failed to write response for %s: %w", ctx, err)
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
