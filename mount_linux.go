package fuse

import (
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

func mount(target string) error {

}

func umount(target string) error {

}

func usermount(target string, options Options) (conn *net.UnixConn, err error) {
	conn, fd, err := unixPair(unix.SOCK_STREAM)
	if err != nil {
		return nil, err
	}
	defer closeErr(fd, &err)
	defer closeOnErr(conn, &err)

	cmd := exec.Command("fusermount", target)
	cmd.Env = []string{"_FUSE_COMMFD=3"}
	cmd.ExtraFiles = []*os.File{fd}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		err = errors.New(stderr.String())
	}

	return conn, err
}

func userumount(target string, lazy bool) error {
	cmd := exec.Command("fusermount", target, "-u")

	if lazy {
		cmd.Args = append(cmd.Args, "-z")
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.New(stderr.String())
	}
	return nil
}

func unixPair(typ int) (*net.UnixConn, *os.File, error) {
	fds, err := unix.Socketpair(unix.AF_LOCAL, typ|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}

	cfd := os.NewFile(uintptr(fds[0]), "")
	defer closeOnErr(cfd, &err)

	pfd := os.NewFile(uintptr(fds[1]), "")
	defer closeOnErr(pfd, &err)

	conn, err := net.FileConn(cfd)
	if err != nil {
		return nil, nil, err
	}
	return conn.(*net.UnixConn), pfd, nil
}

func closeErr(closer io.Closer, err *error) {
	if err == nil {
		panic("nil error")
	}
	if cerr := closer.Close(); *err == nil {
		*err = cerr
	}
}

func closeOnErr(closer io.Closer, err *error) {
	if err == nil {
		panic("nil error")
	}
	if *err != nil {
		*err = closer.Close()
	}
}
