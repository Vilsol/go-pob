package main

import (
	"bytes"
	"os/exec"
	"strings"
)

func awGit(dir string, args ...string) (string, bool, string) {
	cmd := exec.Command("git", args...) //nolint:gosec,noctx // as in the original
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	return strings.TrimRight(out.String(), "\n"), err == nil, errb.String()
}

func awExitStatus(err error) int {
	var ee *exec.ExitError
	if errorsAs(err, &ee) {
		return ee.ExitCode()
	}
	return 128
}
