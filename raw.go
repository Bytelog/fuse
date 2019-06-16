package fuse

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"unsafe"
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
	conn   *net.UnixConn
}

func Serve(target string) error {
	return (&Server{}).Serve(target)
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
	// todo: deadlines?

	// fuse_in_header + fuse_init_in

	for {
		b := pool.Get().([]byte)
		n, _ := dev.Read(b[:cap(b)])
		b = b[:n]

		in := (*fuse_in_header)(unsafe.Pointer(&b[0]))
		sz_skip := int(unsafe.Sizeof(fuse_in_header{}))

		switch in.opcode {
		case FUSE_INIT:
			init := (*fuse_init_in)(unsafe.Pointer(&b[sz_skip]))
			fmt.Printf("%+v", init)
		default:
			panic("unsupported")
		}

		return nil
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
