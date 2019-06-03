package fuse

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

// from: http://man7.org/linux/man-pages/man8/mount.fuse.8.html
// we may want to not support all of these. Just listing them for now.
type Options struct {
	// our options
	Debug bool

	// mount options
	DefaultPermissions bool
	AllowOther bool
	RootMode uint32
	BlockDevice bool
	BlockSize int
	MaxRead int
	FD int
	UID int
	GID int
	FSName string
	SubType string

	// libfuse options
	AllowRoot bool
	AutoUnmount bool // can we make this default behavior? It's convenient.
}

func (o Options) String() string {
	return ""
}

// WIP
func mount(target string, options Options) (err error) {
	// create the mount directory
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if err := os.Mkdir(target, 0777); err != nil {
			return err
		}
	}

	// attempt to clean up existing mounts
	_ = exec.Command("fusermount", "-u", target).Run()

	// todo: is there a possible performance boost by using the unordered
	// unixgram variant (SOCK_DGRAM)?
	conn, fd, err := unixPair(unix.SOCK_STREAM)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = conn.Close()
			_ = fd.Close()
		}
	}()

	// register the mount
	var stderr bytes.Buffer
	cmd := exec.Command("fusermount", target, options.String())
	cmd.Stderr = &stderr
	cmd.Env = []string{"_FUSE_COMMFD=3"}
	cmd.ExtraFiles = []*os.File{fd}
	if err = cmd.Run(); err != nil {
		return errors.New(stderr.String())
	}

	var data [4]byte
	control := make([]byte, 1024)
	_, oobn, flags, _, err := conn.ReadMsgUnix(data[:], control)
	// todo: retry?
	if err != nil {
		return err
	}

	// todo: more bad situation checks/handling.
	fmt.Println("flags: ", flags)

	if oobn <= unix.SizeofCmsghdr {
		return fmt.Errorf("short control")
	}

	return nil
}

func umount(target string) error {
	var stderr bytes.Buffer
	cmd := exec.Command("fusermount", "-u", target)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// todo: decide on error format
		return errors.New(stderr.String())
	}
	return nil
}

func unixPair(typ int) (*net.UnixConn, *os.File, error) {
	fds, err := unix.Socketpair(unix.AF_LOCAL, typ | unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	cfd, pfd := os.NewFile(uintptr(fds[0]), ""), os.NewFile(uintptr(fds[1]), "")
	conn, err := net.FileConn(cfd)
	if err != nil {
		return nil, nil, err
	}
	_ = cfd.Close()
	return conn.(*net.UnixConn), pfd, nil
}
