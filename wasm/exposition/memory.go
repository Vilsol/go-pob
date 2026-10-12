package exposition

import "runtime"

func LiveHeapMB() float64 {
	runtime.GC()

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return float64(stats.HeapAlloc) / (1 << 20)
}
