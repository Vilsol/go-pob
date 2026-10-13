package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/toolchain"
)

func nrExec(argv []string) int {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "scripts/nice.sh: %s: command not found\n", argv[0])
		return 127
	}
	code, err := replaceProcess(path, argv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scripts/nice.sh: %s: %s\n", argv[0], errnoText(err))
		return 126
	}
	return code
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
		if cur, err := rawGetpriority(); err == nil {
			toolchain.SetprioritySelf(nrTargetNiceness(cur, inc))
		}
	}

	if _, err := exec.LookPath("ionice"); err == nil {
		toolchain.IoprioIdleSelf()
	}

	return nrExec(argv)
}

func nrTargetNiceness(rawGetpriority, inc int) int { return (20 - rawGetpriority) + inc }
