package calculator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestItemConditionAllSlotsAlwaysExcludesSelf pins the PREMISE that lets `Actor.EquippedSlots` drop jewels and grafts unconditionally.
//
//	if (not tag.allSlots or tag.allSlots and (item.type ~= "Jewel" and item.type ~= "Graft"))
//	   and slot ~= itemSlot or not tag.excludeSelf then
//	( A and slot ~= itemSlot ) or ( not tag.excludeSelf )
func TestItemConditionAllSlotsAlwaysExcludesSelf(t *testing.T) {
	t.Run("pinned ModParser.lua", func(t *testing.T) {
		pobRoot := os.Getenv("POB_DIR")
		if pobRoot == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
			}
			pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
		}
		luaPath := filepath.Join(pobRoot, "src", "Modules", "ModParser.lua")
		luaSrc, err := os.ReadFile(luaPath)
		if err != nil {
			t.Skipf("NOT-RUN: no pinned ModParser.lua at %s (set POB_DIR). This has NOT told you the "+
				"premise holds — it has told you it could not look.", luaPath)
		}

		found := 0
		for i, line := range strings.Split(string(luaSrc), "\n") {
			if !strings.Contains(line, "allSlots") || !strings.Contains(line, "ItemCondition") {
				continue
			}
			found++
			if !strings.Contains(line, "excludeSelf") {
				t.Errorf("ModParser.lua:%d sets allSlots on an ItemCondition WITHOUT excludeSelf. "+
					"ModStore.lua:673 then does NOT skip jewels and grafts there, and "+
					"Actor.EquippedSlots drops them anyway — the divergence is now live:\n\t%s",
					i+1, strings.TrimSpace(line))
			}
			if !strings.Contains(line, "searchCond") {
				t.Errorf("ModParser.lua:%d sets allSlots on an ItemCondition WITHOUT searchCond. "+
					"ModStore.lua:678-702's rarity/corrupted/name loops carry no Jewel/Graft test, "+
					"so EquippedSlots dropping them is now a live divergence:\n\t%s",
					i+1, strings.TrimSpace(line))
			}
		}
		if found == 0 {
			t.Fatalf("REFUSING: found no allSlots ItemCondition rows in the pinned ModParser.lua. "+
				"The entry shape has moved and this test is vacuous (%s)", luaPath)
		}
	})

	t.Run("our mod_parser.go", func(t *testing.T) {
		src, err := os.ReadFile("mod_parser.go")
		if err != nil {
			t.Fatalf("cannot read mod_parser.go: %v", err)
		}
		chain := regexp.MustCompile(`(?m)^.*WithAllSlots\(\).*$`)
		hits := chain.FindAllString(string(src), -1)
		if len(hits) == 0 {
			t.Fatalf("REFUSING: no WithAllSlots() call sites in mod_parser.go; the sole allSlots " +
				"producer has moved or been dropped and this test is vacuous")
		}
		for _, h := range hits {
			if strings.Contains(strings.TrimSpace(h), "//") &&
				strings.Index(h, "//") < strings.Index(h, "WithAllSlots") {
				continue
			}
			if !strings.Contains(h, "WithExcludeSelf()") {
				t.Errorf("a WithAllSlots() producer does not chain WithExcludeSelf(); "+
					"Actor.EquippedSlots' unconditional Jewel/Graft drop is now a live divergence "+
					"(ModStore.lua:673 short-circuits the Jewel/Graft test when excludeSelf is "+
					"false):\n\t%s", strings.TrimSpace(h))
			}
			if !strings.Contains(h, ".Search(") {
				t.Errorf("a WithAllSlots() producer does not set a Search condition; "+
					"ModStore.lua:678-702's rarity/corrupted/name loops carry no Jewel/Graft test, "+
					"so EquippedSlots dropping them is now a live divergence:\n\t%s",
					strings.TrimSpace(h))
			}
		}
	})
}
