//go:build linux

package asklog_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/protolambda/asklog"
)

// openTerminal opens a new pseudo-terminal and returns its terminal side.
// It skips the test if the system provides no pseudo-terminals.
func openTerminal(t *testing.T) *os.File {
	t.Helper()
	ptmx, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal: %v", err)
	}
	t.Cleanup(func() { _ = ptmx.Close() })
	var n uint32
	control(t, ptmx, func(fd uintptr) error {
		var unlock int32
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); errno != 0 {
			return fmt.Errorf("unlock pseudo-terminal: %w", errno)
		}
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); errno != 0 {
			return fmt.Errorf("get pseudo-terminal number: %w", errno)
		}
		return nil
	})
	pts, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal: %v", err)
	}
	t.Cleanup(func() { _ = pts.Close() })
	return pts
}

// control calls fn with the file descriptor of f, without changing the mode of f as f.Fd would.
func control(t *testing.T, f *os.File, fn func(fd uintptr) error) {
	t.Helper()
	rc, err := f.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var fnErr error
	if err := rc.Control(func(fd uintptr) { fnErr = fn(fd) }); err != nil {
		t.Fatal(err)
	}
	if fnErr != nil {
		t.Fatal(fnErr)
	}
}

// nonblocking reports whether the file descriptor of f is in non-blocking mode,
// which the deadlines of f need.
func nonblocking(t *testing.T, f *os.File) bool {
	t.Helper()
	var flags uintptr
	control(t, f, func(fd uintptr) error {
		r, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
		if errno != 0 {
			return fmt.Errorf("get file status flags: %w", errno)
		}
		flags = r
		return nil
	})
	return flags&syscall.O_NONBLOCK != 0
}

func TestConfigDefaultColorTerminal(t *testing.T) {
	pts := openTerminal(t)
	if !nonblocking(t, pts) {
		t.Fatal("expected a pseudo-terminal file in non-blocking mode")
	}
	cfg := asklog.Config{Out: pts}
	cfg.Default()
	if !cfg.Color {
		t.Fatal("expected color: Out is a terminal")
	}
	// Default must not change the file: os.File.Fd would put it in blocking mode,
	// and then its deadlines no longer interrupt a blocked write.
	if !nonblocking(t, pts) {
		t.Fatal("Default put Out in blocking mode")
	}
	// Nothing reads the other side of the pseudo-terminal, so the write blocks until the deadline.
	if err := pts.SetWriteDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
		t.Fatalf("set write deadline: %v", err)
	}
	if _, err := pts.Write(make([]byte, 1<<20)); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("expected the write deadline to end the write, got %v", err)
	}
}

func TestConfigDefaultColorWrappedTerminal(t *testing.T) {
	pts := openTerminal(t)
	// Only Out itself is checked: a writer that wraps a terminal has no file descriptor.
	cfg := asklog.Config{Out: struct{ io.Writer }{pts}}
	cfg.Default()
	if cfg.Color {
		t.Fatal("expected no color: Out wraps a terminal but is not one")
	}
}
