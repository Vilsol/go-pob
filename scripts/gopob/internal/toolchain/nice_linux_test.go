//go:build linux

package toolchain

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// TestNiceReachesEveryThread: NICENESS IS PER-THREAD ON LINUX, SO "THE PROCESS IS NICED" MUST BE ASSERTED OF EVERY THREAD.
func TestNiceReachesEveryThread(t *testing.T) {
	if os.Getenv("GOPOB_EVERY_THREAD_CHILD") == "" {
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), exe, "-test.run=^TestNiceReachesEveryThread$", "-test.v", "-test.count=1") //nolint:gosec // re-running this test binary
		cmd.Env = append(os.Environ(), "GOPOB_EVERY_THREAD_CHILD=1")
		out, err := cmd.CombinedOutput()
		if !strings.Contains(string(out), "=== RUN   TestNiceReachesEveryThread") {
			t.Fatalf("the child ran nothing (%v):\n%s", err, out)
		}
		if err != nil {
			t.Fatalf("child failed: %v\n%s", err, out)
		}
		return
	}
	stop := make(chan struct{})
	defer close(stop)
	for range 4 {
		go func() {
			runtime.LockOSThread()
			<-stop
		}()
	}
	time.Sleep(50 * time.Millisecond)

	before, err := Niceness()
	if err != nil {
		t.Skipf("niceness unreadable on this platform: %v", err)
	}
	want := min(before+3, 19)
	t.Setenv("GOPOB_NICED", "")
	t.Setenv("NICE", strconv.Itoa(want))
	Nice()

	ents, err := os.ReadDir("/proc/self/task")
	if err != nil {
		t.Skipf("no /proc/self/task: %v", err)
	}
	if len(ents) < 2 {
		t.Fatalf("only %d thread(s) — the fixture has nothing to miss", len(ents))
	}
	for _, e := range ents {
		tid, _ := strconv.Atoi(e.Name())
		raw, err := unix.Getpriority(unix.PRIO_PROCESS, tid)
		if err != nil {
			continue
		}
		if got := 20 - raw; got != want {
			t.Errorf("thread %d is at niceness %d, want %d (main thread started at %d)", tid, got, want, before)
		}
	}
}
