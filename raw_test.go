package fuse

import (
	"fmt"
	"testing"

	"golang.org/x/sys/unix"
)

func LoggingMiddleware(h HandlerFunc) HandlerFunc {
	return func(req Requester, resp Responder) {
		fmt.Printf("%s: %+v\n", req, req.Headers())
		h(req, resp)
	}
}

func TestBasic(t *testing.T) {
	handler := func(req Requester, resp Responder) {
		switch req.(type) {
		case *InitRequest, *AccessRequest:
		default:
			fmt.Println("UNHANDLED: ", req.String())
			if err := resp.Reply(unix.ENOSYS); err != nil {
				panic(err)
			}
		}
	}

	// attach logger
	handler = LoggingMiddleware(handler)

	if err := Serve(HandlerFunc(handler), "/tmp/mnt"); err != nil {
		t.Fatalf("%v", err)
	}
}
