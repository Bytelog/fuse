package fuse

import (
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func LoggingMiddleware(h HandlerFunc) HandlerFunc {
	return func(req Requester, resp Responder) {
		fmt.Printf("%s: %+v\n", req, req.Headers())
		h(req, resp)
	}
}

func TestBasic(t *testing.T) {
	ready := make(chan struct{})

	handler := func(req Requester, resp Responder) {
		switch v := req.(type) {
		case *InitRequest:
			close(ready)
		case *LookupRequest:
			fmt.Println("request name: ", v.Name)
			assert(t, resp.Reply(unix.ENOENT))
		default:
			assert(t, resp.Reply(unix.ENOSYS))
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

	fmt.Println(names)

	// wait with a timeout, in case fuse is misbehaving. Normally we should
	// panic with i/o timeout before exiting.
	time.Sleep(5 * time.Second)
}

func assert(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%v", err)
	}
}
