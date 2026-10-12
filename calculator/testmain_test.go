package calculator

import "os"

func registerVersionInTestCacheKey() {
	_ = os.Getenv("POE_VERSION")
}
