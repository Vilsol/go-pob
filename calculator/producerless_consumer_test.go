package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProducerlessConsumersAreStillProducerless is a TRIPWIRE, not a coverage test.
func TestProducerlessConsumersAreStillProducerless(t *testing.T) {
	pending := map[string]string{}

	roots := []string{".", "../data", "../mod", "../moddb"}
	var candidates []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") || strings.Contains(path, "calc_offence.go") {
				return nil
			}
			candidates = append(candidates, path)
			return nil
		})
		if err != nil {
			t.Fatalf("REFUSING: cannot walk %s: %v", root, err)
		}
	}
	if len(candidates) == 0 {
		t.Fatal("REFUSING: the sweep found no non-test .go files — empty must mean " +
			"'I looked and found nothing', never 'I did not look'")
	}

	found := map[string][]string{}
	for _, path := range candidates {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("REFUSING: cannot read %s: %v", path, err)
		}
		for name := range pending {
			if strings.Contains(string(raw), `"`+name+`"`) {
				found[name] = append(found[name], path)
			}
		}
	}

	for name, where := range pending {
		if sites := found[name]; len(sites) > 0 {
			t.Errorf("%s now has a producer (%s) — the other half of a both-ends-absent pair "+
				"landed. The consumer in calc_offence.go has been correct and inert since it was "+
				"written; re-run the authoritative gate and record what the PAIR is worth, then "+
				"delete this row. Expected home was: %s", name, strings.Join(sites, ", "), where)
		}
	}
}
