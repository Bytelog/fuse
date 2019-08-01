// Port of fuse_kernel.h to the Go programming language.
//
// Modifications include naming transformations to account for go conventions,
// fmt.Stringer implementations, and go implementations of several macros.
//
// Modifications by Dylan Allbee and released into public domain.
// Use of this source code is governed by the original license:

/*
   This file defines the kernel interface of FUSE
   Copyright (C) 2001-2008  Miklos Szeredi <miklos@szeredi.hu>

   This program can be distributed under the terms of the GNU GPL.
   See the file COPYING.

   This -- and only this -- header file may also be distributed under
   the terms of the BSD Licence as follows:

   Copyright (C) 2001-2007 Miklos Szeredi. All rights reserved.

   Redistribution and use in source and binary forms, with or without
   modification, are permitted provided that the following conditions
   are met:
   1. Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
   2. Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.

   THIS SOFTWARE IS PROVIDED BY AUTHOR AND CONTRIBUTORS ``AS IS'' AND
   ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
   IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
   ARE DISCLAIMED.  IN NO EVENT SHALL AUTHOR OR CONTRIBUTORS BE LIABLE
   FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
   DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
   OR SERVICES LOSS OF USE, DATA, OR PROFITS OR BUSINESS INTERRUPTION)
   HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
   LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
   OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
   SUCH DAMAGE.
*/

/*
 * This file defines the kernel interface of FUSE
 *
 * Protocol changelog:
 *
 * 7.9:
 *  - new GetattrIn input argument of GETATTR
 *  - add LkFlags in LkIn
 *  - add LockOwner field to SetattrIn, ReadIn and WriteIn
 *  - add Blksize field to Attr
 *  - add file flags field to ReadIn and WriteIn
 *  - Add ATIME_NOW and MTIME_NOW flags to SetattrIn
 *
 * 7.10
 *  - add nonseekable open flag
 *
 * 7.11
 *  - add IOCTL message
 *  - add unsolicited notification support
 *  - add POLL message and NOTIFY_POLL notification
 *
 * 7.12
 *  - add umask flag to input argument of create, mknod and mkdir
 *  - add notification messages for invalidation of inodes and
 *    directory entries
 *
 * 7.13
 *  - make max number of background requests and congestion threshold
 *    tunables
 *
 * 7.14
 *  - add splice support to fuse device
 *
 * 7.15
 *  - add store notify
 *  - add retrieve notify
 *
 * 7.16
 *  - add BATCH_FORGET request
 *  - IOCTL_UNRESTRICTED shall now return with array of 'IoctlIovec'
 *  - add IOCTL_32BIT flag
 *
 * 7.17
 *  - add FLOCK_LOCKS and RELEASE_FLOCK_UNLOCK
 *
 * 7.18
 *  - add IOCTL_DIR flag
 *  - add NOTIFY_DELETE
 *
 * 7.19
 *  - add FALLOCATE
 *
 * 7.20
 *  - add AUTO_INVAL_DATA
 *
 * 7.21
 *  - add READDIRPLUS
 *  - send the requested events in POLL request
 *
 * 7.22
 *  - add ASYNC_DIO
 *
 * 7.23
 *  - add WRITEBACK_CACHE
 *  - add TimeGran to InitOut
 *  - add reserved space to InitOut
 *  - add FATTR_CTIME
 *  - add Ctime and Ctimensec to SetattrIn
 *  - add RENAME2 request
 *  - add NO_OPEN_SUPPORT flag
 *
 *  7.24
 *  - add LSEEK for SEEK_HOLE and SEEK_DATA support
 *
 *  7.25
 *  - add PARALLEL_DIROPS
 *
 *  7.26
 *  - add HANDLE_KILLPRIV
 *  - add POSIX_ACL
 *
 *  7.27
 *  - add ABORT_ERROR
 *
 *  7.28
 *  - add COPY_FILE_RANGE
 *  - add FOPEN_CACHE_DIR
 *  - add MAX_PAGES, add MaxPages to InitOut
 *  - add CACHE_SYMLINKS
 *
 *  7.29
 *  - add NO_OPENDIR_SUPPORT flag
 *
 *  7.30
 *  - add EXPLICIT_INVAL_DATA
 *  - add IOCTL_COMPAT_X32
 *
 *  7.31
 *  - add WRITE_KILL_PRIV flag
 */

