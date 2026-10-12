package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestOfferingRowsCarryOnlyTheSkillNameTag: ModParser.lua:3392 carries exactly ONE tag — the SkillName list.
func TestOfferingRowsCarryOnlyTheSkillNameTag(t *testing.T) {
	assertOfferingRow(t, "Offering Skills Triggered this way also affect you",
		[]string{"Bone Offering", "Flesh Offering", "Spirit Offering"})
	assertOfferingRow(t, "Your Offering Skills also affect you",
		[]string{"Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"})
}

func assertOfferingRow(t *testing.T, line string, wantNames []string) {
	t.Helper()

	mods, extra := parseMod(line, 1)
	if extra != "" {
		t.Errorf("%q: did not parse completely, extra %q", line, extra)
		return
	}
	if len(mods) != 1 {
		t.Errorf("%q: want exactly one mod, got %d", line, len(mods))
		return
	}

	got := mods[0]
	if got.Name() != "ExtraSkillMod" {
		t.Errorf("%q: emitted %q, want ExtraSkillMod", line, got.Name())
		return
	}

	tags := got.Tags()
	if len(tags) != 1 {
		var kinds []string
		for _, tg := range tags {
			kinds = append(kinds, string(tg.Type()))
		}
		t.Errorf("%q: carries %d tags %v; ModParser.lua carries exactly one (SkillName). An "+
			"EXTRA tag is an over-restriction that gates the mod out while every producer census "+
			"still reports it present", line, len(tags), kinds)
		return
	}
	name, ok := tags[0].(*mod.SkillNameTag)
	if !ok {
		t.Errorf("%q: the one tag is %T, want *mod.SkillNameTag", line, tags[0])
		return
	}
	if strings.Join(name.SkillNameList, "|") != strings.Join(wantNames, "|") {
		t.Errorf("%q: skill names are %v, want %v", line, name.SkillNameList, wantNames)
	}
}

// TestOfferingRowsTagCountMatchesThePin: The premise is read from the PIN rather than restated.
func TestOfferingRowsTagCountMatchesThePin(t *testing.T) {
	src := pinnedPoBSrcOrSkip(t)
	body, err := os.ReadFile(filepath.Join(src, "Modules", "ModParser.lua"))
	if err != nil {
		t.Skipf("NOT-RUN: cannot read the pinned ModParser.lua: %v", err)
	}

	seen := 0
	for _, raw := range strings.Split(string(body), "\n") {
		if !strings.Contains(raw, "offering skills") || !strings.Contains(raw, "also affect you") {
			continue
		}
		seen++
		if n := strings.Count(raw, `{ type = "`); n != 1 {
			t.Errorf("pinned row %q carries %d tags, this port mirrors 1", strings.TrimSpace(raw), n)
		}
		if !strings.Contains(raw, `type = "SkillName"`) {
			t.Errorf("pinned row %q no longer tags on SkillName", strings.TrimSpace(raw))
		}
		if strings.Contains(raw, "SocketedIn") {
			t.Errorf("pinned row %q now carries SocketedIn; this port dropped it", strings.TrimSpace(raw))
		}
	}
	if seen != 2 {
		t.Fatalf("REFUSING: found %d `offering skills also affect you` rows in the pinned "+
			"ModParser.lua, expected 2 (:2923 and :3392). An empty or short scan here is the "+
			"CHECKER failing, not the pin", seen)
	}
}
