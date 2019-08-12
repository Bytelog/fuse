package fuse

import (
	"context"
	"errors"
	"log"
	"os"
	"sync/atomic"
)

var (
	ErrServerClosed = errors.New("fuse: server closed")
)

const (
	start = iota + 1
	serve
	stop
)

type Logger interface {
	Printf(format string, args ...interface{})
}

// from: http://man7.org/linux/man-pages/man8/mount.fuse.8.html
// we may want to not support all of these. Just listing them for now.
type Options struct {
	// ErrorLog specifies an optional logger for errors encountered while
	// serving filesystem requests.
	// If nil, uses the log package's standard logger.
	ErrorLog Logger

	// DebugLog specifies an optional logger for debug information.
	// If nil, debug information will not be logged.
	DebugLog Logger

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

type Server struct {
	Options Options

	// mounted directory
	target string

	// directory created by the server
	created string

	state uint32

	*logger
	session *session
}

// Mount the FUSE filesystem and handle requests, creating the target directory
// if necessary. Blocks until the session has been initialized and is accepting
// filesystem requests.
//
// ErrServerClosed is returned after a call to Shutdown, or on subsequent calls
// to Serve.
func (s *Server) Serve(fs Filesystem, target string) (err error) {
	if !atomic.CompareAndSwapUint32(&s.state, 0, start) {
		return ErrServerClosed
	}

	if fs == nil {
		panic("fuse: nil filesystem")
	}

	s.logger = &logger{
		ErrorLog: s.Options.ErrorLog,
		DebugLog: s.Options.DebugLog,
	}

	defer func() {
		atomic.StoreUint32(&s.state, serve)
		if err != nil {
			s.debugf("session error: %s", err)
			_ = s.Shutdown(context.Background())
		}
	}()

	if _, err = os.Stat(target); errors.Is(err, os.ErrNotExist) {
		s.debugf("%s", err)
		s.debugf("mkdir %s -m 755", target)
		if err = os.Mkdir(target, 0755); err != nil {
			return err
		}
		s.created = target
	}

	// attempt to clean up any existing mounts
	// todo: abort via fusectl?
	_ = umount(target)

	s.debugf("mounting target %s", target)
	dev, err := mount(target)
	if err != nil {
		return err
	}
	s.target = target
	s.session = &session{
		logger:  s.logger,
		fs:      fs,
		opts:    defaultOpts,
		errc:    make(chan error, 1),
		sem:     semaphore{},
		done:    make(chan struct{}),
		starved: make(chan struct{}, 1),
	}
	return s.session.start(dev)
}

// Shutdown gracefully shuts down the FUSE server without interrupt any active
// connections. Shutdown stops listening to requests and waits indefinitely for
// each connection to become idle before closing it. Any directories or mounts
// that were created by Serve will be removed.
//
// After Shutdown is called, future calls to Serve and Shutdown will return
// ErrServerClosed.
//
// If the provided context expires before a graceful shutdown can complete,
// Shutdown will forcefully abort the active fuse session.
//
// Returns any error encountered from the Server's connections.
func (s *Server) Shutdown(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&s.state, serve, stop) {
		return ErrServerClosed
	}

	var errs []error

	if s.session != nil {
		s.debugf("closing session")
		errs = append(errs, s.session.close(ctx))
	}

	if s.target != "" {
		s.debugf("unmounting target %s", s.target)
		errs = append(errs, umount(s.target))
	}

	if s.created != "" {
		s.debugf("removing directory %s", s.created)
		errs = append(errs, os.Remove(s.created))
	}

	// report first error encountered
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

type logger struct {
	ErrorLog Logger
	DebugLog Logger
}

func (l *logger) logf(format string, args ...interface{}) {
	if l.ErrorLog != nil {
		l.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (l *logger) debugf(format string, args ...interface{}) {
	if l.DebugLog != nil {
		l.DebugLog.Printf(format, args...)
	}
}