package proto

import (
	"fmt"
	"unsafe"
)

// Version negotiation:
//
// Both the kernel and userspace send the version they support in the INIT
// request and reply respectively.
//
// If the major versions match then both shall use the smallest of the two minor
// versions for communication.
//
// If the kernel supports a larger major version, then userspace shall reply
// with the major version it supports, ignore the rest of the INIT message and
// expect a new INIT message from the kernel with a matching major version.
//
// If the library supports a larger major version, then it shall fall back to
// the major protocol version sent by the kernel for communication and reply
// with that major version (and an arbitrary supported minor version).

// Version number of this interface
const KERNEL_VERSION = 7

// Minor version number of this interface
const KERNEL_MINOR_VERSION = 31

// The node ID of the root inode
const ROOT_ID = 1

// Make sure all structures are padded to 64bit boundary, so 32bit userspace
// works under 64bit kernels

type Attr struct {
	Ino       uint64
	Size      uint64
	Blocks    uint64
	Atime     uint64
	Mtime     uint64
	Ctime     uint64
	Atimensec uint32
	Mtimensec uint32
	Ctimensec uint32
	Mode      uint32
	Nlink     uint32
	Uid       uint32
	Gid       uint32
	Rdev      uint32
	Blksize   uint32
	_         uint32
}

type KStatFS struct {
	Blocks  uint64
	Bfree   uint64
	Bavail  uint64
	Files   uint64
	Ffree   uint64
	Bsize   uint32
	Namelen uint32
	Frsize  uint32
	_       uint32
	_       [6]uint32
}

type FileLock struct {
	Start uint64
	End   uint64
	Type  uint32
	Pid   uint32 // tgid
}

// Bitmasks for SetattrIn.Valid
const (
	FATTR_MODE      = 1 << 0
	FATTR_UID       = 1 << 1
	FATTR_GID       = 1 << 2
	FATTR_SIZE      = 1 << 3
	FATTR_ATIME     = 1 << 4
	FATTR_MTIME     = 1 << 5
	FATTR_FH        = 1 << 6
	FATTR_ATIME_NOW = 1 << 7
	FATTR_MTIME_NOW = 1 << 8
	FATTR_LOCKOWNER = 1 << 9
	FATTR_CTIME     = 1 << 10
)

// Flags returned by the OPEN request
const (
	// bypass page cache for this open file
	FOPEN_DIRECT_IO = 1 << 0

	// don't invalidate the data cache on open
	FOPEN_KEEP_CACHE = 1 << 1

	// the file is not seekable
	FOPEN_NONSEEKABLE = 1 << 2

	// allow caching this directory
	FOPEN_CACHE_DIR = 1 << 3

	// the file is stream-like (no file position at all)
	FOPEN_STREAM = 1 << 4
)

