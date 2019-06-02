package fuse

import (
	"testing"
)

func TestMain(t *testing.T) {
	if err := mount("/tmp/mnt", ""); err != nil {
		t.Fatalf("%v", err)
	}
}
