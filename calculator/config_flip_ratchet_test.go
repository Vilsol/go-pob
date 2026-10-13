package calculator

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	configFlipAgreeFloor    = 766
	configFlipDefectCeiling = 0
)

func TestConfigFlipRatchet(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "config-flip", "flip.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	header := strings.Split(lines[0], "\t")
	col := slices.Index(header, "class")
	if col < 0 || len(lines) < 2 {
		t.Fatalf("REFUSING: flip.tsv has no class column or no rows (%d lines)", len(lines))
	}
	counts := map[configFlipClass]int{}
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) != len(header) {
			t.Fatalf("REFUSING: malformed row %q", line)
		}
		counts[configFlipClass(fields[col])]++
	}
	agree, defects := counts[flipAgreeMoves], counts[flipDisagree]+counts[flipOursOnly]
	if agree < configFlipAgreeFloor {
		t.Errorf("AGREE-MOVES %d, below the floor %d", agree, configFlipAgreeFloor)
	}
	if defects > configFlipDefectCeiling {
		t.Errorf("DISAGREE + OURS-ONLY %d, above the ceiling %d", defects, configFlipDefectCeiling)
	}
	if agree > configFlipAgreeFloor || defects < configFlipDefectCeiling {
		t.Logf("raise configFlipAgreeFloor to %d / lower configFlipDefectCeiling to %d", agree, defects)
	}
}
