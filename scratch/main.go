package main

import (
	"bytelog.org/fuse"
)

func main() {

	fs := fuse.DefaultFilesystem{}
	fuse.Serve(&fs, "tmp")
}
