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

func mount(target string) (net.Conn, error) {
	return usermount(target, Options{})
}

func umount(target string) error {
	return userumount(target, false)
}

func usermount(target string, options Options) (conn net.Conn, err error) {
	pair, err := unixPair(unix.SOCK_STREAM)
	if err != nil {
		return nil, err
	}
	defer closeErr(pair[0], &err)
	defer closeErr(pair[1], &err)

	cmd := exec.Command("fusermount", target)
	cmd.Env = []string{"_FUSE_COMMFD=3"}
	cmd.ExtraFiles = pair[1:]

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return nil, errors.New(stderr.String())
	}

	return connect(pair[0])
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

func connect(f *os.File) (conn net.Conn, err error) {
	fuseConn, err := net.FileConn(f)
	if err != nil {
		return nil, err
	}
	defer closeErr(fuseConn, &err)

	fd, err := receiveFD(fuseConn.(*net.UnixConn))
	return net.FileConn(os.NewFile(uintptr(fd), ""))
}

func receiveFD(conn *net.UnixConn) (fd int, err error) {
	oob := make([]byte, unix.CmsgSpace(unix.SizeofInt)/unix.SizeofPtr)
	_, n, _, _, err := conn.ReadMsgUnix(nil, oob)
	if err != nil {
		return 0, err
	}
	if n < len(oob) {
		return 0, errors.New("short socket control message")
	}

	messages, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return 0, err
	}
	if len(messages) == 0 {
		return 0, errors.New("no socket control message")
	}

	fds, err := unix.ParseUnixRights(&messages[0])
	if err != nil {
		return 0, err
	}

	if len(fds) == 0 || fds[0] < 0 {
		return 0, errors.New("received bad fd")
	}

	unix.CloseOnExec(fds[0])
	return fds[0], nil
}

func unixPair(typ int) (pair [2]*os.File, err error) {
	fds, err := unix.Socketpair(unix.AF_LOCAL, typ|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return
	}

	pair[0] = os.NewFile(uintptr(fds[0]), "")
	pair[1] = os.NewFile(uintptr(fds[1]), "")
	return
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
		_ = closer.Close()
	}
}
