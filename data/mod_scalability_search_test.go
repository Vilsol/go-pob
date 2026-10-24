package data

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
)

func referenceFindScalableLine(line string, values []string) (string, []string, bool) {
	replaceNth := func(input, replacement string, n int) string {
		count := 0
		return replaceLimit(hashRe, input, -1, func([]string) string {
			count++
			if count == n {
				return replacement
			}
			return "#"
		})
	}

	var check func(start, numSubstitutions int, indices []int) (string, []string, bool)
	check = func(start, numSubstitutions int, indices []int) (string, []string, bool) {
		if len(indices) == numSubstitutions {
			modified := line
			substituted := 0
			for _, idx := range indices {
				modified = replaceNth(modified, values[idx], idx+1-substituted)
				substituted++
			}
			if _, ok := raw.ModScalability[scalabilityKey(modified)]; !ok {
				return "", nil, false
			}
			used := map[int]bool{}
			for _, idx := range indices {
				used[idx] = true
			}
			var remaining []string
			for i, v := range values {
				if !used[i] {
					remaining = append(remaining, v)
				}
			}
			return modified, remaining, true
		}
		for j := start; j < len(values); j++ {
			next := make([]int, len(indices), len(indices)+1)
			copy(next, indices)
			if m, r, ok := check(j+1, numSubstitutions, append(next, j)); ok {
				return m, r, true
			}
		}
		return "", nil, false
	}

	for i := len(values); i >= 1; i-- {
		if m, r, ok := check(0, i, nil); ok {
			return m, r, true
		}
	}
	if _, ok := raw.ModScalability[scalabilityKey(line)]; ok {
		return line, values, true
	}
	return "", nil, false
}

// TestFindScalableLineMatchesTheEnumeration grades the shipped search against the literal port.
func TestFindScalableLineMatchesTheEnumeration(t *testing.T) {
	const maxValues = 9

	f, err := os.Open(filepath.Join("..", "testdata", "corpus-"+raw.LatestVersion))
	if err != nil {
		t.Skipf("corpus is absent: %v", err)
	}
	names, err := f.Readdirnames(-1)
	_ = f.Close()
	if err != nil {
		t.Fatalf("corpus is unreadable: %v", err)
	}
	if len(names) < 100 {
		t.Fatalf("REFUSING: the corpus directory holds %d entries; this test would compare almost "+
			"nothing and report agreement", len(names))
	}

	stride := 5
	if os.Getenv("CORPUS") == "true" {
		stride = 1
	}
	slices.Sort(names)

	bodies := map[string]bool{}
	for i := 0; i < len(names); i += stride {
		name := names[i]
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		fh, err := os.Open(filepath.Join("..", "testdata", "corpus-"+raw.LatestVersion, name))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		inItem := false
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			switch {
			case strings.HasPrefix(line, "<Item "), strings.HasPrefix(line, "<Item>"):
				inItem = true
				continue
			case strings.HasPrefix(line, "</Item>"):
				inItem = false
				continue
			}
			if !inItem || line == "" || strings.HasPrefix(line, "<") {
				continue
			}
			bodies[stripBraceTags(line)] = true
		}
		_ = fh.Close()
	}
	if len(bodies) < 5000 {
		t.Fatalf("REFUSING: only %d distinct bodies scanned; the scan stopped matching and this "+
			"test would report agreement it did not measure", len(bodies))
	}

	compared, skipped, mismatches := 0, 0, 0
	var skipSample []string
	for body := range bodies {
		for _, rangeVal := range []float64{0, 0.5, 1} {
			stripped, values := stripLineForScalability(body, rangeVal)
			if len(values) > maxValues {
				skipped++
				if len(skipSample) < 6 && rangeVal == 0.5 {
					skipSample = append(skipSample, body)
				}
				continue
			}
			compared++
			wantLine, wantRemaining, wantOK := referenceFindScalableLine(stripped, values)
			gotLine, gotRemaining, gotOK := findScalableLine(stripped, values)
			if gotOK != wantOK || gotLine != wantLine ||
				strings.Join(gotRemaining, "\x00") != strings.Join(wantRemaining, "\x00") {
				mismatches++
				if mismatches <= 5 {
					t.Errorf("findScalableLine(%q, %v) = (%q, %v, %v), enumeration says (%q, %v, %v)",
						stripped, values, gotLine, gotRemaining, gotOK, wantLine, wantRemaining, wantOK)
				}
			}
		}
	}
	if compared < 10000 {
		t.Errorf("only %d comparisons ran; the sweep is not covering the corpus", compared)
	}
	t.Logf("%d comparisons, %d mismatches, %d line/range pairs skipped for having more than %d "+
		"values; a sample of what was skipped: %q", compared, mismatches, skipped, maxValues, skipSample)
}

func stripBraceTags(line string) string {
	for strings.HasPrefix(line, "{") {
		i := strings.IndexByte(line, '}')
		if i < 0 {
			break
		}
		line = line[i+1:]
	}
	return line
}
