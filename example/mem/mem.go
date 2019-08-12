package mem

import (
	"math/rand"
	"sync"
	"time"

	"bytelog.org/fuse"

	"github.com/google/btree"
)

var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(0))
)

const (
	validSec  = ^uint64(0)
	validNano = uint32(999999999)
)

func randint() uint64 {
	rngMu.Lock()
	defer rngMu.Unlock()
	return rng.Uint64()
}

func now() (uint64, uint32) {
	t := time.Now()
	return uint64(t.Unix()), uint32(t.UnixNano())
}

var _ fuse.Filesystem = &FS{}

type item interface {
	Key() string
}

type key string

func (k key) Key() string {
	return string(k)
}

func (k key) Less(than btree.Item) bool {
	return string(k) < than.(item).Key()
}

type Link struct {
	Name   string
	NodeID uint64
}

func (l Link) Key() string {
	return l.Name
}

func (l Link) Less(than btree.Item) bool {
	return l.Name < than.(item).Key()
}

type Node struct {
	Attr  fuse.Attr
	XAttr [][]byte
}

type FS struct {
	fuse.Filesystem

	linkMu sync.RWMutex
	links  *btree.BTree

	nodeMu sync.RWMutex
	nodes  map[uint64]*Node
}

func New() *FS {
	return &FS{
		Filesystem: fuse.DefaultFilesystem,
		links:      btree.New(32),
		nodes:      make(map[uint64]*Node),
	}
}

func (fs *FS) Init(ctx *fuse.Context, _ *fuse.InitIn, _ *fuse.InitOut) error {
	link := &Link{
		Name:   "/",
		NodeID: 1,
	}
	sec, nsec := now()
	fs.nodes[link.NodeID] = &Node{
		Attr: fuse.Attr{
			Ino:       1,
			Size:      4096,
			Blocks:    1,
			Atime:     sec,
			Atimensec: nsec,
			Mtime:     sec,
			Mtimensec: nsec,
			Ctime:     sec,
			Ctimensec: nsec,
			Mode:      0755,
			Nlink:     1,
			Uid:       ctx.UID,
			Gid:       ctx.GID,
			Blksize:   4 * 1024,
		},
	}
	fs.links.ReplaceOrInsert(link)
	return nil
}

func (fs *FS) Access(_ *fuse.Context, in *fuse.AccessIn) error {
	return nil
}

func (fs *FS) Getattr(ctx *fuse.Context, in *fuse.GetattrIn, out *fuse.GetattrOut) error {
	// todo: respect FH & flag.

	fs.linkMu.RLock()
	defer fs.linkMu.RUnlock()

	link, ok := fs.links.Get(key("/")).(*Link)
	if !ok {
		return fuse.ENOENT
	}

	fs.nodeMu.RLock()
	defer fs.nodeMu.RUnlock()

	node := fs.nodes[link.NodeID]
	*out = fuse.GetattrOut{
		AttrValid:     validSec,
		AttrValidNsec: validNano,
		Attr:          node.Attr,
	}
	out.Attr.Uid = ctx.UID
	out.Attr.Gid = ctx.GID
	return nil
}
