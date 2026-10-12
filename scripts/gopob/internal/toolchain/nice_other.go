//go:build !linux

package toolchain

import "os"

func applyPriority() int { _ = os.Setenv("GOPOB_NICED", "1"); return -1 }

func currentNiceness() (int, error) { return 0, nil }

func SetprioritySelf(int) {}

func IoprioIdleSelf() {}
