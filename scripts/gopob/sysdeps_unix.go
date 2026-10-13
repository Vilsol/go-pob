//go:build unix

package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func executableByMe(path string) bool { return unix.Access(path, unix.X_OK) == nil }

func pidSignalZero(pid int) bool { return syscall.Kill(pid, 0) == nil }

func rawGetpriority() (int, error) { return unix.Getpriority(unix.PRIO_PROCESS, 0) } //nolint:wrapcheck // the caller only tests it

func replaceProcess(path string, argv []string) (int, error) {
	return 0, syscall.Exec(path, argv, os.Environ()) //nolint:gosec,wrapcheck // errnoText renders the bare errno; running the operator's command IS this tool; it is a prefix, as in the original
}
