package fuse

import (
	"testing"
)

func TestBasic(t *testing.T) {
	fs := DefaultFilesystem{}
	if err := Serve(&fs, "/tmp/mnt"); err != nil {
		t.Fatalf("%v", err)
	}
	t.Fail()
}
