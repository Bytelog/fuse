package fuse

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
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
	_ = umount(target)

	// register the mount
	dev, err := mount(target)
	if err != nil {
		return err
	}

	defer umount(target)

	sess := &session{
		handler: s.handler,
		opts:    defaultOpts,
		errc:    make(chan error, 1),
		sem:     semaphore{avail: 1},
	}
	return sess.loop(dev)
}

// note: request struct should be pooled
// note: all active requests will need to be in a map[id]*request or something
// to facilitate interrupts.

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 64*1024)
	},
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

	// how long a request has to respond
	WriteTimeout time.Duration
}

var defaultOpts = opts{
	CloneFD: true,
}

type session struct {
	handler Handler
	opts    opts
	errc    chan error
	flags   uint32

	sem semaphore
}

func (s *session) loop(dev *os.File) error {
	// todo: dynamic thread scaling

	const threads = 4
	// todo: open multiple connections to /dev/fuse to allow for multi-threading
	// IOCTL(FUSE_DEV_IOC_CLONE, &session_fd)

	c := &conn{
		sess: s,
		dev:  dev,
	}

	if err := c.accept(); err != nil {
		panic(err)
	}
	if err := c.accept(); err != nil {
		panic(err)
	}
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

func (c *conn) accept() (err error) {
	defer closeOnErr(c.dev, &err)

	if c.sess.opts.ReadTimeout > 0 {
		// todo: each time this gets reset, consider
		// bumping number of workers based on some heuristic
		deadline := time.Now().Add(c.sess.opts.ReadTimeout)
		if err := c.dev.SetReadDeadline(deadline); err != nil {
			return err
		}
	}

	buf := pool.Get().([]byte)[:]
	n, err := c.dev.Read(buf[:cap(buf)])
	if err != nil {
		return fmt.Errorf("failed read from fuse device: %w", err)
	}

	req := &request{
		header: (*proto.InHeader)(unsafe.Pointer(&buf[0])),
		conn:   c,
		buf:    buf,
	}

	if n < int(headerInSize) || n < int(req.header.Len) {
		return fmt.Errorf("unexpected request size: %d", n)
	}

	return handle(req)
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
