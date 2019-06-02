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

// WIP
func mount(target string, opts string) (err error) {
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
	// todo: args
	var stderr bytes.Buffer
	cmd := exec.Command("fusermount", target)
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
