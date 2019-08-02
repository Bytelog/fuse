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
	target  string
	handler Handler
	conn    *net.UnixConn

	session *session
}

func Serve(fs Handler, target string) error {
	return (&Server{handler: fs}).Serve(target)
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
		handler: s.handler,
		opts:    defaultOpts,
		errc:    make(chan error, 1),
		sem:     semaphore{avail: 1},
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

	major        uint32
	minor        uint32
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
	handler Handler
	opts    opts
	errc    chan error

	sem semaphore
}

func (s *session) loop(dev *os.File) error {
	// todo: dynamic thread scaling. fusectl for pending requests?

	const threads = 4
	// todo: open multiple connections to /dev/fuse to allow for multi-threading
	// IOCTL(FUSE_DEV_IOC_CLONE, &session_fd)

	c := &conn{
		sess: s,
		dev:  dev,
	}

	c.poll()
	return nil
}

type conn struct {
	sess *session
	dev  *os.File
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

	if c.sess.opts.ReadTimeout > 0 {
		// todo: each time this gets reset, consider
		// bumping number of workers based on some heuristic
		deadline := time.Now().Add(c.sess.opts.ReadTimeout)
		if err := c.dev.SetReadDeadline(deadline); err != nil {
			panic(err)
		}
	}

	ctx := c.acquireCtx()
	n, err := c.dev.Read(ctx.buf[:cap(ctx.buf)])
	if err != nil {
		return fmt.Errorf("failed read from fuse device: %w", err)
	}

	if n < int(headerInSize) || n < int(ctx.req.Header.Len) {
		return fmt.Errorf("unexpected Context size: %d", n)
	}

	start := ctx.req.Header.Len
	ctx.resp = RawResponse{
		Header: (*proto.OutHeader)(unsafe.Pointer(&ctx.buf[start])),
		Data:   unsafe.Pointer(&ctx.buf[start+headerOutSize]),
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
	code := ctx.req.Header.OpCode
	if len(ops) < int(code) || ops[code] == nil {
		return fmt.Errorf("%w: %s", ErrUnsupportedOp, code)
	}

	size, err := ops[code](ctx)

	var errno syscall.Errno
	if errors.As(err, &errno) && errno != 0 {
		size, err = 0, nil
	}
	if err != nil {
		return fmt.Errorf("handler error in OP_%s: %w", code, err)
	}

	*ctx.resp.Header = proto.OutHeader{
		Unique: ctx.req.Header.Unique,
		Error:  -int32(errno),
		Len:    headerOutSize + size,
	}

	if c.sess.opts.WriteTimeout > 0 {
		deadline := time.Now().Add(c.sess.opts.WriteTimeout)
		if err := c.dev.SetWriteDeadline(deadline); err != nil {
			panic(err)
		}
	}

	p := ctx.buf[ctx.req.Header.Len : ctx.req.Header.Len+ctx.resp.Header.Len]
	if _, err = c.dev.Write(p); err != nil {
		return fmt.Errorf("failed to write response for OP_%s: %w", code, err)
	}
	return nil
}

func (c *conn) acquireCtx() (ctx *Context) {
	v := ctxPool.Get()
	if v == nil {
		ctx = &Context{
			buf: make([]byte, 64*1024),
		}
		ctx.req.Header = (*proto.InHeader)(unsafe.Pointer(&ctx.buf[0]))
		ctx.req.Data = unsafe.Pointer(&ctx.buf[headerInSize])
	} else {
		ctx = v.(*Context)
	}
	ctx.resp = RawResponse{}
	ctx.sess = c.sess
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
