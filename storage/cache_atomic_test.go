package storage

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	cacheProbeChildEnv = "GO_POB_CACHE_ATOMIC_CHILD"
	cacheProbeKey      = "https://example.invalid/go-pob/atomic-cache-probe"
	cacheProbeLen      = 8 << 20
	cacheProbeChildren = 40
)

func cacheProbeValue() []byte {
	return bytes.Repeat([]byte{0xA5}, cacheProbeLen)
}

func useTempCacheDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	t.Cleanup(func() {
		InitializeStorage(nil, nil, nil, nil, nil, nil)
	})

	t.Setenv(CacheDirEnv, dir)
	InitializeStorage(nil, nil, nil, nil, nil, nil)

	return dir
}

// TestSetCacheCanOverwriteTheSameKey pins that writing a key twice in one process completes.
func TestSetCacheCanOverwriteTheSameKey(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("no filesystem cache under js/wasm")
	}

	useTempCacheDir(t)

	s := Get()
	value := []byte("some cached asset")

	done := make(chan error, 1)
	go func() {
		for i := 0; i < 3; i++ {
			if err := s.SetCache(cacheProbeKey, value); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("failed writing cache: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SetCache blocked overwriting a key it had already written")
	}

	got, err := s.GetCache(cacheProbeKey)
	if err != nil {
		t.Fatalf("failed reading cache: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("read back %q, want %q", got, value)
	}
}

// TestCacheProbeChild is the reader half of TestSetCacheIsAtomicAcrossProcesses.
func TestCacheProbeChild(t *testing.T) {
	if os.Getenv(cacheProbeChildEnv) == "" {
		t.Skip("reader half of TestSetCacheIsAtomicAcrossProcesses")
	}

	watchdog := time.AfterFunc(30*time.Second, func() {
		fmt.Println("PROBE hang")
		os.Exit(3)
	})
	defer watchdog.Stop()

	s := Get()
	if !s.ExistsInCache(cacheProbeKey) {
		fmt.Println("PROBE absent")
		return
	}

	got, err := s.GetCache(cacheProbeKey)
	switch {
	case err != nil:
		fmt.Println("PROBE error")
	case got == nil:
		fmt.Println("PROBE absent")
	case len(got) != cacheProbeLen:
		fmt.Printf("PROBE torn %d\n", len(got))
	case !bytes.Equal(got, cacheProbeValue()):
		fmt.Println("PROBE corrupt")
	default:
		fmt.Println("PROBE whole")
	}
}

// TestSetCacheIsAtomicAcrossProcesses asserts the contract every other go-pob process on the box depends on.
func TestSetCacheIsAtomicAcrossProcesses(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("no subprocesses under js/wasm")
	}

	dir := useTempCacheDir(t)

	value := cacheProbeValue()
	s := Get()
	if err := s.SetCache(cacheProbeKey, value); err != nil {
		t.Fatalf("failed seeding cache: %v", err)
	}

	stop := make(chan struct{})
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for {
			select {
			case <-stop:
				return
			default:
			}

			InitializeStorage(nil, nil, nil, nil, nil, nil)
			_ = Get().SetCache(cacheProbeKey, value)
		}
	}()

	seen := make(map[string]int)
	for i := 0; i < cacheProbeChildren; i++ {
		//nolint:gosec // os.Args[0] is this test binary
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCacheProbeChild$", "-test.count=1")
		cmd.Env = append(os.Environ(), cacheProbeChildEnv+"=1", CacheDirEnv+"="+dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("reader process %d failed: %v\n%s", i, err, out)
		}

		for _, line := range strings.Split(string(out), "\n") {
			if !strings.HasPrefix(line, "PROBE ") {
				continue
			}

			result := strings.Fields(line)[1]
			seen[result]++

			switch result {
			case "torn", "corrupt", "error":
				t.Errorf("reader process %d observed a %s value: %s", i, result, line)
			case "hang":
				t.Errorf("reader process %d blocked forever on a key that exists", i)
			}
		}
	}

	close(stop)
	<-writerDone

	t.Logf("observations over %d reader processes: %v", cacheProbeChildren, seen)

	if seen["whole"] == 0 {
		t.Fatalf("no reader process ever observed the cached value; the probe proves nothing: %v", seen)
	}
}
