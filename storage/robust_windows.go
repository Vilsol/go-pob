//go:build windows

package storage

import (
	"errors"
	"os"
	"syscall"
	"time"
)

// Windows refuses to open a file another process is renaming over, and to rename over a file another process has
// open. Both clear within milliseconds, so the cache retries them as cmd/go/internal/robustio does for Go's own cache.
const robustTimeout = 2 * time.Second

func isTransient(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	// ERROR_ACCESS_DENIED, ERROR_SHARING_VIOLATION, ERROR_LOCK_VIOLATION
	return errno == syscall.ERROR_ACCESS_DENIED || errno == 32 || errno == 33
}

func retry(f func() error) error {
	start := time.Now()
	for delay := time.Millisecond; ; delay *= 2 {
		err := f()
		if err == nil || !isTransient(err) || time.Since(start) > robustTimeout {
			return err
		}
		time.Sleep(delay)
	}
}

func readFileRobust(path string) ([]byte, error) {
	var b []byte
	err := retry(func() error {
		var err error
		b, err = os.ReadFile(path)
		return err //nolint:wrapcheck // os.ReadFile's error, passed on
	})
	return b, err
}

func renameRobust(from, to string) error {
	return retry(func() error { return os.Rename(from, to) }) //nolint:wrapcheck // os.Rename's error, passed on
}
