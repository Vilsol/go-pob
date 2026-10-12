package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/toolchain"
)

func nrExec(argv []string) int {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "scripts/nice.sh: %s: command not found\n", argv[0])
		return 127
	}
	if err := syscall.Exec(path, argv, os.Environ()); err != nil { //nolint:gosec // running the operator's command IS this tool; it is a prefix, as in the original
		fmt.Fprintf(os.Stderr, "scripts/nice.sh: %s: %s\n", argv[0], errnoText(err))
		return 126
	}
	return 0
}

func niceRun(argv []string) int {
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "usage: scripts/nice.sh <command> [args...]")
		return 64
	}

	if _, err := exec.LookPath(argv[0]); err != nil {
		fmt.Fprintf(os.Stderr, "scripts/nice.sh: %s: command not found\n", argv[0])
		return 127
	}

	niceness := "10"
	if v := os.Getenv("NICE"); v != "" {
		niceness = v
	}

	if _, err := exec.LookPath("nice"); os.Getenv("GOPOB_NICED") != "" || niceness == "0" || err != nil {
		return nrExec(argv)
	}

	_ = os.Setenv("GOPOB_NICED", "1")

	if inc, err := strconv.Atoi(niceness); err == nil {
		if cur, err := unix.Getpriority(unix.PRIO_PROCESS, 0); err == nil {
			toolchain.SetprioritySelf(nrTargetNiceness(cur, inc))
		}
	}

	if _, err := exec.LookPath("ionice"); err == nil {
		toolchain.IoprioIdleSelf()
	}

	return nrExec(argv)
}

func nrTargetNiceness(rawGetpriority, inc int) int { return (20 - rawGetpriority) + inc }
