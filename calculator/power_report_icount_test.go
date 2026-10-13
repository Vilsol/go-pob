//go:build linux

package calculator

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestICountPowerReport counts the user-space instructions of one power report of POWER_ICOUNT_STAT.
func TestICountPowerReport(t *testing.T) {
	names := strings.Fields(os.Getenv("POWER_ICOUNT_BUILDS"))
	if len(names) == 0 {
		t.SkipNow()
	}
	stat := os.Getenv("POWER_ICOUNT_STAT")
	if stat == "" {
		stat = "FullDPS"
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	fds := openInstructionCounters(t)
	for _, name := range names {
		build := parseCorpusBuild(t, name+".xml")
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start := readInstructionCounters(t, fds)
		pb := GetMiscCalculator(build).NewPowerBuilder(powerOracleStat(stat), 0)
		for pb.Progress() < 1 {
			pb.Step(time.Second)
		}
		instr := readInstructionCounters(t, fds) - start
		runtime.ReadMemStats(&after)
		if instr == 0 || pb.calcs == 0 {
			t.Fatalf("REFUSING: %s counted %d instructions over %d what-ifs", name, instr, pb.calcs)
		}
		n := float64(pb.calcs)
		t.Logf("ICOUNT %s %s whatifs %d instr/whatif %.0f allocs/whatif %.0f bytes/whatif %.0f gcs %d", name[:strings.IndexByte(name, '-')],
			stat, pb.calcs, float64(instr)/n, float64(after.Mallocs-before.Mallocs)/n, float64(after.TotalAlloc-before.TotalAlloc)/n, after.NumGC-before.NumGC)
	}
}
