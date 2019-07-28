package fuse

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	fs := DefaultFilesystem{}
	go func() {
		if err := Serve(&fs, "/tmp/mnt"); err != nil {
			t.Fatalf("%v", err)
		}
	}()

	fmt.Println("wait")
	time.Sleep(time.Second)
	f, err := os.Open("/tmp/mnt")
	fmt.Println(f, err)
	time.Sleep(time.Second * 10)
	t.Fail()
}
