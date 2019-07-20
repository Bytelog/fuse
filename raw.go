package fuse

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"unsafe"
)

// todo: open multiple connections to /dev/fuse to allow for multi-threading
// IOCTL(FUSE_DEV_IOC_CLONE, &session_fd)

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
	_ = umount(target)

	// register the mount
	dev, err := mount(target)
	if err != nil {
		return err
	}

	defer umount(target)
	return s.loop(dev)
}

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 64*1024)
	},
}

func (s *Server) loop(dev *os.File) (err error) {
	defer closeErr(dev, &err)

	const inSize = int(unsafe.Sizeof(fuse_in_header{}))
	const outSize = int(unsafe.Sizeof(fuse_out_header{}))

	// todo: deadlines?
	// todo: what's the max length we'll encounter?
	// todo: what about recovering from errors?
	buf := make([]byte, 1024)
	for {
		n, err := dev.Read(buf[:cap(buf)])
		if err != nil {
			return fmt.Errorf("reading fuse device: %v", err)
		}

		in := (*fuse_in_header)(unsafe.Pointer(&buf[0]))
		if n < inSize || n < int(in.len) {
			return fmt.Errorf("expected at least %d bytes, read %d", inSize, n)
		}

		if len(ops) < int(in.opcode) {
			return fmt.Errorf("unsupported op: %d", in.opcode)
		}

		op := ops[in.opcode]

		if op.handler == nil {
			return fmt.Errorf("unsupported op: %d", in.opcode)
		}

		if op.outSize > 0 {
			out := (*fuse_out_header)(unsafe.Pointer(&buf[op.inSize]))
			*out = fuse_out_header{
				// todo: len needs to check error
				// todo: use error
				len:    op.outSize,
				unique: in.unique,
			}
		}

		op.handler(s, unsafe.Pointer(&buf[0]), unsafe.Pointer(&buf[op.inSize]))

		if op.outSize > 0 {
			buf = buf[op.inSize : op.inSize+op.outSize]
			if _, err := dev.Write(buf); err != nil {
				return err
			}
		}
	}
}

type op struct {
	handler func(s *Server, in, out unsafe.Pointer)
	inSize  uint32
	outSize uint32
}

var ops = [...]op{
	FUSE_INIT: op{
		handler: (*Server).handleInit,
		inSize:  uint32(unsafe.Sizeof(fuse_init_in{})),
		outSize: uint32(unsafe.Sizeof(fuse_init_out{})),
	},
}

func (s *Server) handleInit(in, out unsafe.Pointer) {
	req, resp := (*fuse_init_in)(in), (*fuse_init_out)(out)

	*resp = fuse_init_out{
		header:        resp.header,
		major:         req.major,
		minor:         req.minor,
		max_readahead: req.max_readahead,
	}
}

func closeErr(closer io.Closer, err *error) {
	if err == nil {
		panic("nil error")
	}
	if cerr := closer.Close(); *err == nil {
		*err = cerr
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
