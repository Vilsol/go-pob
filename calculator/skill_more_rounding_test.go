package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestSkillModListMoreRoundsPerName pins ModList.lua:118-150's per-name `round(modResult, 2)` at the level that matters: a skill's own mod list.
func TestSkillModListMoreRoundsPerName(t *testing.T) {
	requireGameData(t, "3.29")
	const build = "0065-chieftain-tectonic-slam.xml"
	const pobMore = 0.77

	corpus := corpusDirForLoadedData(t)
	xml, err := os.ReadFile(filepath.Join(corpus, build))
	if err != nil {
		t.Fatalf("REFUSING: cannot read fixture %s: %v", build, err)
	}
	b, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatalf("REFUSING: cannot parse fixture %s: %v", build, err)
	}
	env := (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
	if env == nil || env.Player == nil || env.Player.MainSkill == nil {
		t.Fatalf("REFUSING: fixture %s built no main skill", build)
	}
	as := env.Player.MainSkill

	raw := 1.0
	n := 0
	for _, r := range as.SkillModList.Tabulate(mod.TypeMore, as.SkillCfg, "Speed") {
		raw *= 1 + r.Value/100
		n++
	}
	if n < 2 {
		t.Fatalf("REFUSING: %s now has %d Speed MORE mods; the per-name product needs at least "+
			"two to land off a 2-decimal step, so a pass would say nothing", build, n)
	}
	if math.Abs(raw-pobMore) < 1e-9 {
		t.Fatalf("REFUSING: the raw product is already %v, so the rounding has no work on this "+
			"fixture and it can no longer fail. Pick another build.", raw)
	}

	got := as.SkillModList.More(as.SkillCfg, "Speed")
	if math.Abs(got-pobMore) > 1e-9 {
		t.Errorf("skillModList.More(cfg, \"Speed\") = %v, PoB reports %v "+
			"(raw product of the %d mods is %v; ModList.lua:143 folds it in as round(x, 2))",
			got, pobMore, n, raw)
	}
}
