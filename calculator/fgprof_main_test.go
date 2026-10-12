package calculator

import (
	"fmt"
	"os"
	"testing"

	"github.com/felixge/fgprof"
)

// TestMain optionally wraps the whole package run in an fgprof profile.
func TestMain(m *testing.M) {
	path := os.Getenv("FGPROF")
	if path == "" {
		os.Exit(m.Run())
	}

	f, err := os.Create(path) //nolint:gosec // an operator-supplied profile path
	if err != nil {
		panic("REFUSING: cannot create FGPROF=" + path + ": " + err.Error())
	}

	stop := fgprof.Start(f, fgprof.FormatPprof)
	code := m.Run()

	if err := stopProfile(stop, f, path); err != nil {
		os.Stderr.WriteString("REFUSING: no fgprof profile written: " + err.Error() + "\n")
		if code == 0 {
			code = 3
		}
	}
	os.Exit(code)
}

func stopProfile(stop func() error, f *os.File, path string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("fgprof panicked (%v) — the run was too short to collect a sample; "+
				"profile it over the full package, not a -run selector", r)
		}
		_ = f.Close()
		if err != nil {
			_ = os.Remove(path)
		}
	}()

	if err := stop(); err != nil {
		return fmt.Errorf("fgprof stop: %w", err)
	}
	return nil
}