// INIT request/reply flags
const (
	// asynchronous read requests
	ASYNC_READ = 1 << 0

	// remote locking for POSIX file locks
	POSIX_LOCKS = 1 << 1

	// kernel sends file handle for fstat, etc... (not yet supported)
	FILE_OPS = 1 << 2

	// handles the O_TRUNC open flag in the filesystem
	ATOMIC_O_TRUNC = 1 << 3

	// filesystem handles lookups of "." and ".."
	EXPORT_SUPPORT = 1 << 4

	// filesystem can handle write size larger than 4kB
	BIG_WRITES = 1 << 5

	// don't apply umask to file mode on create operations
	DONT_MASK = 1 << 6

	// kernel supports splice write on the device
	SPLICE_WRITE = 1 << 7

	// kernel supports splice move on the device
	SPLICE_MOVE = 1 << 8

	// kernel supports splice read on the device
	SPLICE_READ = 1 << 9

	// remote locking for BSD style file locks
	FLOCK_LOCKS = 1 << 10

	// kernel supports ioctl on directories
	HAS_IOCTL_DIR = 1 << 11

	// automatically invalidate cached pages
	AUTO_INVAL_DATA = 1 << 12

	// do READDIRPLUS READDIR+LOOKUP in one)
	DO_READDIRPLUS = 1 << 13

	// adaptive readdirplus
	READDIRPLUS_AUTO = 1 << 14

	// asynchronous direct I/O submission
	ASYNC_DIO = 1 << 15

	// use writeback cache for buffered writes
	WRITEBACK_CACHE = 1 << 16

	// kernel supports zero-message opens
	NO_OPEN_SUPPORT = 1 << 17

	// allow parallel lookups and readdir
	PARALLEL_DIROPS = 1 << 18

	// fs handles killing suid/sgid/cap on write/chown/trunc
	HANDLE_KILLPRIV = 1 << 19

	// filesystem supports posix acls
	POSIX_ACL = 1 << 20

	// reading the device after abort returns ECONNABORTED
	ABORT_ERROR = 1 << 21

	// InitOut.MaxPages contains the max number of req pages
	MAX_PAGES = 1 << 22

	// cache READLINK responses
	CACHE_SYMLINKS = 1 << 23

	// kernel supports zero-message opendir
	NO_OPENDIR_SUPPORT = 1 << 24

	// only invalidate cached pages on explicit request
	EXPLICIT_INVAL_DATA = 1 << 25
)

// CUSE INIT request/reply flags
const (
	// use unrestricted ioctl
	CUSE_UNRESTRICTED_IOCTL = 1 << 0
)

// Release flags
const (
	RELEASE_FLUSH        = 1 << 0
	RELEASE_FLOCK_UNLOCK = 1 << 1
)

// Getattr flags
const GETATTR_FH = 1 << 0

// Lock flags
const LK_FLOCK = 1 << 0

// WRITE flags
const (
	// delayed write from page cache, file handle is guessed
	WRITE_CACHE = 1 << 0

	// lock_owner field is valid
	WRITE_LOCKOWNER = 1 << 1

	// kill suid and sgid bits
	WRITE_KILL_PRIV = 1 << 2
)

// Read flags
const READ_LOCKOWNER = 1 << 1

// Ioctl flags
const (
	// 32bit compat ioctl on 64bit machine
	IOCTL_COMPAT = 1 << 0

	// not restricted to well-formed ioctls, retry allowed
	IOCTL_UNRESTRICTED = 1 << 1

	// retry with new iovecs
	IOCTL_RETRY = 1 << 2

	// 32bit ioctl
	IOCTL_32BIT = 1 << 3

	// is a directory
	IOCTL_DIR = 1 << 4

	// x32 compat ioctl on 64bit machine (64bit time_t)
	IOCTL_COMPAT_X32 = 1 << 5

	// maximum of in_iovecs + out_iovecs
	IOCTL_MAX_IOV = 256
)

// Poll flags
const (
	// request poll notify
	POLL_SCHEDULE_NOTIFY = 1 << 0
)

// Fsync flags
const (
	// Sync data only, not metadata
	FSYNC_FDATASYNC = 1 << 0
)

type OpCode uint32

func (code OpCode) String() string {
	if int(code) < len(opCodeText) && opCodeText[code] != "" {
		return opCodeText[code]
	}
	if code == CUSE_INIT {
		return "CUSE_INIT"
	}
	return fmt.Sprintf("UNKNOWN(%d)", code)
}

