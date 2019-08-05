package fuse

import (
	"net"
	"os"
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
