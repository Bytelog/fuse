package fuse

import (
	"bytes"
	"errors"
	"net"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

func mount(target string) (*os.File, error) {
	return usermount(target, Options{})
}

func umount(target string) error {
	return userumount(target, false)
}

func usermount(target string, options Options) (dev *os.File, err error) {
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

	conn, err := net.FileConn(pair[0])
	if err != nil {
		return nil, err
	}
	defer closeErr(conn, &err)
	return receiveDev(conn.(*net.UnixConn))
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

func receiveDev(conn *net.UnixConn) (dev *os.File, err error) {
	oob := make([]byte, unix.CmsgSpace(unix.SizeofInt))
	_, n, _, _, err := conn.ReadMsgUnix(nil, oob)
	if err != nil {
		return nil, err
	}
	if n < len(oob) {
		return nil, errors.New("short socket control message")
	}

	messages, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, err
	}
	if len(messages) != 1 {
		return nil, errors.New("bad socket control message")
	}

	fds, err := unix.ParseUnixRights(&messages[0])
	if err != nil {
		return nil, err
	}

	if len(fds) != 1 || fds[0] < 0 {
		return nil, errors.New("received bad fd")
	}

	if err := unix.SetNonblock(fds[0], true); err != nil {
		return nil, err
	}
	unix.CloseOnExec(fds[0])
	return os.NewFile(uintptr(fds[0]), "/dev/fuse"), nil
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
