package toolchain

func Nice() int { return applyPriority() }

func Niceness() (int, error) { return currentNiceness() }
