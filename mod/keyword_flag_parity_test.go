package mod

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// TestKeywordFlagsMatchThePinnedPoB pins our KeywordFlag values against PoB's own Data/Global.lua, by PARSING IT — not by transcription.
func TestKeywordFlagsMatchThePinnedPoB(t *testing.T) {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			pobDir = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
		}
	}
	globalLua := filepath.Join(pobDir, "src", "Data", "Global.lua")
	raw, err := os.ReadFile(globalLua)
	if err != nil {
		t.Skipf("pinned PoB not readable at %s (%v) — set POB_DIR", globalLua, err)
	}

	re := regexp.MustCompile(`(?m)^KeywordFlag\.(\w+)\s*=\s*(0x[0-9a-fA-F]+)`)
	matches := re.FindAllStringSubmatch(string(raw), -1)
	if len(matches) < 20 {
		t.Fatalf("REFUSING: parsed only %d KeywordFlag rows from %s — the file's shape changed and "+
			"this test would silently check almost nothing", len(matches), globalLua)
	}

	ours := KeywordFlagsByName

	var checked int
	for _, m := range matches {
		name, hex := m[1], m[2]
		want, err := strconv.ParseInt(hex[2:], 16, 64)
		if err != nil {
			t.Fatalf("cannot parse %s = %s: %v", name, hex, err)
		}
		got, known := ours[name]
		if !known {
			t.Errorf("PoB declares KeywordFlag.%s = %s and mod/common.go does not model it.\n"+
				"⚠️ Adding it means THREE places, not one: the constant here, plus the KW_BITS "+
				"tables in oracle/skill_stat_map_emit.py and oracle/skill_base_mods_emit.py, "+
				"which decode PoB's raw numeric keywordFlags into these names.", name, hex)
			continue
		}
		checked++
		if int64(got) != want {
			t.Errorf("KeywordFlag%s = 0x%08x, PoB says %s.\n"+
				"⚠️ Do NOT 'fix' this by editing only this package: oracle/skill_stat_map_emit.py "+
				"and oracle/skill_base_mods_emit.py each carry a KW_BITS table that decodes PoB's "+
				"RAW numeric keywordFlags into these names, and they must agree. A half-fix leaves "+
				"the generated data internally inconsistent, which is worse than the offset.",
				name, int64(got), hex)
		}
	}
	if checked < len(matches) {
		t.Errorf("REFUSING: %d of the pin's %d flags were compared; the rest are unmodelled and "+
			"this test is checking less than it appears to", checked, len(matches))
	}
}
