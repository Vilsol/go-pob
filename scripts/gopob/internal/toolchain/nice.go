package toolchain

import "errors"

var ErrNicenessUnsupported = errors.New("process niceness is only supported on linux")

func Nice() int { return applyPriority() }

func Niceness() (int, error) { return currentNiceness() }