// FUSE operation codes
const (
	LOOKUP          = OpCode(1)
	FORGET          = OpCode(2) // no reply
	GETATTR         = OpCode(3)
	SETATTR         = OpCode(4)
	READLINK        = OpCode(5)
	SYMLINK         = OpCode(6)
	MKNOD           = OpCode(8)
	MKDIR           = OpCode(9)
	UNLINK          = OpCode(10)
	RMDIR           = OpCode(11)
	RENAME          = OpCode(12)
	LINK            = OpCode(13)
	OPEN            = OpCode(14)
	READ            = OpCode(15)
	WRITE           = OpCode(16)
	STATFS          = OpCode(17)
	RELEASE         = OpCode(18)
	FSYNC           = OpCode(20)
	SETXATTR        = OpCode(21)
	GETXATTR        = OpCode(22)
	LISTXATTR       = OpCode(23)
	REMOVEXATTR     = OpCode(24)
	FLUSH           = OpCode(25)
	INIT            = OpCode(26)
	OPENDIR         = OpCode(27)
	READDIR         = OpCode(28)
	RELEASEDIR      = OpCode(29)
	FSYNCDIR        = OpCode(30)
	GETLK           = OpCode(31)
	SETLK           = OpCode(32)
	SETLKW          = OpCode(33)
	ACCESS          = OpCode(34)
	CREATE          = OpCode(35)
	INTERRUPT       = OpCode(36)
	BMAP            = OpCode(37)
	DESTROY         = OpCode(38)
	IOCTL           = OpCode(39)
	POLL            = OpCode(40)
	NOTIFY_REPLY    = OpCode(41)
	BATCH_FORGET    = OpCode(42)
	FALLOCATE       = OpCode(43)
	READDIRPLUS     = OpCode(44)
	RENAME2         = OpCode(45)
	LSEEK           = OpCode(46)
	COPY_FILE_RANGE = OpCode(47)
)

// CUSE specific operations
const (
	CUSE_INIT = OpCode(4096)
)

var opCodeText = [...]string{
	LOOKUP:          "LOOKUP",
	FORGET:          "FORGET",
	GETATTR:         "GETATTR",
	SETATTR:         "SETATTR",
	READLINK:        "READLINK",
	SYMLINK:         "SYMLINK",
	MKNOD:           "MKNOD",
	MKDIR:           "MKDIR",
	UNLINK:          "UNLINK",
	RMDIR:           "RMDIR",
	RENAME:          "RENAME",
	LINK:            "LINK",
	OPEN:            "OPEN",
	READ:            "READ",
	WRITE:           "WRITE",
	STATFS:          "STATFS",
	RELEASE:         "RELEASE",
	FSYNC:           "FSYNC",
	SETXATTR:        "SETXATTR",
	GETXATTR:        "GETXATTR",
	LISTXATTR:       "LISTXATTR",
	REMOVEXATTR:     "REMOVEXATTR",
	FLUSH:           "FLUSH",
	INIT:            "INIT",
	OPENDIR:         "OPENDIR",
	READDIR:         "READDIR",
	RELEASEDIR:      "RELEASEDIR",
	FSYNCDIR:        "FSYNCDIR",
	GETLK:           "GETLK",
	SETLK:           "SETLK",
	SETLKW:          "SETLKW",
	ACCESS:          "ACCESS",
	CREATE:          "CREATE",
	INTERRUPT:       "INTERRUPT",
	BMAP:            "BMAP",
	DESTROY:         "DESTROY",
	IOCTL:           "IOCTL",
	POLL:            "POLL",
	NOTIFY_REPLY:    "NOTIFY_REPLY",
	BATCH_FORGET:    "BATCH_FORGET",
	FALLOCATE:       "FALLOCATE",
	READDIRPLUS:     "READDIRPLUS",
	RENAME2:         "RENAME2",
	LSEEK:           "LSEEK",
	COPY_FILE_RANGE: "COPY_FILE_RANGE",
}

type NotifyCode int32

func (code NotifyCode) String() string {
	if int(code) < len(notifyCodeText) && notifyCodeText[code] != "" {
		return notifyCodeText[code]
	}
	return fmt.Sprintf("UNKNOWN(%d)", code)
}

const (
	_                  = iota
	NOTIFY_POLL        = 1
	NOTIFY_INVAL_INODE = 2
	NOTIFY_INVAL_ENTRY = 3
	NOTIFY_STORE       = 4
	NOTIFY_RETRIEVE    = 5
	NOTIFY_DELETE      = 6
	NOTIFY_CODE_MAX    = iota
)

var notifyCodeText = [...]string{
	NOTIFY_POLL:        "POLL",
	NOTIFY_INVAL_INODE: "INVAL_INODE",
	NOTIFY_INVAL_ENTRY: "INVAL_ENTRY",
	NOTIFY_STORE:       "STORE",
	NOTIFY_RETRIEVE:    "RETRIEVE",
	NOTIFY_DELETE:      "DELETE",
	NOTIFY_CODE_MAX:    "CODE_MAX",
}

