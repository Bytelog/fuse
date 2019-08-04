package fuse

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func LoggingMiddleware(h HandlerFunc) HandlerFunc {
	return func(ctx *Context, req Request, resp Response) error {
		return h(ctx, req, resp)
	}
}

func TestBasic(t *testing.T) {
	ready := make(chan struct{})

	handler := func(ctx *Context, req Request, resp Response) error {
		switch req.(type) {
		case *InitIn:
			close(ready)
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

	go func() {
		if err := Serve(HandlerFunc(handler), "/tmp/mnt"); err != nil {
			panic(err)
		}
	}()

	<-ready
	f, err := os.Open("/tmp/mnt/")
	assert(t, err)

	names, err := f.Readdirnames(0)
	assert(t, err)

	fmt.Println("READDIR NAMES: ", names)

	// wait with a timeout, in case fuse is misbehaving. Normally we should
	// panic with i/o timeout before exiting.
	time.Sleep(5 * time.Second)
}

func assert(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%v", err)
	}
}
