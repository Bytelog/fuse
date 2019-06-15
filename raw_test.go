package fuse

import (
	"testing"
)

func TestMain(t *testing.T) {
	if err := Serve("/tmp/mnt"); err != nil {
		t.Fatalf("%v", err)
	}
	t.Fail()
}