// The read buffer is required to be at least 8k, but may be much larger
const MIN_READ_BUFFER = 8192

const COMPAT_ENTRY_OUT_SIZE = 120

type EntryOut struct {
	// Inode ID
	Nodeid uint64

	// Inode generation: Nodeid:gen must be unique for the fs's lifetime
	Generation uint64

	// Cache timeout for the name
	EntryValid uint64

	// Cache timeout for the attributes
	AttrValid uint64

	EntryValidNsec uint32
	AttrValidNsec  uint32
	Attr           Attr
}

type ForgetIn struct {
	Nlookup uint64
}

type ForgetOne struct {
	Nodeid  uint64
	Nlookup uint64
}

type BatchForgetIn struct {
	Count uint32
	_     uint32
}

type GetattrIn struct {
	GetattrFlags uint32
	_            uint32
	Fh           uint64
}

const COMPAT_ATTR_OUT_SIZE = 96

type AttrOut struct {
	// cache timeout for the attributes
	AttrValid     uint64
	AttrValidNsec uint32
	_             uint32
	Attr          Attr
}

const COMPAT_MKNOD_IN_SIZE = 8

type MknodIn struct {
	Mode  uint32
	Rdev  uint32
	Umask uint32
	_     uint32
}

type MkdirIn struct {
	Mode  uint32
	Umask uint32
}

type RenameIn struct {
	Newdir uint64
}

type Rename2In struct {
	Newdir uint64
	Flags  uint32
	_      uint32
}

type LinkIn struct {
	Oldnodeid uint64
}

type SetattrIn struct {
	Valid     uint32
	_         uint32
	Fh        uint64
	Size      uint64
	LockOwner uint64
	Atime     uint64
	Mtime     uint64
	Ctime     uint64
	Atimensec uint32
	Mtimensec uint32
	Ctimensec uint32
	Mode      uint32
	_         uint32
	Uid       uint32
	Gid       uint32
	_         uint32
}

type OpenIn struct {
	Flags uint32
	_     uint32
}

type CreateIn struct {
	Flags uint32
	Mode  uint32
	Umask uint32
	_     uint32
}

type OpenOut struct {
	Fh        uint64
	OpenFlags uint32
	_         uint32
}

type ReleaseIn struct {
	Fh           uint64
	Flags        uint32
	ReleaseFlags uint32
	LockOwner    uint64
}

type FlushIn struct {
	Fh        uint64
	_         uint32
	_         uint32
	LockOwner uint64
}

type ReadIn struct {
	Fh        uint64
	Offset    uint64
	Size      uint32
	ReadFlags uint32
	LockOwner uint64
	Flags     uint32
	_         uint32
}

const COMPAT_WRITE_IN_SIZE = 24

type WriteIn struct {
	Fh         uint64
	Offset     uint64
	Size       uint32
	WriteFlags uint32
	LockOwner  uint64
	Flags      uint32
	_          uint32
}

type WriteOut struct {
	Size uint32
	_    uint32
}

const COMPAT_STATFS_SIZE = 48

type StatFSOut struct {
	St KStatFS
}

type FsyncIn struct {
	Fh         uint64
	FsyncFlags uint32
	_          uint32
}

type SetxattrIn struct {
	Size  uint32
	Flags uint32
}

type GetxattrIn struct {
	Size uint32
	_    uint32
}

type GetxattrOut struct {
	Size uint32
	_    uint32
}

type LkIn struct {
	Fh      uint64
	Owner   uint64
	Lk      FileLock
	LkFlags uint32
	_       uint32
}

type LkOut struct {
	Lk FileLock
}

type AccessIn struct {
	Mask uint32
	_    uint32
}

type InitIn struct {
	Major        uint32
	Minor        uint32
	MaxReadahead uint32
	Flags        uint32
}

const COMPAT_INIT_OUT_SIZE = 8
const COMPAT_22_INIT_OUT_SIZE = 24

