//go:build linux

package toolchain

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

const (
	ioprioWhoProcess = 1
	ioprioClassBE    = 2
	ioprioLevelIdle  = 7
)

func applyPriority() int {
	if os.Getenv("GOPOB_NICED") != "" {
		return -1
	}
	n := 10
	if v := os.Getenv("NICE"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return -1
		}
		if parsed == 0 {
			_ = os.Setenv("GOPOB_NICED", "1")
			return -1
		}
		n = parsed
	}
	SetprioritySelf(n)
	IoprioIdleSelf()

	_ = os.Setenv("GOPOB_NICED", "1")

	if actual, err := currentNiceness(); err == nil {
		return actual
	}
	return n
}

func forEveryThread(fn func(tid int)) {
	done := map[int]bool{}
	for {
		ents, err := os.ReadDir("/proc/self/task")
		if err != nil {
			fn(0)
			return
		}
		fresh := false
		for _, e := range ents {
			tid, err := strconv.Atoi(e.Name())
			if err != nil || done[tid] {
				continue
			}
			done[tid] = true
			fresh = true
			fn(tid)
		}
		if !fresh {
			return
		}
	}
}

func SetprioritySelf(n int) {
	forEveryThread(func(tid int) { _ = unix.Setpriority(unix.PRIO_PROCESS, tid, n) })
}

func IoprioIdleSelf() {
	forEveryThread(func(tid int) {
		_, _, _ = unix.Syscall(unix.SYS_IOPRIO_SET, ioprioWhoProcess, uintptr(tid),
			uintptr(ioprioClassBE<<13|ioprioLevelIdle))
	})
}

func currentNiceness() (int, error) {
	n, err := unix.Getpriority(unix.PRIO_PROCESS, 0)
	if err != nil {
		return 0, fmt.Errorf("cannot read this process's niceness: %w", err)
	}
	return 20 - n, nil
}
