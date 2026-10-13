//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
)

var errNoGetpriority = errors.New("getpriority does not exist on windows")

func executableByMe(path string) bool {
	fi, err := os.Stat(path) //nolint:gosec // an operator-supplied binary path, as on unix
	return err == nil && fi.Mode().IsRegular()
}

func pidSignalZero(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = p.Release()
	return true
}

func rawGetpriority() (int, error) { return 0, errNoGetpriority } //nolint:wrapcheck // a sentinel

func replaceProcess(path string, argv []string) (int, error) {
	cmd := exec.Command(path, argv[1:]...) //nolint:gosec,noctx // running the operator's command IS this tool; it is a prefix, as in the original
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return 0, err //nolint:wrapcheck // errnoText renders it as the unix half does
}
