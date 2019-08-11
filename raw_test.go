package fuse

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"testing"
	"time"
)

type loggy struct {
	prefix string
}

func (l loggy) Printf(format string, args ...interface{}) {
	_, _ = fmt.Printf(l.prefix+format+"\n", args...)
}

func setup(fs Filesystem, target string) (func() <-chan error, error) {
	srv := &Server{
		Options: Options{
			ErrorLog: loggy{prefix: "== error: ",},
			DebugLog: loggy{prefix: " - debug: ",},
		},
	}
	if err := srv.Serve(fs, target); err != nil {
		return nil, err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	done := make(chan struct{}, 1)
	errc := make(chan error, 1)

	go func() {
		ctx := context.Background()
		var cancel func()
		select {
		case <-sig:
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		case <-done:
			ctx, cancel = context.WithTimeout(ctx, time.Second)
			defer cancel()
		}
		errc <- srv.Shutdown(ctx)
	}()

	shutdown := func() <-chan error {
		close(done)
		return errc
	}
	return shutdown, nil
}

func LoggingMiddleware(h HandlerFunc) HandlerFunc {
	return func(ctx *Context, req Request, resp Response) error {
		return h(ctx, req, resp)
	}
}

func TestBasic(t *testing.T) {
	handler := func(ctx *Context, req Request, resp Response) error {
		switch req.(type) {
		case *InitIn:
			return nil
		// case *LookupIn:
		//	return ENOENT
		default:
			fmt.Println("RESPONDING ENOSYS to", ctx)
			return ENOSYS
		}
	}

	// attach logger
	handler = LoggingMiddleware(handler)

	shutdown, err := setup(HandlerFunc(handler), "/tmp/mount")
	assert(t, err)
	defer func() { _ = <-shutdown() }()

	/*
		f, err := os.Open("/tmp/mount/")
		assert(t, err)

		names, err := f.Readdirnames(0)
		assert(t, err)

		fmt.Println("READDIR NAMES: ", names)
	*/

	time.Sleep(10 * time.Second)
	t.Fail()
}

func assert(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%v", err)
	}
}
