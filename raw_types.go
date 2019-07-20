package fuse

// from fuse_kernel.h
//
// kept as close as possible to original definitions, in hopes
// for future code generation

type fuse_in_header struct {
	len    uint32
	opcode opcode
	unique uint64
	nodeid uint64
	uid    uint32
	gid    uint32
	pid    uint32
	_      uint32
}

type fuse_out_header struct {
	len    uint32
	error  uint32
	unique uint64
}

/**
 * INIT request/reply flags
 *
 * FUSE_ASYNC_READ: asynchronous read requests
 * FUSE_POSIX_LOCKS: remote locking for POSIX file locks
 * FUSE_FILE_OPS: kernel sends file handle for fstat, etc... (not yet supported)
 * FUSE_ATOMIC_O_TRUNC: handles the O_TRUNC open flag in the filesystem
 * FUSE_EXPORT_SUPPORT: filesystem handles lookups of "." and ".."
 * FUSE_BIG_WRITES: filesystem can handle write size larger than 4kB
 * FUSE_DONT_MASK: don't apply umask to file mode on create operations
 * FUSE_SPLICE_WRITE: kernel supports splice write on the device
 * FUSE_SPLICE_MOVE: kernel supports splice move on the device
 * FUSE_SPLICE_READ: kernel supports splice read on the device
 * FUSE_FLOCK_LOCKS: remote locking for BSD style file locks
 * FUSE_HAS_IOCTL_DIR: kernel supports ioctl on directories
 * FUSE_AUTO_INVAL_DATA: automatically invalidate cached pages
 * FUSE_DO_READDIRPLUS: do READDIRPLUS (READDIR+LOOKUP in one)
 * FUSE_READDIRPLUS_AUTO: adaptive readdirplus
 * FUSE_ASYNC_DIO: asynchronous direct I/O submission
 * FUSE_WRITEBACK_CACHE: use writeback cache for buffered writes
 * FUSE_NO_OPEN_SUPPORT: kernel supports zero-message opens
 * FUSE_PARALLEL_DIROPS: allow parallel lookups and readdir
 * FUSE_HANDLE_KILLPRIV: fs handles killing suid/sgid/cap on write/chown/trunc
 * FUSE_POSIX_ACL: filesystem supports posix acls
 * FUSE_ABORT_ERROR: reading the device after abort returns ECONNABORTED
 * FUSE_MAX_PAGES: init_out.max_pages contains the max number of req pages
 * FUSE_CACHE_SYMLINKS: cache READLINK responses
 * FUSE_NO_OPENDIR_SUPPORT: kernel supports zero-message opendir
 */
const (
	FUSE_ASYNC_READ         = (1 << 0)
	FUSE_POSIX_LOCKS        = (1 << 1)
	FUSE_FILE_OPS           = (1 << 2)
	FUSE_ATOMIC_O_TRUNC     = (1 << 3)
	FUSE_EXPORT_SUPPORT     = (1 << 4)
	FUSE_BIG_WRITES         = (1 << 5)
	FUSE_DONT_MASK          = (1 << 6)
	FUSE_SPLICE_WRITE       = (1 << 7)
	FUSE_SPLICE_MOVE        = (1 << 8)
	FUSE_SPLICE_READ        = (1 << 9)
	FUSE_FLOCK_LOCKS        = (1 << 10)
	FUSE_HAS_IOCTL_DIR      = (1 << 11)
	FUSE_AUTO_INVAL_DATA    = (1 << 12)
	FUSE_DO_READDIRPLUS     = (1 << 13)
	FUSE_READDIRPLUS_AUTO   = (1 << 14)
	FUSE_ASYNC_DIO          = (1 << 15)
	FUSE_WRITEBACK_CACHE    = (1 << 16)
	FUSE_NO_OPEN_SUPPORT    = (1 << 17)
	FUSE_PARALLEL_DIROPS    = (1 << 18)
	FUSE_HANDLE_KILLPRIV    = (1 << 19)
	FUSE_POSIX_ACL          = (1 << 20)
	FUSE_ABORT_ERROR        = (1 << 21)
	FUSE_MAX_PAGES          = (1 << 22)
	FUSE_CACHE_SYMLINKS     = (1 << 23)
	FUSE_NO_OPENDIR_SUPPORT = (1 << 24)
)

type fuse_init_in struct {
	header        fuse_in_header
	major         uint32
	minor         uint32
	max_readahead uint32
	flags         uint32
}

type fuse_init_out struct {
	header               fuse_out_header
	major                uint32
	minor                uint32
	max_readahead        uint32
	flags                uint32
	max_background       uint16
	congestion_threshold uint16
	max_write            uint32
	time_gran            uint32
	max_pages            uint16
	_                    uint16
	unused               [8]uint32
}

// opcodes
type opcode uint32

const (
	FUSE_LOOKUP          = opcode(1)
	FUSE_FORGET          = opcode(2)
	FUSE_GETATTR         = opcode(3)
	FUSE_SETATTR         = opcode(4)
	FUSE_READLINK        = opcode(5)
	FUSE_SYMLINK         = opcode(6)
	FUSE_MKNOD           = opcode(8)
	FUSE_MKDIR           = opcode(9)
	FUSE_UNLINK          = opcode(10)
	FUSE_RMDIR           = opcode(11)
	FUSE_RENAME          = opcode(12)
	FUSE_LINK            = opcode(13)
	FUSE_OPEN            = opcode(14)
	FUSE_READ            = opcode(15)
	FUSE_WRITE           = opcode(16)
	FUSE_STATFS          = opcode(17)
	FUSE_RELEASE         = opcode(18)
	FUSE_FSYNC           = opcode(20)
	FUSE_SETXATTR        = opcode(21)
	FUSE_GETXATTR        = opcode(22)
	FUSE_LISTXATTR       = opcode(23)
	FUSE_REMOVEXATTR     = opcode(24)
	FUSE_FLUSH           = opcode(25)
	FUSE_INIT            = opcode(26)
	FUSE_OPENDIR         = opcode(27)
	FUSE_READDIR         = opcode(28)
	FUSE_RELEASEDIR      = opcode(29)
	FUSE_FSYNCDIR        = opcode(30)
	FUSE_GETLK           = opcode(31)
	FUSE_SETLK           = opcode(32)
	FUSE_SETLKW          = opcode(33)
	FUSE_ACCESS          = opcode(34)
	FUSE_CREATE          = opcode(35)
	FUSE_INTERRUPT       = opcode(36)
	FUSE_BMAP            = opcode(37)
	FUSE_DESTROY         = opcode(38)
	FUSE_IOCTL           = opcode(39)
	FUSE_POLL            = opcode(40)
	FUSE_NOTIFY_REPLY    = opcode(41)
	FUSE_BATCH_FORGET    = opcode(42)
	FUSE_FALLOCATE       = opcode(43)
	FUSE_READDIRPLUS     = opcode(44)
	FUSE_RENAME2         = opcode(45)
	FUSE_LSEEK           = opcode(46)
	FUSE_COPY_FILE_RANGE = opcode(47)

	/* CUSE specific operations */
	CUSE_INIT = 4096
)
