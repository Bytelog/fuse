package fuse

import (
	"fmt"
	"testing"

	"golang.org/x/sys/unix"
)

func TestBasic(t *testing.T) {
	handler := func(req Requester, resp Responder) {
		switch v := req.(type) {
		case *InitRequest:
			fmt.Printf("<%s>\n", req)
		case *AccessRequest:
			fmt.Println(v.UID, v.GID, v.PID)
		default:
			fmt.Println("UNHANDLED: ", req.String())
			if err := resp.Reply(unix.ENOSYS); err != nil {
				panic(err)
			}
		}
	}

	if err := Serve(HandlerFunc(handler), "/tmp/mnt"); err != nil {
		t.Fatalf("%v", err)
	}
}
