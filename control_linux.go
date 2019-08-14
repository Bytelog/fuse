package fuse

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"bytelog.org/fuse/proto"
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

func clone(dev *os.File) (clone *os.File, err error) {
	cloneFD, err := unix.Open("/dev/fuse", unix.O_RDWR|unix.O_CLOEXEC, 0755)
	if err != nil {
		return nil, err
	}
	defer closeOnErr(clone, &err)
	rawConn, err := dev.SyscallConn()
	if err != nil {
		return nil, err
	}
	var rawErr error
	err = rawConn.Control(func(fd uintptr) {
		req := uint(proto.DEV_IOC_CLONE)
		rawErr = unix.IoctlSetPointerInt(cloneFD, req, int(fd))
	})
	if err = firstErr(err, rawErr); err != nil {
		return nil, err
	}
	if err := unix.SetNonblock(cloneFD, true); err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(cloneFD), "/dev/fuse"), nil
}

func deviceNumber(target string) (int, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return 0, nil
	}
	defer f.Close()

	// https://www.kernel.org/doc/Documentation/filesystems/proc.txt
	const expr = `^[^:]+:(\d+) \S+ %s .+ - fuse /dev/fuse`
	re := regexp.MustCompile(fmt.Sprintf(expr, regexp.QuoteMeta(target)))

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if matches := re.FindStringSubmatch(scanner.Text()); len(matches) > 1 {
			return strconv.Atoi(matches[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("%s not found in mountinfo", target)
}

func fusectl_abort(device int) error {
	path := fmt.Sprintf("/sys/fs/fuse/connections/%d/abort", device)
	f, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, "1")
	return err
}

func fusectl_waiting(device int) (int, error) {
	path := fmt.Sprintf("/sys/fs/fuse/connections/%d/waiting", device)
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	buf := make([]byte, 32)
	n, err := f.Read(buf)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(buf[:n])))
}

func firstErr(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
