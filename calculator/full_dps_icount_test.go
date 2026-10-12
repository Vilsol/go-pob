//go:build linux

package calculator

import (
	"encoding/binary"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func openInstructionCounters(t *testing.T) []int {
	t.Helper()
	var pmus []uint64
	for _, dev := range []string{"cpu_core", "cpu_atom"} {
		raw, err := os.ReadFile("/sys/bus/event_source/devices/" + dev + "/type")
		if err != nil {
			continue
		}
		if v, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 32); err == nil {
			pmus = append(pmus, v)
		}
	}
	if len(pmus) == 0 {
		pmus = []uint64{0}
	}
	fds := make([]int, 0, len(pmus))
	for _, pmu := range pmus {
		var attr [64]byte
		binary.LittleEndian.PutUint32(attr[0:], 0)
		binary.LittleEndian.PutUint32(attr[4:], 64)
		binary.LittleEndian.PutUint64(attr[8:], pmu<<32|1)
		binary.LittleEndian.PutUint64(attr[40:], 1<<5|1<<6)
		fd, _, errno := syscall.Syscall6(syscall.SYS_PERF_EVENT_OPEN, uintptr(unsafe.Pointer(&attr[0])), 0, ^uintptr(0), ^uintptr(0), 0, 0)
		if errno != 0 {
			t.Fatalf("REFUSING: perf_event_open(pmu %d): %v", pmu, errno)
		}
		fds = append(fds, int(fd))
	}
	return fds
}

func readInstructionCounters(t *testing.T, fds []int) uint64 {
	t.Helper()
	var total uint64
	for _, fd := range fds {
		var buf [8]byte
		if _, err := syscall.Read(fd, buf[:]); err != nil {
			t.Fatal(err)
		}
		total += binary.LittleEndian.Uint64(buf[:])
	}
	return total
}

// TestICountBuildOutput counts the user-space instructions of ICOUNT_N main calculations of each build in ICOUNT_BUILDS on one locked thread.
func TestICountBuildOutput(t *testing.T) {
	names := strings.Fields(os.Getenv("ICOUNT_BUILDS"))
	if len(names) == 0 {
		t.SkipNow()
	}
	n, err := strconv.ParseUint(os.Getenv("ICOUNT_N"), 10, 32)
	if err != nil || n < 1 {
		t.Fatal("REFUSING: ICOUNT_N must be a positive count")
	}
	initBenchData(t)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	fds := openInstructionCounters(t)
	for _, name := range names {
		build := loadHeavyBuild(t, name)
		c := &Calculator{PoB: build}
		c.BuildOutput(OutputModeMain)
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start := readInstructionCounters(t, fds)
		for range n {
			c.BuildOutput(OutputModeMain)
		}
		instr := readInstructionCounters(t, fds) - start
		runtime.ReadMemStats(&after)
		if instr == 0 {
			t.Fatalf("REFUSING: %s counted 0 instructions", name)
		}
		t.Logf("ICOUNT %s instr/calc %d allocs/calc %d bytes/calc %d gc/100calc %d", name[:strings.IndexByte(name, '-')],
			instr/n, (after.Mallocs-before.Mallocs)/n, (after.TotalAlloc-before.TotalAlloc)/n,
			uint64(after.NumGC-before.NumGC)*100/n)
	}
}
