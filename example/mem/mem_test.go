package mem

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"testing"
	"time"

	"bytelog.org/fuse"
)

type loggy struct {
	prefix string
}

func (l loggy) Printf(format string, args ...interface{}) {
	_, _ = fmt.Printf(l.prefix+format+"\n", args...)
}

func setup(fs fuse.Filesystem, target string) (func() <-chan error, error) {
	srv := &fuse.Server{
		Options: fuse.Options{
			ErrorLog: loggy{prefix: "== error: ",},
			DebugLog: loggy{prefix: " - debug: ",},
		},
	}
	if err := srv.Serve(fs, target); err != nil {
		return nil, err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	done := make(chan struct{}, 1)
	errc := make(chan error, 1)

	go func() {
		ctx := context.Background()
		var cancel func()
		select {
		case <-sig:
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		case <-done:
			ctx, cancel = context.WithTimeout(ctx, time.Second)
			defer cancel()
		}
		errc <- srv.Shutdown(ctx)
	}()

	shutdown := func() <-chan error {
		close(done)
		return errc
	}
	return shutdown, nil
}

func TestBasic(t *testing.T) {
	fs := New()

	shutdown, err := setup(fs, "/tmp/mnt")
	assert(t, err)
	defer func() { _ = <-shutdown() }()

	f, err := os.Open("/tmp/mnt/")
	assert(t, err)

	names, err := f.Readdirnames(0)
	assert(t, err)

	fmt.Println("READDIR NAMES: ", names)

	t.Fail()
}

func assert(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%v", err)
	}
}
