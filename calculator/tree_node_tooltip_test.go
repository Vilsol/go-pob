package calculator

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

var grantsSkillRe = regexp.MustCompile(`^Grants Level (\d+) (.+) Skill$`)

// TestNodeSkillTooltips: every tree node whose line grants a skill.
func TestNodeSkillTooltips(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0005-occultist-vaal-double-strike.xml")
	graded := 0
	for key, n := range specTree().Nodes {
		for _, line := range n.Stats {
			m := grantsSkillRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			id, _ := strconv.ParseInt(key, 10, 64)
			known := false
			for _, v := range utils.CastSlice[mod.ExtraSkill](asModList(ParseMod(line, false).ModList).List(nil, "ExtraSkill")) {
				known = known || newGemTooltipEffect(v.SkillID) != nil
			}
			if !known {
				continue
			}
			tips := NodeSkillTooltips(build, id, ItemTooltipContext{})
			if len(tips) == 0 || len(tips[0]) == 0 {
				t.Errorf("node %s %q: no gem tooltip", key, line)
				continue
			}
			graded++
			head := tips[0][0].Text
			if !strings.Contains(head, m[2]) {
				t.Errorf("node %s: the tooltip opens %q, not %q", key, head, m[2])
			}
			found := false
			for _, l := range tips[0] {
				if strings.Contains(l.Text, "Level: ") && strings.Contains(l.Text, m[1]) {
					found = true
				}
			}
			if !found {
				t.Errorf("node %s: no level %s line in %v", key, m[1], tips[0])
			}
		}
	}
	if graded < 3 {
		t.Fatalf("graded only %d skill-granting nodes", graded)
	}
}
