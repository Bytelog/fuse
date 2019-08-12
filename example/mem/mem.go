package mem

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	"golang.org/x/sys/unix"

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
	blockSize = 64 * 1024
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
	Name   string
	Attr   fuse.Attr
	XAttrs []XAttr
}

type XAttr struct {
	Name  string
	Value []byte
}

func (n Node) Getxattr(name string) ([]byte, bool) {
	i := sort.Search(len(n.XAttrs), func(i int) bool {
		return n.XAttrs[i].Name >= name
	})
	if i < len(n.XAttrs) && n.XAttrs[i].Name == name {
		return n.XAttrs[i].Value, true
	}
	return nil, false
}

func (n *Node) Setxattr(name string, value []byte) {
	i := sort.Search(len(n.XAttrs), func(i int) bool {
		return n.XAttrs[i].Name >= name
	})
	if i < len(n.XAttrs) && n.XAttrs[i].Name == name {
		n.XAttrs[i].Value = value
	}
	n.XAttrs = append(n.XAttrs, XAttr{})
	copy(n.XAttrs[i+1:], n.XAttrs[i:])
	n.XAttrs[i] = XAttr{
		Name:  name,
		Value: value,
	}
}

type FS struct {
	fuse.Filesystem

	linkMu sync.RWMutex
	links  *btree.BTree

	nodeMu sync.RWMutex
	nodes  map[uint64]*Node

	openMu sync.RWMutex
	open   map[uint64]*Node
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
		Name: "/",
		Attr: fuse.Attr{
			Ino:       1,
			Atime:     sec,
			Atimensec: nsec,
			Mtime:     sec,
			Mtimensec: nsec,
			Ctime:     sec,
			Ctimensec: nsec,
			Mode:      unix.S_IFDIR | 0755,
			Nlink:     1,
			Uid:       ctx.UID,
			Gid:       ctx.GID,
			Blksize:   blockSize,
		},
	}
	fs.links.ReplaceOrInsert(link)
	return nil
}

func (fs *FS) Access(_ *fuse.Context, in *fuse.AccessIn) error {
	return nil
}

// Get file attributes.
// ctx.NodeID: Node
// in.Fh: File Handle (if non-zero)
// out.Size may be ignored by the kernel if writeback caching is enabled.
func (fs *FS) Getattr(ctx *fuse.Context, in *fuse.GetattrIn, out *fuse.GetattrOut) error {
	var node *Node

	if in.Fh > 0 {
		fs.openMu.RLock()
		node = fs.open[in.Fh]
		fs.openMu.RUnlock()
	} else {
		fs.nodeMu.RLock()
		node = fs.nodes[ctx.NodeID]
		fs.nodeMu.RUnlock()
	}

	*out = fuse.GetattrOut{
		AttrValid:     validSec,
		AttrValidNsec: validNano,
		Attr:          node.Attr,
	}
	return nil
}

// Get an extended attribute of a node
// in.Size: maximum size of the value to send
func (fs *FS) Getxattr(ctx *fuse.Context, in *fuse.GetxattrIn, out *fuse.GetxattrOut) error {
	fs.nodeMu.RLock()
	defer fs.nodeMu.RUnlock()

	node, ok := fs.nodes[ctx.NodeID]
	if !ok {
		return fuse.ENOENT
	}

	out.Value, _ = node.Getxattr(in.Name)
	return nil
}

// Look up an entry by name and get its attributes.
// ctx.NodeID: Parent Directory
// in.Name: Entry Name
func (fs *FS) Lookup(ctx *fuse.Context, in *fuse.LookupIn, out *fuse.LookupOut) error {
	fs.nodeMu.RLock()
	defer fs.nodeMu.RUnlock()

	parent, ok := fs.nodes[ctx.NodeID]
	if !ok {
		return fuse.ENOENT
	}

	fs.linkMu.RLock()
	defer fs.linkMu.RUnlock()

	link, ok := fs.links.Get(key(parent.Name + in.Name)).(*Link)
	if !ok {
		return fuse.ENOENT
	}

	node, ok := fs.nodes[link.NodeID]
	*out = fuse.LookupOut{
		EntryOut: fuse.EntryOut{
			Nodeid:         link.NodeID,
			Generation:     1, // todo: uniqueness
			EntryValid:     validSec,
			EntryValidNsec: validNano,
			AttrValid:      validSec,
			AttrValidNsec:  validNano,
			Attr:           node.Attr,
		},
	}

	return fuse.ENOENT
}