type InitOut struct {
	Major               uint32
	Minor               uint32
	MaxReadahead        uint32
	Flags               uint32
	MaxBackground       uint16
	CongestionThreshold uint16
	MaxWrite            uint32
	TimeGran            uint32
	MaxPages            uint16
	_                   uint16
	_                   [8]uint32
}

const CUSE_INIT_INFO_MAX = 4096

type CuseInitIn struct {
	Major uint32
	Minor uint32
	_     uint32
	Flags uint32
}

type CuseInitOut struct {
	Major    uint32
	Minor    uint32
	U        uint32
	Flags    uint32
	MaxRead  uint32
	MaxWrite uint32
	DevMajor uint32 // chardev major
	DevMinor uint32 // chardev minor
	_        [10]uint32
}

type InterruptIn struct {
	Unique uint64
}

type BmapIn struct {
	Block     uint64
	Blocksize uint32
	_         uint32
}

type BmapOut struct {
	Block uint64
}

type IoctlIn struct {
	Fh      uint64
	Flags   uint32
	Cmd     uint32
	Arg     uint64
	InSize  uint32
	OutSize uint32
}

type IoctlIovec struct {
	Base uint64
	Len  uint64
}

type IoctlOut struct {
	Result  int32
	Flags   uint32
	InIovs  uint32
	OutIovs uint32
}

type PollIn struct {
	Fh     uint64
	Kh     uint64
	Flags  uint32
	Events uint32
}

type PollOut struct {
	Revents uint32
	Padding uint32
}

type NotifyPollWakeupOut struct {
	Kh uint64
}

type FallocateIn struct {
	Fh     uint64
	Offset uint64
	Length uint64
	Mode   uint32
	_      uint32
}

type InHeader struct {
	Len    uint32
	OpCode OpCode
	Unique uint64
	Nodeid uint64
	Uid    uint32
	Gid    uint32
	Pid    uint32
	_      uint32
}

type OutHeader struct {
	Len    uint32
	Error  int32
	Unique uint64
}

type Dirent struct {
	Ino     uint64
	Off     uint64
	Namelen uint32
	Type    uint32

	// placeholder zero-byte value.
	Name struct{}
}

const NAME_OFFSET = uint32(unsafe.Offsetof(Dirent{}.Name))

func DirentSize(ent Dirent) uint32 {
	return NAME_OFFSET + ent.Namelen
}

func DirentAlign(namelen uint32) uint32 {
	const max = uint32(unsafe.Sizeof(uint64(0)) - 1)
	return (namelen + max) &^ max
}

type Direntplus struct {
	EntryOut EntryOut
	Dirent   Dirent
}

const NAME_OFFSET_DIRENTPLUS = uint32(unsafe.Offsetof(Direntplus{}.Dirent.Name))

func DirentplusSize(ent Direntplus) uint32 {
	return NAME_OFFSET_DIRENTPLUS + ent.Dirent.Namelen
}

type NotifyInvalInodeOut struct {
	Ino uint64
	Off int64
	Len int64
}

type NotifyInvalEntryOut struct {
	Parent  uint64
	Namelen uint32
	_       uint32
}

type NotifyDeleteOut struct {
	Parent  uint64
	Child   uint64
	Namelen uint32
	_       uint32
}

type NotifyStoreOut struct {
	Nodeid uint64
	Offset uint64
	Size   uint32
	_      uint32
}

type NotifyRetrieveOut struct {
	NotifyUnique uint64
	Nodeid       uint64
	Offset       uint64
	Size         uint32
	_            uint32
}

// Matches the size of WriteIn
type NotifyRetrieveIn struct {
	_      uint64
	Offset uint64
	Size   uint32
	_      uint32
	_      uint64
	_      uint64
}

// Device ioctls:
const (
	// _IOR(229, 0, uint32_t)
	DEV_IOC_CLONE = uint32(0x8004e500)
)

type LseekIn struct {
	Fh     uint64
	Offset uint64
	Whence uint32
	_      uint32
}

type LseekOut struct {
	Offset uint64
}

type CopyFileRangeIn struct {
	FhIn      uint64
	OffIn     uint64
	NodeidOut uint64
	FhOut     uint64
	OffOut    uint64
	Len       uint64
	Flags     uint64
}
