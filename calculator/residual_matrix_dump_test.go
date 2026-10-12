package calculator

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestResidualMatrixDump writes the FULL (build, stat) mismatch matrix in one corpus pass.
func TestResidualMatrixDump(t *testing.T) {
	out := os.Getenv("MATRIX_OUT")
	if out == "" {
		t.Skip("set MATRIX_OUT=<file> to dump the residual matrix")
	}
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, names := corpusNamesForLoadedData(t)

	type row struct {
		build, stat, class string
		pob, ours          float64
	}
	var mu sync.Mutex
	rows := make([]row, 0, 400000)
	built := 0

	runConcurrently(len(names), 12, func(i int, cache *EnvironmentCache) {
		name := names[i]
		rec, ok := oracle[name]
		if !ok {
			return
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			return
		}
		local := make([]row, 0, 128)
		for _, pair := range oracleTablePairs(rec, env) {
			for k, want := range pair.exp {
				got := pair.got[k]
				if math.IsNaN(got) || math.IsInf(got, 0) {
					local = append(local, row{name, pair.prefix + k, "nan", want, got})
					continue
				}
				if math.Abs(got-want) <= oracleDumpTolerance*math.Abs(want) {
					continue
				}
				class := "wrong"
				switch {
				case want != 0 && got == 0:
					class = "miss"
				case want == 0 && got != 0:
					class = "extra"
				}
				local = append(local, row{name, pair.prefix + k, class, want, got})
			}
		}
		mu.Lock()
		built++
		rows = append(rows, local...)
		mu.Unlock()
	})

	if built < len(names)/2 {
		t.Fatalf("REFUSING: built %d of %d builds; a matrix over half a corpus is not a matrix",
			built, len(names))
	}

	f, err := os.Create(out)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer func() { _ = f.Close() }()
	w := bufio.NewWriterSize(f, 1<<20)
	_, _ = fmt.Fprintln(w, "build\tstat\tclass\tpob\tours")
	for _, r := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%g\t%g\n", r.build, r.stat, r.class, r.pob, r.ours)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("wrote %s: %d mismatch rows over %d builds", out, len(rows), built)
}
